package e2e

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/session"
	"github.com/termbus/termbus/internal/ssh"
	"github.com/termbus/termbus/internal/tunnel"
	"github.com/termbus/termbus/pkg/types"
)

func TestE2E_TunnelLocalForward(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	config := loadTestConfig(t)
	t.Logf("INPUT: Creating local port forward tunnel")
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()

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

	hostKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)
	sshPool.SetConnection(hostKey, client)

	sessionMgr := session.New(eventBus, sshPool, nil)
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	sessionHostConfig := &types.SSHHostConfig{
		Host:     config.Hosts.SSHServer.Hostname,
		HostName: config.Hosts.SSHServer.Hostname,
		User:     config.Hosts.SSHServer.Username,
		Port:     config.Hosts.SSHServer.Port,
	}

	s, err := sessionMgr.CreateSession(sessionHostConfig)
	require.NoError(t, err)
	err = sessionMgr.ConnectSession(s.ID)
	require.NoError(t, err)

	localPort := 28080
	testTunnel := &types.ForwardTunnel{
		ID:         fmt.Sprintf("tunnel_local_%d", os.Getpid()),
		SessionID:  s.ID,
		Type:       types.ForwardTypeLocal,
		LocalAddr:  fmt.Sprintf("127.0.0.1:%d", localPort),
		RemoteAddr: "127.0.0.1:22",
		Status:     types.TunnelStatusStopped,
		AutoStart:  false,
	}

	err = tunnelMgr.CreateTunnel(s.ID, testTunnel)
	require.NoError(t, err)

	err = tunnelMgr.StartTunnel(testTunnel.ID)
	require.NoError(t, err)

	tunnel, err := tunnelMgr.GetTunnel(testTunnel.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TunnelStatusRunning, tunnel.Status)

	time.Sleep(500 * time.Millisecond)

	conn, err := net.Dial("tcp", testTunnel.LocalAddr)
	if err == nil {
		conn.Close()
	}

	err = tunnelMgr.StopTunnel(testTunnel.ID)
	require.NoError(t, err)

	tunnel, _ = tunnelMgr.GetTunnel(testTunnel.ID)
	if tunnel != nil {
		assert.Equal(t, types.TunnelStatusStopped, tunnel.Status)
	}

	tunnelMgr.DeleteTunnel(testTunnel.ID)

	sessionMgr.DisconnectSession(s.ID)
	sessionMgr.DeleteSession(s.ID)
	sshManager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)
}

func TestE2E_TunnelRemoteForward(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	config := loadTestConfig(t)
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()

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

	hostKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)
	sshPool.SetConnection(hostKey, client)

	sessionMgr := session.New(eventBus, sshPool, nil)
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	sessionHostConfig := &types.SSHHostConfig{
		Host:     config.Hosts.SSHServer.Hostname,
		HostName: config.Hosts.SSHServer.Hostname,
		User:     config.Hosts.SSHServer.Username,
		Port:     config.Hosts.SSHServer.Port,
	}

	s, err := sessionMgr.CreateSession(sessionHostConfig)
	require.NoError(t, err)
	err = sessionMgr.ConnectSession(s.ID)
	require.NoError(t, err)

	testTunnel := &types.ForwardTunnel{
		ID:         fmt.Sprintf("tunnel_remote_%d", os.Getpid()),
		SessionID:  s.ID,
		Type:       types.ForwardTypeRemote,
		LocalAddr:  "127.0.0.1:0",
		RemoteAddr: "127.0.0.1:28080",
		Status:     types.TunnelStatusStopped,
		AutoStart:  false,
	}

	err = tunnelMgr.CreateTunnel(s.ID, testTunnel)
	require.NoError(t, err)

	err = tunnelMgr.StartTunnel(testTunnel.ID)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	tunnel, err := tunnelMgr.GetTunnel(testTunnel.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TunnelStatusRunning, tunnel.Status)

	err = tunnelMgr.StopTunnel(testTunnel.ID)
	require.NoError(t, err)

	tunnelMgr.DeleteTunnel(testTunnel.ID)

	sessionMgr.DisconnectSession(s.ID)
	sessionMgr.DeleteSession(s.ID)
	sshManager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)
}

func TestE2E_TunnelDynamicSOCKS(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	config := loadTestConfig(t)
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()

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

	hostKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)
	sshPool.SetConnection(hostKey, client)

	sessionMgr := session.New(eventBus, sshPool, nil)
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	sessionHostConfig := &types.SSHHostConfig{
		Host:     config.Hosts.SSHServer.Hostname,
		HostName: config.Hosts.SSHServer.Hostname,
		User:     config.Hosts.SSHServer.Username,
		Port:     config.Hosts.SSHServer.Port,
	}

	s, err := sessionMgr.CreateSession(sessionHostConfig)
	require.NoError(t, err)
	err = sessionMgr.ConnectSession(s.ID)
	require.NoError(t, err)

	testTunnel := &types.ForwardTunnel{
		ID:         fmt.Sprintf("tunnel_socks_%d", os.Getpid()),
		SessionID:  s.ID,
		Type:       types.ForwardTypeDynamic,
		LocalAddr:  "127.0.0.1:0",
		RemoteAddr: "",
		Status:     types.TunnelStatusStopped,
		AutoStart:  false,
	}

	err = tunnelMgr.CreateTunnel(s.ID, testTunnel)
	require.NoError(t, err)

	err = tunnelMgr.StartTunnel(testTunnel.ID)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	tunnel, err := tunnelMgr.GetTunnel(testTunnel.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TunnelStatusRunning, tunnel.Status)

	tunnels := tunnelMgr.ListTunnels(s.ID)
	assert.GreaterOrEqual(t, len(tunnels), 1)

	err = tunnelMgr.StopTunnel(testTunnel.ID)
	require.NoError(t, err)

	tunnelMgr.DeleteTunnel(testTunnel.ID)

	sessionMgr.DisconnectSession(s.ID)
	sessionMgr.DeleteSession(s.ID)
	sshManager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)
}

func TestE2E_TunnelListAndStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	config := loadTestConfig(t)
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()

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

	hostKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)
	sshPool.SetConnection(hostKey, client)

	sessionMgr := session.New(eventBus, sshPool, nil)
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	sessionHostConfig := &types.SSHHostConfig{
		Host:     config.Hosts.SSHServer.Hostname,
		HostName: config.Hosts.SSHServer.Hostname,
		User:     config.Hosts.SSHServer.Username,
		Port:     config.Hosts.SSHServer.Port,
	}

	s, err := sessionMgr.CreateSession(sessionHostConfig)
	require.NoError(t, err)
	err = sessionMgr.ConnectSession(s.ID)
	require.NoError(t, err)

	tunnels := tunnelMgr.ListTunnels(s.ID)
	initialCount := len(tunnels)

	testTunnel := &types.ForwardTunnel{
		ID:         fmt.Sprintf("tunnel_list_%d", os.Getpid()),
		SessionID:  s.ID,
		Type:       types.ForwardTypeLocal,
		LocalAddr:  "127.0.0.1:0",
		RemoteAddr: "127.0.0.1:80",
		Status:     types.TunnelStatusStopped,
		AutoStart:  false,
	}

	err = tunnelMgr.CreateTunnel(s.ID, testTunnel)
	require.NoError(t, err)

	tunnels = tunnelMgr.ListTunnels(s.ID)
	assert.Equal(t, initialCount+1, len(tunnels))

	tun, err := tunnelMgr.GetTunnel(testTunnel.ID)
	require.NoError(t, err)
	assert.NotNil(t, tun)

	err = tunnelMgr.StartTunnel(testTunnel.ID)
	require.NoError(t, err)

	tun, err = tunnelMgr.GetTunnel(testTunnel.ID)
	assert.Equal(t, types.TunnelStatusRunning, tun.Status)

	err = tunnelMgr.StopTunnel(testTunnel.ID)
	require.NoError(t, err)

	tun, err = tunnelMgr.GetTunnel(testTunnel.ID)
	assert.Equal(t, types.TunnelStatusStopped, tun.Status)

	tunnelMgr.DeleteTunnel(testTunnel.ID)

	tunnels = tunnelMgr.ListTunnels(s.ID)
	assert.Equal(t, initialCount, len(tunnels))

	sessionMgr.DisconnectSession(s.ID)
	sessionMgr.DeleteSession(s.ID)
	sshManager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)
}
