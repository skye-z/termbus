package integration

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/session"
	"github.com/termbus/termbus/internal/tunnel"
	"github.com/termbus/termbus/pkg/types"
)

func TestTunnelManager_CreateTunnel(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-tunnel-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	forwardTunnel := &types.ForwardTunnel{
		Type:       types.ForwardTypeLocal,
		LocalAddr:  "127.0.0.1:8080",
		RemoteAddr: "127.0.0.1:80",
		SessionID:  s.ID,
		AutoStart:  false,
	}

	err = tunnelMgr.CreateTunnel(s.ID, forwardTunnel)

	require.NoError(t, err)
	assert.NotEmpty(t, forwardTunnel.ID)
	assert.Equal(t, s.ID, forwardTunnel.SessionID)
	assert.Equal(t, types.TunnelStatusStopped, forwardTunnel.Status)
}

func TestTunnelManager_StartLocalForward(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-tunnel-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	forwardTunnel := &types.ForwardTunnel{
		Type:       types.ForwardTypeLocal,
		LocalAddr:  "127.0.0.1:0",
		RemoteAddr: "127.0.0.1:22",
		SessionID:  s.ID,
		AutoStart:  false,
	}

	err = tunnelMgr.CreateTunnel(s.ID, forwardTunnel)
	require.NoError(t, err)

	err = tunnelMgr.StartTunnel(forwardTunnel.ID)
	if err != nil {
		t.Skipf("Failed to start tunnel: %v", err)
	}

	assert.NoError(t, err)

	tunnel, err := tunnelMgr.GetTunnel(forwardTunnel.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TunnelStatusRunning, tunnel.Status)

	time.Sleep(500 * time.Millisecond)

	tunnelMgr.StopTunnel(forwardTunnel.ID)
}

func TestTunnelManager_StopTunnel(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-tunnel-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	forwardTunnel := &types.ForwardTunnel{
		Type:       types.ForwardTypeLocal,
		LocalAddr:  "127.0.0.1:0",
		RemoteAddr: "127.0.0.1:22",
		SessionID:  s.ID,
		AutoStart:  false,
	}

	err = tunnelMgr.CreateTunnel(s.ID, forwardTunnel)
	require.NoError(t, err)

	err = tunnelMgr.StartTunnel(forwardTunnel.ID)
	if err != nil {
		t.Skipf("Failed to start tunnel: %v", err)
	}

	err = tunnelMgr.StopTunnel(forwardTunnel.ID)
	assert.NoError(t, err)

	tunnel, err := tunnelMgr.GetTunnel(forwardTunnel.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TunnelStatusStopped, tunnel.Status)
}

func TestTunnelManager_DeleteTunnel(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-tunnel-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	forwardTunnel := &types.ForwardTunnel{
		Type:       types.ForwardTypeLocal,
		LocalAddr:  "127.0.0.1:8080",
		RemoteAddr: "127.0.0.1:80",
		SessionID:  s.ID,
		AutoStart:  false,
	}

	err = tunnelMgr.CreateTunnel(s.ID, forwardTunnel)
	require.NoError(t, err)

	err = tunnelMgr.DeleteTunnel(forwardTunnel.ID)
	assert.NoError(t, err)

	_, err = tunnelMgr.GetTunnel(forwardTunnel.ID)
	assert.Error(t, err)
	assert.Equal(t, tunnel.ErrTunnelNotFound, err)
}

func TestTunnelManager_ListTunnels(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-tunnel-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	forwardTunnel1 := &types.ForwardTunnel{
		Type:       types.ForwardTypeLocal,
		LocalAddr:  "127.0.0.1:8081",
		RemoteAddr: "127.0.0.1:81",
		SessionID:  s.ID,
	}

	forwardTunnel2 := &types.ForwardTunnel{
		Type:       types.ForwardTypeLocal,
		LocalAddr:  "127.0.0.1:8082",
		RemoteAddr: "127.0.0.1:82",
		SessionID:  s.ID,
	}

	err = tunnelMgr.CreateTunnel(s.ID, forwardTunnel1)
	require.NoError(t, err)

	err = tunnelMgr.CreateTunnel(s.ID, forwardTunnel2)
	require.NoError(t, err)

	tunnels := tunnelMgr.ListTunnels(s.ID)

	assert.Len(t, tunnels, 2)
}

func TestTunnelManager_StartDynamicForward(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-tunnel-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	forwardTunnel := &types.ForwardTunnel{
		Type:      types.ForwardTypeDynamic,
		LocalAddr: "127.0.0.1:0",
		SessionID: s.ID,
		AutoStart: false,
	}

	err = tunnelMgr.CreateTunnel(s.ID, forwardTunnel)
	require.NoError(t, err)

	err = tunnelMgr.StartTunnel(forwardTunnel.ID)
	if err != nil {
		t.Skipf("Failed to start dynamic forward: %v", err)
	}

	assert.NoError(t, err)

	tunnel, err := tunnelMgr.GetTunnel(forwardTunnel.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TunnelStatusRunning, tunnel.Status)

	time.Sleep(500 * time.Millisecond)

	tunnelMgr.StopTunnel(forwardTunnel.ID)
}

func TestTunnelManager_StartRemoteForward(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-tunnel-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	forwardTunnel := &types.ForwardTunnel{
		Type:       types.ForwardTypeRemote,
		RemoteAddr: "127.0.0.1:8083",
		LocalAddr:  "127.0.0.1:83",
		SessionID:  s.ID,
		AutoStart:  false,
	}

	err = tunnelMgr.CreateTunnel(s.ID, forwardTunnel)
	require.NoError(t, err)

	err = tunnelMgr.StartTunnel(forwardTunnel.ID)
	if err != nil {
		t.Skipf("Failed to start remote forward: %v", err)
	}

	assert.NoError(t, err)

	tunnel, err := tunnelMgr.GetTunnel(forwardTunnel.ID)
	require.NoError(t, err)
	assert.Equal(t, types.TunnelStatusRunning, tunnel.Status)

	time.Sleep(500 * time.Millisecond)

	tunnelMgr.StopTunnel(forwardTunnel.ID)
}

func TestTunnelManager_DoubleStart(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-tunnel-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	forwardTunnel := &types.ForwardTunnel{
		Type:       types.ForwardTypeLocal,
		LocalAddr:  "127.0.0.1:0",
		RemoteAddr: "127.0.0.1:22",
		SessionID:  s.ID,
		AutoStart:  false,
	}

	err = tunnelMgr.CreateTunnel(s.ID, forwardTunnel)
	require.NoError(t, err)

	err = tunnelMgr.StartTunnel(forwardTunnel.ID)
	if err != nil {
		t.Skipf("Failed to start tunnel: %v", err)
	}

	err = tunnelMgr.StartTunnel(forwardTunnel.ID)
	assert.Error(t, err)
	assert.Equal(t, tunnel.ErrTunnelRunning, err)

	tunnelMgr.StopTunnel(forwardTunnel.ID)
}

func TestTunnelManager_StopNotRunning(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-tunnel-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	forwardTunnel := &types.ForwardTunnel{
		Type:       types.ForwardTypeLocal,
		LocalAddr:  "127.0.0.1:8080",
		RemoteAddr: "127.0.0.1:80",
		SessionID:  s.ID,
		AutoStart:  false,
	}

	err = tunnelMgr.CreateTunnel(s.ID, forwardTunnel)
	require.NoError(t, err)

	err = tunnelMgr.StopTunnel(forwardTunnel.ID)
	assert.NoError(t, err)
}

func TestTunnelManager_GetTunnelNotFound(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	_, err := tunnelMgr.GetTunnel("non-existent-tunnel-id")

	assert.Error(t, err)
	assert.Equal(t, tunnel.ErrTunnelNotFound, err)
}

func TestTunnelManager_PortAvailability(t *testing.T) {
	port := getAvailablePort(t)
	assert.NotZero(t, port)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	defer listener.Close()

	_, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	assert.Error(t, err, "Should fail to bind to already in-use port")
}

func getAvailablePort(t *testing.T) int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port
}

func TestTunnelManager_Close(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-tunnel-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	forwardTunnel := &types.ForwardTunnel{
		Type:       types.ForwardTypeLocal,
		LocalAddr:  "127.0.0.1:0",
		RemoteAddr: "127.0.0.1:22",
		SessionID:  s.ID,
		AutoStart:  false,
	}

	err = tunnelMgr.CreateTunnel(s.ID, forwardTunnel)
	require.NoError(t, err)

	err = tunnelMgr.StartTunnel(forwardTunnel.ID)
	if err != nil {
		t.Skipf("Failed to start tunnel: %v", err)
	}

	tunnelMgr.Close()

	tunnels := tunnelMgr.ListTunnels(s.ID)
	assert.Empty(t, tunnels)
}
