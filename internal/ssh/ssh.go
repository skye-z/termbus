package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

var (
	ErrConnectionFailed     = errors.New("ssh connection failed")
	ErrAuthenticationFailed = errors.New("ssh authentication failed")
	ErrHostKeyVerification  = errors.New("ssh host key verification failed")
)

// SSHManager SSH连接管理器
type SSHManager struct {
	config       *SSHConfig
	hostKeyStore *HostKeyStore
	eventBus     EventBus
	mu           sync.RWMutex
	connections  map[string]*ssh.Client
	refs         map[string]int
}

// SSHConfig SSH配置
type SSHConfig struct {
	ConfigPath        string
	KnownHostsPath    string
	DefaultTimeout    int
	KeepaliveEnabled  bool
	KeepaliveInterval int
}

// EventBus 事件总线接口
type EventBus interface {
	Publish(topic string, args ...interface{})
}

// HostKeyStore 主机密钥存储
type HostKeyStore struct {
	knownHostsPath string
	strictMode     bool
}

// NewSSHManager 创建SSH管理器
func NewSSHManager(config *SSHConfig, eventBus EventBus) *SSHManager {
	return &SSHManager{
		config:       config,
		hostKeyStore: NewHostKeyStore(config.KnownHostsPath, true),
		eventBus:     eventBus,
		connections:  make(map[string]*ssh.Client),
		refs:         make(map[string]int),
	}
}

// NewHostKeyStore 创建主机密钥存储
func NewHostKeyStore(knownHostsPath string, strictMode bool) *HostKeyStore {
	return &HostKeyStore{
		knownHostsPath: knownHostsPath,
		strictMode:     strictMode,
	}
}

// GetHostConfig 从SSH Config文件获取主机配置
func (m *SSHManager) GetHostConfig(hostAlias string) (*HostConfig, error) {
	user := ssh_config.Get(hostAlias, "User")
	hostname := ssh_config.Get(hostAlias, "Hostname")
	portStr := ssh_config.Get(hostAlias, "Port")
	port := 22
	if portStr != "" {
		fmt.Sscanf(portStr, "%d", &port)
	}
	identityFile := ssh_config.Get(hostAlias, "IdentityFile")
	proxyJump := ssh_config.Get(hostAlias, "ProxyJump")
	proxyCommand := ssh_config.Get(hostAlias, "ProxyCommand")

	if hostname == "" {
		hostname = hostAlias
	}
	if user == "" {
		user = "root"
	}

	return &HostConfig{
		Host:         hostAlias,
		HostName:     hostname,
		User:         user,
		Port:         port,
		IdentityFile: identityFile,
		ProxyJump:    proxyJump,
		ProxyCommand: proxyCommand,
	}, nil
}

// HostConfig 主机配置
type HostConfig struct {
	Host         string
	HostName     string
	User         string
	Port         int
	IdentityFile string
	ProxyJump    string
	ProxyCommand string
	Password     string
}

// Connect 建立SSH连接
func (m *SSHManager) Connect(hostConfig *HostConfig, password string) (*ssh.Client, error) {
	connKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)

	m.mu.RLock()
	client, exists := m.connections[connKey]
	if exists {
		m.refs[connKey]++
		m.mu.RUnlock()
		return client, nil
	}
	m.mu.RUnlock()

	authMethods, err := m.getAuthMethods(hostConfig, password)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAuthenticationFailed, err)
	}

	config := &ssh.ClientConfig{
		User:            hostConfig.User,
		Auth:            authMethods,
		HostKeyCallback: m.hostKeyStore.HostKeyCallback(),
		Timeout:         time.Duration(m.config.DefaultTimeout) * time.Second,
	}

	var dialer net.Dialer
	conn, err := dialer.Dial("tcp", fmt.Sprintf("%s:%d", hostConfig.HostName, hostConfig.Port))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, "", config)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	client = ssh.NewClient(sshConn, chans, reqs)

	if m.config.KeepaliveEnabled {
		go m.keepalive(client)
	}

	m.mu.Lock()
	m.connections[connKey] = client
	m.refs[connKey] = 1
	m.mu.Unlock()

	return client, nil
}

// Disconnect 断开SSH连接
func (m *SSHManager) Disconnect(host, user string, port int) error {
	connKey := fmt.Sprintf("%s@%s:%d", user, host, port)

	m.mu.Lock()
	defer m.mu.Unlock()

	if refs, exists := m.refs[connKey]; exists {
		refs--
		if refs <= 0 {
			if client, ok := m.connections[connKey]; ok {
				client.Close()
				delete(m.connections, connKey)
				delete(m.refs, connKey)
			}
		} else {
			m.refs[connKey] = refs
		}
	}

	return nil
}

// getAuthMethods 获取认证方法
func (m *SSHManager) getAuthMethods(hostConfig *HostConfig, password string) ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	if agentMethod, err := m.getAgentAuthMethod(); err == nil {
		authMethods = append(authMethods, agentMethod)
	}

	if publicKeyMethod, err := m.getPublicKeyAuthMethod(hostConfig.IdentityFile, password); err == nil {
		authMethods = append(authMethods, publicKeyMethod)
	}

	if password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	if len(authMethods) == 0 {
		return nil, errors.New("no authentication methods available")
	}

	return authMethods, nil
}

// getAgentAuthMethod 获取SSH Agent认证
func (m *SSHManager) getAgentAuthMethod() (ssh.AuthMethod, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, errors.New("SSH_AUTH_SOCK not set")
	}

	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, err
	}

	agentClient := agent.NewClient(conn)
	signers, err := agentClient.Signers()
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeysCallback(func() ([]ssh.Signer, error) { return signers, nil }), nil
}

// getPublicKeyAuthMethod 获取公钥认证
func (m *SSHManager) getPublicKeyAuthMethod(identityFile, password string) (ssh.AuthMethod, error) {
	if identityFile == "" {
		homeDir, _ := os.UserHomeDir()
		identityFile = filepath.Join(homeDir, ".ssh", "id_rsa")
	}

	var signer ssh.Signer
	var err error

	keyBytes, err := os.ReadFile(identityFile)
	if err != nil {
		return nil, err
	}

	if password != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(password))
	} else {
		signer, err = ssh.ParsePrivateKey(keyBytes)
	}
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(signer), nil
}

// HostKeyCallback 主机密钥回调
func (h *HostKeyStore) HostKeyCallback() ssh.HostKeyCallback {
	if h.strictMode {
		callback, err := knownhosts.New(h.knownHostsPath)
		if err != nil {
			return ssh.InsecureIgnoreHostKey()
		}
		return callback
	}
	return ssh.InsecureIgnoreHostKey()
}

// keepalive 保活
func (m *SSHManager) keepalive(client *ssh.Client) {
	ticker := time.NewTicker(time.Duration(m.config.KeepaliveInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
		if err != nil {
			return
		}
	}
}

// ScanSSHConfigs 扫描所有SSH配置
func (m *SSHManager) ScanSSHConfigs() ([]HostConfig, error) {
	var configs []HostConfig

	file, err := os.ReadFile(m.config.ConfigPath)
	if err != nil {
		return nil, err
	}

	lines := bytes.Split(file, []byte("\n"))
	for _, line := range lines {
		lineStr := strings.TrimSpace(string(line))
		if lineStr == "" || strings.HasPrefix(lineStr, "#") {
			continue
		}

		if strings.HasPrefix(lineStr, "Host ") {
			host := strings.TrimSpace(strings.TrimPrefix(lineStr, "Host "))
			if strings.Contains(host, "*") {
				continue
			}
			config, err := m.GetHostConfig(host)
			if err != nil {
				continue
			}
			configs = append(configs, *config)
		}
	}

	return configs, nil
}

// ListIdentities 列出所有私钥文件
func ListIdentities() ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	var identities []string

	err = filepath.WalkDir(sshDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasPrefix(d.Name(), "id_") && !strings.HasSuffix(d.Name(), ".pub") {
			identities = append(identities, path)
		}
		return nil
	})

	return identities, err
}
