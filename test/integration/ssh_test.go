package integration

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/ssh"
)

func TestSSHManager_PasswordAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()

	cfg := &ssh.SSHConfig{
		ConfigPath:        "",
		KnownHostsPath:    "",
		DefaultTimeout:    10,
		KeepaliveEnabled:  false,
		KeepaliveInterval: 60,
	}

	manager := ssh.NewSSHManager(cfg, eventBus)

	host := "127.0.0.1"
	user := "testuser"
	port := 2222
	password := "testpass"

	hostConfig := &ssh.HostConfig{
		Host:     host,
		HostName: host,
		User:     user,
		Port:     port,
		Password: password,
	}

	client, err := manager.Connect(hostConfig, password)
	if err != nil {
		t.Skipf("Failed to connect: %v", err)
	}

	assert.NotNil(t, client)
	assert.NoError(t, client.Close())

	manager.Disconnect(host, user, port)
}

func TestSSHManager_PublicKeyAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()

	cfg := &ssh.SSHConfig{
		ConfigPath:        "",
		KnownHostsPath:    "",
		DefaultTimeout:    10,
		KeepaliveEnabled:  false,
		KeepaliveInterval: 60,
	}

	manager := ssh.NewSSHManager(cfg, eventBus)

	homeDir, _ := os.UserHomeDir()
	keyPath := fmt.Sprintf("%s/.ssh/id_rsa", homeDir)

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Skip("SSH key not found")
	}

	hostConfig := &ssh.HostConfig{
		Host:         "test-host",
		HostName:     "127.0.0.1",
		User:         "testuser",
		Port:         2222,
		IdentityFile: keyPath,
		Password:     "",
	}

	client, err := manager.Connect(hostConfig, "")
	if err != nil {
		t.Skipf("Failed to connect: %v", err)
	}

	assert.NotNil(t, client)
	assert.NoError(t, client.Close())
}

func TestSSHManager_ConnectionPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()

	cfg := &ssh.SSHConfig{
		ConfigPath:        "",
		KnownHostsPath:    "",
		DefaultTimeout:    10,
		KeepaliveEnabled:  false,
		KeepaliveInterval: 60,
	}

	manager := ssh.NewSSHManager(cfg, eventBus)

	hostConfig := &ssh.HostConfig{
		Host:     "127.0.0.1",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
		Password: "testpass",
	}

	client1, err := manager.Connect(hostConfig, "testpass")
	if err != nil {
		t.Skipf("Failed to connect: %v", err)
	}
	require.NotNil(t, client1)

	client2, err := manager.Connect(hostConfig, "testpass")
	require.NoError(t, err)
	assert.NotNil(t, client2)

	assert.Same(t, client1, client2, "Connection pool should reuse the same client")

	manager.Disconnect("127.0.0.1", "testuser", 2222)
}

func TestSSHManager_ConfigParsing(t *testing.T) {
	eventBus := eventbus.New()

	cfg := &ssh.SSHConfig{
		ConfigPath:        "",
		KnownHostsPath:    "",
		DefaultTimeout:    10,
		KeepaliveEnabled:  false,
		KeepaliveInterval: 60,
	}

	manager := ssh.NewSSHManager(cfg, eventBus)

	homeDir, _ := os.UserHomeDir()
	configPath := fmt.Sprintf("%s/.ssh/config", homeDir)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("SSH config file not found")
	}

	configs, err := manager.ScanSSHConfigs()
	assert.NoError(t, err)
	assert.NotEmpty(t, configs)
}

func TestSSHService_RequestConnect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	cfg := &ssh.SSHConfig{
		ConfigPath:        "",
		KnownHostsPath:    "",
		DefaultTimeout:    10,
		KeepaliveEnabled:  false,
		KeepaliveInterval: 60,
	}

	sshManager := ssh.NewSSHManager(cfg, eventbus.NewManager())

	service := ssh.NewSSHService(sshManager, nil, eventbus.NewManager())
	defer service.Stop()

	err := service.Start()
	assert.NoError(t, err)

	service.RequestConnect("test-session", "testpass")

	time.Sleep(1 * time.Second)
}

func TestSSHManager_InvalidCredentials(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()

	cfg := &ssh.SSHConfig{
		ConfigPath:        "",
		KnownHostsPath:    "",
		DefaultTimeout:    10,
		KeepaliveEnabled:  false,
		KeepaliveInterval: 60,
	}

	manager := ssh.NewSSHManager(cfg, eventBus)

	hostConfig := &ssh.HostConfig{
		Host:     "127.0.0.1",
		HostName: "127.0.0.1",
		User:     "invalid",
		Port:     2222,
		Password: "wrongpass",
	}

	client, err := manager.Connect(hostConfig, "wrongpass")

	assert.Error(t, err)
	assert.Nil(t, client)
}
