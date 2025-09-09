package session

import (
	"fmt"
	"sync"
	"time"

	"github.com/termbus/termbus/internal/logger"
	"github.com/termbus/termbus/pkg/types"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

var (
	ErrSessionNotFound     = fmt.Errorf("session not found")
	ErrWindowNotFound      = fmt.Errorf("window not found")
	ErrPaneNotFound        = fmt.Errorf("pane not found")
	ErrSessionNotConnected = fmt.Errorf("session not connected")
)

// SessionManager 会话管理器
type SessionManager struct {
	sessions        map[string]*types.Session
	activeSessionID string
	eventBus        types.EventBus
	sshPool         *SSHConnectionPool
	store           *SessionStore
	autoSave        bool
	mu              sync.RWMutex
}

// New 创建会话管理器
func New(eventBus types.EventBus, sshPool *SSHConnectionPool, store *SessionStore) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*types.Session),
		eventBus: eventBus,
		sshPool:  sshPool,
		store:    store,
		autoSave: true,
	}
}

// WithAutoSave 设置是否自动保存
func (m *SessionManager) WithAutoSave(autoSave bool) *SessionManager {
	m.autoSave = autoSave
	return m
}

// CreateSession 创建会话
func (m *SessionManager) CreateSession(hostConfig *types.SSHHostConfig) (*types.Session, error) {
	sessionID := GenerateSessionID()

	now := time.Now()
	session := &types.Session{
		ID:         sessionID,
		HostConfig: hostConfig,
		State:      types.SessionStateDisconnected,
		Windows:    make(map[string]*types.Window),
		CreatedAt:  now,
		KeepaliveConfig: &types.KeepaliveConfig{
			Enabled:  true,
			Interval: 60,
			CountMax: 3,
		},
		ReconnectConfig: &types.ReconnectConfig{
			Enabled:     true,
			MaxAttempts: 3,
			Interval:    5,
		},
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	go m.saveSession(session)

	m.eventBus.Publish("session.created", session)
	logger.GetLogger().Info("session created",
		zap.String("session_id", sessionID),
		zap.String("host", hostConfig.HostName),
	)

	return session, nil
}

// ConnectSession 连接会话
func (m *SessionManager) ConnectSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	session.State = types.SessionStateConnecting
	m.eventBus.Publish("session.state.changed", session)

	hostConfig := session.HostConfig
	hostKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)

	_, err := m.sshPool.GetConnection(hostKey)
	if err != nil {
		session.State = types.SessionStateError
		session.ErrorMsg = err.Error()
		m.eventBus.Publish("session.state.changed", session)
		return fmt.Errorf("failed to connect: %w", err)
	}

	session.State = types.SessionStateConnected
	now := time.Now()
	session.ConnectedAt = &now

	defaultWindow := m.createDefaultWindow(sessionID, hostConfig.Host)
	session.Windows[defaultWindow.ID] = defaultWindow
	session.ActiveWindowID = defaultWindow.ID

	m.saveSession(session)

	m.eventBus.Publish("session.connected", session)
	logger.GetLogger().Info("session connected",
		zap.String("session_id", sessionID),
		zap.String("host", hostConfig.HostName),
	)

	return nil
}

// DisconnectSession 断开会话
func (m *SessionManager) DisconnectSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	if session.State != types.SessionStateConnected {
		return nil
	}

	hostConfig := session.HostConfig
	hostKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)

	m.sshPool.ReleaseConnection(hostKey)

	session.State = types.SessionStateDisconnected
	session.ConnectedAt = nil

	m.eventBus.Publish("session.disconnected", session)
	logger.GetLogger().Info("session disconnected",
		zap.String("session_id", sessionID),
	)

	return nil
}

// DeleteSession 删除会话
func (m *SessionManager) DeleteSession(sessionID string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return err
	}

	if session.State == types.SessionStateConnected {
		m.DisconnectSession(sessionID)
	}

	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()

	if m.store != nil {
		m.store.Delete(sessionID)
	}

	m.eventBus.Publish("session.deleted", session)
	logger.GetLogger().Info("session deleted",
		zap.String("session_id", sessionID),
	)

	return nil
}

// GetSession 获取会话
func (m *SessionManager) GetSession(sessionID string) (*types.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	return session, nil
}

// ListSessions 列出所有会话
func (m *SessionManager) ListSessions() []*types.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*types.Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// SetActiveSession 设置活动会话
func (m *SessionManager) SetActiveSession(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[sessionID]; !exists {
		return ErrSessionNotFound
	}

	m.activeSessionID = sessionID
	m.eventBus.Publish("session.active.changed", sessionID)

	return nil
}

// GetActiveSession 获取活动会话
func (m *SessionManager) GetActiveSession() (*types.Session, error) {
	if m.activeSessionID == "" {
		return nil, fmt.Errorf("no active session")
	}

	return m.GetSession(m.activeSessionID)
}

// GetSSHClient 获取SSH客户端
func (m *SessionManager) GetSSHClient(sessionID string) (*ssh.Client, error) {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	if session.State != types.SessionStateConnected {
		return nil, ErrSessionNotConnected
	}

	hostConfig := session.HostConfig
	hostKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)

	return m.sshPool.GetConnection(hostKey)
}

// createDefaultWindow 创建默认窗口
func (m *SessionManager) createDefaultWindow(sessionID, hostAlias string) *types.Window {
	windowID := GenerateWindowID()

	window := &types.Window{
		ID:        windowID,
		SessionID: sessionID,
		HostID:    hostAlias,
		HostAlias: hostAlias,
		Panes:     make(map[string]*types.Pane),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	shellPane := &types.Pane{
		ID:        GeneratePaneID(),
		Type:      types.PaneTypeShell,
		Title:     fmt.Sprintf("Shell: %s", hostAlias),
		SessionID: sessionID,
		Active:    true,
	}

	window.Panes[shellPane.ID] = shellPane
	window.ActivePaneID = shellPane.ID

	return window
}

// generateSessionID 生成会话ID
func GenerateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}

// generateWindowID 生成窗口ID
func GenerateWindowID() string {
	return fmt.Sprintf("window_%d", time.Now().UnixNano())
}

// generatePaneID 生成窗格ID
func GeneratePaneID() string {
	return fmt.Sprintf("pane_%d", time.Now().UnixNano())
}

// SSHConnectionPool SSH连接池
type SSHConnectionPool struct {
	connections map[string]*ssh.Client
	refs        map[string]int
	mu          sync.RWMutex
}

// NewSSHConnectionPool 创建连接池
func NewSSHConnectionPool() *SSHConnectionPool {
	return &SSHConnectionPool{
		connections: make(map[string]*ssh.Client),
		refs:        make(map[string]int),
	}
}

// GetConnection 获取连接
func (p *SSHConnectionPool) GetConnection(key string) (*ssh.Client, error) {
	p.mu.RLock()
	client, exists := p.connections[key]
	if exists {
		p.refs[key]++
		p.mu.RUnlock()
		return client, nil
	}
	p.mu.RUnlock()

	return nil, fmt.Errorf("connection not found")
}

// SetConnection 设置连接
func (p *SSHConnectionPool) SetConnection(key string, client *ssh.Client) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.connections[key]; !exists {
		p.connections[key] = client
		p.refs[key] = 1
	} else {
		p.refs[key]++
	}
}

// ReleaseConnection 释放连接
func (p *SSHConnectionPool) ReleaseConnection(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if refs, exists := p.refs[key]; exists {
		refs--
		if refs <= 0 {
			if client, ok := p.connections[key]; ok {
				client.Close()
			}
			delete(p.connections, key)
			delete(p.refs, key)
		} else {
			p.refs[key] = refs
		}
	}
}

// saveSession 保存会话
func (m *SessionManager) saveSession(session *types.Session) {
	if m.store == nil || !m.autoSave {
		return
	}

	if err := m.store.Save(session); err != nil {
		logger.GetLogger().Warn("Failed to save session",
			zap.String("session_id", session.ID),
			zap.Any("error", err),
		)
	}
}

// LoadSessions 加载所有会话
func (m *SessionManager) LoadSessions() error {
	if m.store == nil {
		return nil
	}

	sessions, err := m.store.LoadAll()
	if err != nil {
		return err
	}

	m.mu.Lock()
	for id, session := range sessions {
		m.sessions[id] = session
	}
	m.mu.Unlock()

	logger.GetLogger().Info("Sessions loaded",
		zap.Int("count", len(sessions)),
	)

	return nil
}

// SaveAll 保存所有会话
func (m *SessionManager) SaveAll() error {
	if m.store == nil {
		return nil
	}

	return m.store.SaveAll(m.sessions)
}
