package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/session"
	"github.com/termbus/termbus/pkg/types"
)

func TestSessionManager_CreateSession(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()

	manager := session.New(eventBus, sshPool, nil)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     22,
		Group:    "test-group",
	}

	s, err := manager.CreateSession(hostConfig)

	require.NoError(t, err)
	assert.NotNil(t, s)
	assert.NotEmpty(t, s.ID)
	assert.Equal(t, types.SessionStateDisconnected, s.State)
	assert.Equal(t, "test-host", s.HostConfig.Host)
	assert.True(t, s.KeepaliveConfig.Enabled)
	assert.True(t, s.ReconnectConfig.Enabled)
	assert.Len(t, s.Windows, 0)
}

func TestSessionManager_GetSession(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()

	manager := session.New(eventBus, sshPool, nil)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     22,
	}

	createdSession, _ := manager.CreateSession(hostConfig)

	retrievedSession, err := manager.GetSession(createdSession.ID)

	require.NoError(t, err)
	assert.NotNil(t, retrievedSession)
	assert.Equal(t, createdSession.ID, retrievedSession.ID)
}

func TestSessionManager_GetSessionNotFound(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()

	manager := session.New(eventBus, sshPool, nil)

	s, err := manager.GetSession("non-existent-session-id")

	assert.Error(t, err)
	assert.Nil(t, s)
	assert.Equal(t, session.ErrSessionNotFound, err)
}

func TestSessionManager_ListSessions(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()

	manager := session.New(eventBus, sshPool, nil)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     22,
	}

	manager.CreateSession(hostConfig)
	manager.CreateSession(hostConfig)
	manager.CreateSession(hostConfig)

	sessions := manager.ListSessions()

	assert.Len(t, sessions, 3)
}

func TestSessionManager_SetActiveSession(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()

	manager := session.New(eventBus, sshPool, nil)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     22,
	}

	s, _ := manager.CreateSession(hostConfig)

	err := manager.SetActiveSession(s.ID)

	require.NoError(t, err)

	activeSession, err := manager.GetActiveSession()

	require.NoError(t, err)
	assert.Equal(t, s.ID, activeSession.ID)
}

func TestSessionManager_DeleteSession(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()

	manager := session.New(eventBus, sshPool, nil)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     22,
	}

	s, _ := manager.CreateSession(hostConfig)

	err := manager.DeleteSession(s.ID)

	require.NoError(t, err)

	_, err = manager.GetSession(s.ID)

	assert.Error(t, err)
	assert.Equal(t, session.ErrSessionNotFound, err)
}

func TestSessionManager_DisconnectSession(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()

	manager := session.New(eventBus, sshPool, nil)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     22,
	}

	s, _ := manager.CreateSession(hostConfig)

	manager.DisconnectSession(s.ID)

	retrievedSession, _ := manager.GetSession(s.ID)

	assert.Equal(t, types.SessionStateDisconnected, retrievedSession.State)
	assert.Nil(t, retrievedSession.ConnectedAt)
}

func TestSessionManager_WithAutoSave(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()

	manager := session.New(eventBus, sshPool, nil)

	manager.WithAutoSave(false)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     22,
	}

	_, _ = manager.CreateSession(hostConfig)

	assert.Len(t, manager.ListSessions(), 1)

	time.Sleep(100 * time.Millisecond)
}

func TestSSHConnectionPool(t *testing.T) {
	pool := session.NewSSHConnectionPool()

	assert.NotNil(t, pool)

	_, err := pool.GetConnection("non-existent")

	assert.Error(t, err)
}

func TestSessionManager_GetSSHClient(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()

	manager := session.New(eventBus, sshPool, nil)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     22,
	}

	s, _ := manager.CreateSession(hostConfig)

	client, err := manager.GetSSHClient(s.ID)

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Equal(t, session.ErrSessionNotConnected, err)
}

func TestGenerateIDs(t *testing.T) {
	id1 := session.GenerateSessionID()
	id2 := session.GenerateSessionID()
	id3 := session.GenerateWindowID()
	id4 := session.GenerateWindowID()
	id5 := session.GeneratePaneID()
	id6 := session.GeneratePaneID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEmpty(t, id3)
	assert.NotEmpty(t, id4)
	assert.NotEmpty(t, id5)
	assert.NotEmpty(t, id6)

	assert.NotEqual(t, id1, id2)
	assert.NotEqual(t, id3, id4)
	assert.NotEqual(t, id5, id6)
}
