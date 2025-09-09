package ssh

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/logger"
	"github.com/termbus/termbus/pkg/types"
	"go.uber.org/zap"
)

// SSHService SSH服务，整合SSH连接和会话管理
type SSHService struct {
	sshManager *SSHManager
	sessionMgr SessionManager
	eventBus   *eventbus.Manager
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// SessionManager 会话管理器接口
type SessionManager interface {
	GetSession(sessionID string) (*types.Session, error)
	ListSessions() []*types.Session
}

// NewSSHService 创建SSH服务
func NewSSHService(sshManager *SSHManager, sessionMgr SessionManager, eventBus *eventbus.Manager) *SSHService {
	ctx, cancel := context.WithCancel(context.Background())
	return &SSHService{
		sshManager: sshManager,
		sessionMgr: sessionMgr,
		eventBus:   eventBus,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start 启动SSH服务
func (s *SSHService) Start() error {
	logger.GetLogger().Info("SSH service starting")

	s.eventBus.Subscribe("session.created", s.handleSessionCreated)
	s.eventBus.Subscribe("session.connect.request", s.handleConnectRequest)
	s.eventBus.Subscribe("session.disconnect.request", s.handleDisconnectRequest)

	s.wg.Add(1)
	go s.reconnectWorker()

	return nil
}

// Stop 停止SSH服务
func (s *SSHService) Stop() error {
	logger.GetLogger().Info("SSH service stopping")

	s.cancel()
	s.wg.Wait()

	return nil
}

// handleSessionCreated 处理会话创建事件
func (s *SSHService) handleSessionCreated(args ...interface{}) {
	if len(args) < 1 {
		return
	}

	session, ok := args[0].(*types.Session)
	if !ok {
		return
	}

	logger.GetLogger().Info("Session created",
		zap.String("session_id", session.ID),
		zap.String("host", session.HostConfig.HostName),
	)
}

// handleConnectRequest 处理连接请求
func (s *SSHService) handleConnectRequest(args ...interface{}) {
	if len(args) < 2 {
		return
	}

	sessionID, ok1 := args[0].(string)
	password, ok2 := args[1].(string)
	if !ok1 || !ok2 {
		return
	}

	session, err := s.sessionMgr.GetSession(sessionID)
	if err != nil {
		logger.GetLogger().Error("Failed to get session",
			zap.String("session_id", sessionID),
			zap.Any("error", err),
		)
		return
	}

	hostConfig := session.HostConfig
	sshHostConfig := &HostConfig{
		Host:         hostConfig.Host,
		HostName:     hostConfig.HostName,
		User:         hostConfig.User,
		Port:         hostConfig.Port,
		IdentityFile: hostConfig.IdentityFile[0],
		ProxyJump:    hostConfig.ProxyJump,
		ProxyCommand: hostConfig.ProxyCommand,
		Password:     password,
	}

	client, err := s.sshManager.Connect(sshHostConfig, password)
	if err != nil {
		logger.GetLogger().Error("Failed to connect SSH",
			zap.String("host", hostConfig.HostName),
			zap.Any("error", err),
		)
		s.eventBus.Publish("session.connect.failed", sessionID, err)
		return
	}

	hostKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)
	s.eventBus.Publish("ssh.client.ready", sessionID, hostKey, client)

	logger.GetLogger().Info("SSH connection established",
		zap.String("session_id", sessionID),
		zap.String("host", hostConfig.HostName),
	)
}

// handleDisconnectRequest 处理断开请求
func (s *SSHService) handleDisconnectRequest(args ...interface{}) {
	if len(args) < 1 {
		return
	}

	sessionID, ok := args[0].(string)
	if !ok {
		return
	}

	session, err := s.sessionMgr.GetSession(sessionID)
	if err != nil {
		return
	}

	hostConfig := session.HostConfig
	s.sshManager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)

	logger.GetLogger().Info("SSH connection closed",
		zap.String("session_id", sessionID),
		zap.String("host", hostConfig.HostName),
	)
}

// reconnectWorker 断线重连工作器
func (s *SSHService) reconnectWorker() {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkAndReconnect()
		}
	}
}

// checkAndReconnect 检查并重连
func (s *SSHService) checkAndReconnect() {
	sessions := s.sessionMgr.ListSessions()

	for _, session := range sessions {
		if session.State != types.SessionStateConnected {
			if session.ReconnectConfig != nil && session.ReconnectConfig.Enabled {
				s.attemptReconnect(session)
			}
		}
	}
}

// attemptReconnect 尝试重连
func (s *SSHService) attemptReconnect(session *types.Session) {
	if session.ErrorMsg == "" {
		return
	}

	logger.GetLogger().Info("Attempting to reconnect",
		zap.String("session_id", session.ID),
		zap.String("host", session.HostConfig.HostName),
	)

	session.State = types.SessionStateReconnecting
	s.eventBus.Publish("session.state.changed", session)

	s.eventBus.Publish("session.reconnect.attempt", session.ID)
}

// RequestConnect 请求连接
func (s *SSHService) RequestConnect(sessionID, password string) {
	s.eventBus.Publish("session.connect.request", sessionID, password)
}

// RequestDisconnect 请求断开
func (s *SSHService) RequestDisconnect(sessionID string) {
	s.eventBus.Publish("session.disconnect.request", sessionID)
}

// GetHostConfig 获取主机配置
func (s *SSHService) GetHostConfig(hostAlias string) (*HostConfig, error) {
	return s.sshManager.GetHostConfig(hostAlias)
}

// ListHostConfigs 列出所有主机配置
func (s *SSHService) ListHostConfigs() ([]HostConfig, error) {
	return s.sshManager.ScanSSHConfigs()
}

// ListIdentities 列出所有私钥
func (s *SSHService) ListIdentities() ([]string, error) {
	return ListIdentities()
}
