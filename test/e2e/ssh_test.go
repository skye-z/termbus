package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/session"
	"github.com/termbus/termbus/internal/ssh"
	"github.com/termbus/termbus/pkg/types"
)

func getProjectRoot() string {
	wd, _ := os.Getwd()
	for wd != "/" {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		wd = filepath.Dir(wd)
	}
	return ""
}

type TestConfig struct {
	Hosts     HostsConfig     `mapstructure:"hosts"`
	TestPaths TestPathsConfig `mapstructure:"test_paths"`
	Timeouts  TimeoutsConfig  `mapstructure:"timeouts"`
}

type HostsConfig struct {
	SSHServer SSHServerConfig `mapstructure:"ssh_server"`
}

type SSHServerConfig struct {
	Hostname string `mapstructure:"hostname"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type TestPathsConfig struct {
	RemoteTmp  string `mapstructure:"remote_tmp"`
	RemoteHome string `mapstructure:"remote_home"`
}

type TimeoutsConfig struct {
	Connect  int `mapstructure:"connect"`
	Command  int `mapstructure:"command"`
	Transfer int `mapstructure:"transfer"`
}

func loadTestConfig(t *testing.T) *TestConfig {
	configPath := filepath.Join(getProjectRoot(), "test", "e2e", "config.yaml")

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	err := v.ReadInConfig()
	require.NoError(t, err, "Failed to read E2E config")

	var config TestConfig
	err = v.Unmarshal(&config)
	require.NoError(t, err, "Failed to unmarshal E2E config")

	return &config
}

func TestE2E_SSHConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	config := loadTestConfig(t)
	t.Logf("INPUT: Connecting to SSH server %s:%d as %s", config.Hosts.SSHServer.Hostname, config.Hosts.SSHServer.Port, config.Hosts.SSHServer.Username)

	eventBus := eventbus.New()

	sshCfg := &ssh.SSHConfig{
		ConfigPath:        "",
		KnownHostsPath:    "",
		DefaultTimeout:    config.Timeouts.Connect,
		KeepaliveEnabled:  false,
		KeepaliveInterval: 60,
	}
	sshManager := ssh.NewSSHManager(sshCfg, eventBus)

	hostConfig := &ssh.HostConfig{
		Host:     config.Hosts.SSHServer.Hostname,
		HostName: config.Hosts.SSHServer.Hostname,
		User:     config.Hosts.SSHServer.Username,
		Port:     config.Hosts.SSHServer.Port,
	}

	client, err := sshManager.Connect(hostConfig, config.Hosts.SSHServer.Password)
	require.NoError(t, err, "Failed to connect to SSH server")
	require.NotNil(t, client)
	t.Logf("OUTPUT: SSH client connected successfully")

	sshPool := session.NewSSHConnectionPool()
	hostKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)
	sshPool.SetConnection(hostKey, client)

	sessionMgr := session.New(eventBus, sshPool, nil)

	sessionHostConfig := &types.SSHHostConfig{
		Host:     config.Hosts.SSHServer.Hostname,
		HostName: config.Hosts.SSHServer.Hostname,
		User:     config.Hosts.SSHServer.Username,
		Port:     config.Hosts.SSHServer.Port,
	}

	s, err := sessionMgr.CreateSession(sessionHostConfig)
	require.NoError(t, err)

	err = sessionMgr.ConnectSession(s.ID)
	require.NoError(t, err, "Failed to connect session")

	sess, _ := sessionMgr.GetSession(s.ID)
	assert.Equal(t, types.SessionStateConnected, sess.State)

	sessionMgr.DisconnectSession(s.ID)
	sessionMgr.DeleteSession(s.ID)
	sshManager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)
}

func TestE2E_SSHCommandExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	config := loadTestConfig(t)
	t.Logf("INPUT: Testing SSH command execution on %s", config.Hosts.SSHServer.Hostname)

	eventBus := eventbus.New()

	sshCfg := &ssh.SSHConfig{
		ConfigPath:        "",
		KnownHostsPath:    "",
		DefaultTimeout:    config.Timeouts.Connect,
		KeepaliveEnabled:  false,
		KeepaliveInterval: 60,
	}
	sshManager := ssh.NewSSHManager(sshCfg, eventBus)

	hostConfig := &ssh.HostConfig{
		Host:     config.Hosts.SSHServer.Hostname,
		HostName: config.Hosts.SSHServer.Hostname,
		User:     config.Hosts.SSHServer.Username,
		Port:     config.Hosts.SSHServer.Port,
	}

	client, err := sshManager.Connect(hostConfig, config.Hosts.SSHServer.Password)
	require.NoError(t, err)
	require.NotNil(t, client)

	sshPool := session.NewSSHConnectionPool()
	hostKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)
	sshPool.SetConnection(hostKey, client)

	sessionMgr := session.New(eventBus, sshPool, nil)

	sessionHostConfig := &types.SSHHostConfig{
		Host:     config.Hosts.SSHServer.Hostname,
		HostName: config.Hosts.SSHServer.Hostname,
		User:     config.Hosts.SSHServer.Username,
		Port:     config.Hosts.SSHServer.Port,
	}

	s, err := sessionMgr.CreateSession(sessionHostConfig)
	require.NoError(t, err)
	defer sessionMgr.DeleteSession(s.ID)

	err = sessionMgr.ConnectSession(s.ID)
	require.NoError(t, err)

	tests := []struct {
		name    string
		command string
		check   func(t *testing.T, output string)
	}{
		{
			name:    "whoami",
			command: "whoami",
			check: func(t *testing.T, output string) {
				assert.Contains(t, output, config.Hosts.SSHServer.Username)
			},
		},
		{
			name:    "pwd",
			command: "pwd",
			check: func(t *testing.T, output string) {
				assert.NotEmpty(t, output)
			},
		},
		{
			name:    "uname",
			command: "uname -a",
			check: func(t *testing.T, output string) {
				assert.NotEmpty(t, output)
			},
		},
		{
			name:    "echo",
			command: "echo 'hello e2e test'",
			check: func(t *testing.T, output string) {
				assert.Contains(t, output, "hello e2e test")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("INPUT: Executing command '%s'", tt.command)

			session, err := client.NewSession()
			require.NoError(t, err)
			defer session.Close()

			output, err := session.CombinedOutput(tt.command)
			require.NoError(t, err)
			t.Logf("OUTPUT: %s", strings.TrimSpace(string(output)))
			tt.check(t, string(output))
		})
	}

	sessionMgr.DisconnectSession(s.ID)
	sshManager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)
}

func TestE2E_SSHConnectionPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	config := loadTestConfig(t)
	eventBus := eventbus.New()

	sshCfg := &ssh.SSHConfig{
		ConfigPath:        "",
		KnownHostsPath:    "",
		DefaultTimeout:    config.Timeouts.Connect,
		KeepaliveEnabled:  false,
		KeepaliveInterval: 60,
	}
	sshManager := ssh.NewSSHManager(sshCfg, eventBus)

	hostConfig := &ssh.HostConfig{
		Host:     config.Hosts.SSHServer.Hostname,
		HostName: config.Hosts.SSHServer.Hostname,
		User:     config.Hosts.SSHServer.Username,
		Port:     config.Hosts.SSHServer.Port,
	}

	client, err := sshManager.Connect(hostConfig, config.Hosts.SSHServer.Password)
	require.NoError(t, err)
	require.NotNil(t, client)

	sshPool := session.NewSSHConnectionPool()
	hostKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)
	sshPool.SetConnection(hostKey, client)

	sessionMgr := session.New(eventBus, sshPool, nil)

	sessionHostConfig := &types.SSHHostConfig{
		Host:     config.Hosts.SSHServer.Hostname,
		HostName: config.Hosts.SSHServer.Hostname,
		User:     config.Hosts.SSHServer.Username,
		Port:     config.Hosts.SSHServer.Port,
	}

	s1, err := sessionMgr.CreateSession(sessionHostConfig)
	require.NoError(t, err)
	err = sessionMgr.ConnectSession(s1.ID)
	require.NoError(t, err)

	s2, err := sessionMgr.CreateSession(sessionHostConfig)
	require.NoError(t, err)
	err = sessionMgr.ConnectSession(s2.ID)
	require.NoError(t, err)

	sess1, _ := sessionMgr.GetSession(s1.ID)
	sess2, _ := sessionMgr.GetSession(s2.ID)

	assert.Equal(t, types.SessionStateConnected, sess1.State)
	assert.Equal(t, types.SessionStateConnected, sess2.State)

	sessionMgr.DisconnectSession(s1.ID)
	sessionMgr.DisconnectSession(s2.ID)
	sessionMgr.DeleteSession(s1.ID)
	sessionMgr.DeleteSession(s2.ID)
	sshManager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)
}

func TestE2E_SSHConfigParsing(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		t.Skip("Home directory not found")
	}

	configPath := fmt.Sprintf("%s/.ssh/config", homeDir)
	_, err = os.Stat(configPath)
	if os.IsNotExist(err) {
		t.Skip("SSH config file not found: " + configPath)
	}

	eventBus := eventbus.New()

	cfg := &ssh.SSHConfig{
		ConfigPath:        configPath,
		KnownHostsPath:    "",
		DefaultTimeout:    10,
		KeepaliveEnabled:  false,
		KeepaliveInterval: 60,
	}

	manager := ssh.NewSSHManager(cfg, eventBus)

	configs, err := manager.ScanSSHConfigs()
	assert.NoError(t, err)
	assert.NotEmpty(t, configs)
}
