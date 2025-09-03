package interfaces

import (
	"context"
	"io"
	"net"
	"os"
	"time"

	"github.com/termbus/termbus/pkg/types"
	"golang.org/x/crypto/ssh"
)

// SessionManager 会话管理器接口
type SessionManager interface {
	CreateSession(hostConfig *types.SSHHostConfig) (*types.Session, error)
	ConnectSession(sessionID string) error
	DisconnectSession(sessionID string) error
	DeleteSession(sessionID string) error
	GetSession(sessionID string) (*types.Session, error)
	ListSessions() []*types.Session
	SetActiveSession(sessionID string) error
	GetActiveSession() (*types.Session, error)
	GetSSHClient(sessionID string) (*ssh.Client, error)
}

// SSHManager SSH连接管理器接口
type SSHManager interface {
	Connect(hostConfig *HostConfig, password string) (*ssh.Client, error)
	Disconnect(host, user string, port int) error
	GetHostConfig(hostAlias string) (*HostConfig, error)
	ScanSSHConfigs() ([]HostConfig, error)
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

// SFTPManager SFTP管理器接口
type SFTPManager interface {
	List(sessionID, path string) ([]types.FileInfo, error)
	Download(sessionID, remotePath, localPath string, progress chan float64) error
	Upload(sessionID, localPath, remotePath string, progress chan float64) error
	Delete(sessionID, path string) error
	Rename(sessionID, oldPath, newPath string) error
	Mkdir(sessionID, path string) error
	ReadFile(sessionID, path string) (string, error)
	WriteFile(sessionID, path, content string) error
}

// TunnelManager 隧道管理器接口
type TunnelManager interface {
	CreateTunnel(sessionID string, tunnel *types.ForwardTunnel) error
	StartTunnel(tunnelID string) error
	StopTunnel(tunnelID string) error
	DeleteTunnel(tunnelID string) error
	ListTunnels(sessionID string) ([]*types.ForwardTunnel, error)
	GetTunnel(tunnelID string) (*types.ForwardTunnel, error)
}

// CommandManager 命令管理器接口
type CommandManager interface {
	ExecuteCommand(sessionID, command string) (string, error)
	BatchExecute(sessionIDs []string, command string) map[string]string
}

// PluginManager 插件管理器接口
type PluginManager interface {
	LoadPlugin(pluginPath string) error
	UnloadPlugin(pluginID string) error
	ListPlugins() ([]PluginInfo, error)
	ExecuteCommand(pluginID, command string, args map[string]interface{}) (string, error)
}

// PluginInfo 插件信息
type PluginInfo struct {
	ID          string
	Name        string
	Version     string
	Description string
	Enabled     bool
}

// AgentRuntime AI Agent运行时接口
type AgentRuntime interface {
	Execute(task string) (*types.AgentPlan, error)
	ConfirmPlan(plan *types.AgentPlan) error
	GetTools() []types.Tool
}

// EventBus 事件总线接口
type EventBus interface {
	Subscribe(topic string, handler interface{})
	Unsubscribe(topic string, handler interface{})
	Publish(topic string, args ...interface{})
}

// TerminalEmulator 终端模拟器接口
type TerminalEmulator interface {
	Write(p []byte) (n int, err error)
	Read(p []byte) (n int, err error)
	Resize(rows, cols uint16) error
	Close() error
}

// ConfigManager 配置管理器接口
type ConfigManager interface {
	Reload() error
	Get() interface{}
	Watch(ch chan<- interface{})
}

// Logger 日志接口
type Logger interface {
	Info(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Debug(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	With(fields ...Field) Logger
}

// Field 日志字段
type Field struct {
	Key   string
	Value interface{}
}

// FileOperation 文件操作接口
type FileOperation interface {
	Upload(localPath, remotePath string) error
	Download(remotePath, localPath string) error
	Delete(path string) error
	Mkdir(path string) error
}

// RemoteExecutor 远程执行器接口
type RemoteExecutor interface {
	Execute(ctx context.Context, cmd string) (string, error)
	ExecuteWithStdin(ctx context.Context, cmd string, stdin io.Reader) (string, error)
	ExecuteWithPTY(ctx context.Context, cmd string, stdin io.Reader, stdout, stderr io.Writer) error
}

// ConnectionPool 连接池接口
type ConnectionPool interface {
	GetConnection(key string) (*ssh.Client, error)
	SetConnection(key string, client *ssh.Client)
	ReleaseConnection(key string)
}

// WindowManager 窗口管理器接口
type WindowManager interface {
	CreateWindow(sessionID string) (*types.Window, error)
	CloseWindow(windowID string) error
	SplitPane(windowID, paneID string, direction string) error
	ClosePane(windowID, paneID string) error
	SetActivePane(windowID, paneID string) error
}

// SessionStore 会话存储接口
type SessionStore interface {
	Save(session *types.Session) error
	Load(sessionID string) (*types.Session, error)
	Delete(sessionID string) error
	List() ([]*types.Session, error)
}

// Auditor 审计接口
type Auditor interface {
	Audit(event, user, host, details string)
	AuditWithFields(fields map[string]interface{})
}

// HostKeyVerifier 主机密钥校验接口
type HostKeyVerifier interface {
	Verify(hostname string, remote net.Addr, key ssh.PublicKey) error
	Add(hostname string, key ssh.PublicKey) error
}

// AuthProvider 认证提供者接口
type AuthProvider interface {
	GetAuthMethods() ([]ssh.AuthMethod, error)
	Name() string
}

// SSHClient SSH客户端接口
type SSHClient interface {
	Connect(hostConfig *HostConfig, password string) (*ssh.Client, error)
	Dial(network, addr string) (net.Conn, error)
	NewSession() (*ssh.Session, error)
	Close() error
}

// SFTPClient SFTP客户端接口
type SFTPClient interface {
	Stat(path string) (os.FileInfo, error)
	ReadDir(path string) ([]os.FileInfo, error)
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte) error
	Create(path string) (*File, error)
	Remove(path string) error
	Rename(oldpath, newpath string) error
	MkdirAll(path string) error
	OpenFile(path string, flag int) (io.ReadWriteCloser, error)
}

// File 文件接口
type File interface {
	io.Reader
	io.Writer
	io.Closer
	Stat() (os.FileInfo, error)
}

type FileInfo interface {
	Name() string
	Size() int64
	Mode() os.FileMode
	ModTime() time.Time
	IsDir() bool
	Sys() interface{}
}
