package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/termbus/termbus/internal/logger"
	"github.com/termbus/termbus/pkg/types"
	"go.uber.org/zap"
)

// SessionStore 会话存储
type SessionStore struct {
	dataDir string
	mu      sync.RWMutex
}

// NewSessionStore 创建会话存储
func NewSessionStore(dataDir string) (*SessionStore, error) {
	store := &SessionStore{
		dataDir: dataDir,
	}

	sessionsDir := filepath.Join(dataDir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	return store, nil
}

// Save 保存会话
func (s *SessionStore) Save(session *types.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionsDir := filepath.Join(s.dataDir, "sessions")
	sessionFile := filepath.Join(sessionsDir, fmt.Sprintf("%s.json", session.ID))

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(sessionFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	logger.GetLogger().Debug("Session saved",
		zap.String("session_id", session.ID),
	)

	return nil
}

// Load 加载会话
func (s *SessionStore) Load(sessionID string) (*types.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionsDir := filepath.Join(s.dataDir, "sessions")
	sessionFile := filepath.Join(sessionsDir, fmt.Sprintf("%s.json", sessionID))

	data, err := os.ReadFile(sessionFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	var session types.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	logger.GetLogger().Debug("Session loaded",
		zap.String("session_id", sessionID),
	)

	return &session, nil
}

// Delete 删除会话
func (s *SessionStore) Delete(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionsDir := filepath.Join(s.dataDir, "sessions")
	sessionFile := filepath.Join(sessionsDir, fmt.Sprintf("%s.json", sessionID))

	if err := os.Remove(sessionFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete session file: %w", err)
	}

	logger.GetLogger().Debug("Session deleted",
		zap.String("session_id", sessionID),
	)

	return nil
}

// List 列出所有会话
func (s *SessionStore) List() ([]*types.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionsDir := filepath.Join(s.dataDir, "sessions")

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*types.Session{}, nil
		}
		return nil, fmt.Errorf("failed to read sessions directory: %w", err)
	}

	sessions := make([]*types.Session, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		sessionID := entry.Name()
		if len(sessionID) > 5 && sessionID[len(sessionID)-5:] == ".json" {
			sessionID = sessionID[:len(sessionID)-5]
		}

		session, err := s.Load(sessionID)
		if err != nil {
			logger.GetLogger().Warn("Failed to load session",
				zap.String("session_id", sessionID),
				zap.Any("error", err),
			)
			continue
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// SaveAll 保存所有会话
func (s *SessionStore) SaveAll(sessions map[string]*types.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, session := range sessions {
		if err := s.saveNoLock(session); err != nil {
			logger.GetLogger().Error("Failed to save session",
				zap.String("session_id", session.ID),
				zap.Any("error", err),
			)
			continue
		}
	}

	return nil
}

// saveNoLock 保存会话（不加锁）
func (s *SessionStore) saveNoLock(session *types.Session) error {
	sessionsDir := filepath.Join(s.dataDir, "sessions")
	sessionFile := filepath.Join(sessionsDir, fmt.Sprintf("%s.json", session.ID))

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(sessionFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// LoadAll 加载所有会话
func (s *SessionStore) LoadAll() (map[string]*types.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions, err := s.List()
	if err != nil {
		return nil, err
	}

	sessionMap := make(map[string]*types.Session)
	for _, session := range sessions {
		sessionMap[session.ID] = session
	}

	return sessionMap, nil
}

// Clean 清理所有会话
func (s *SessionStore) Clean() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionsDir := filepath.Join(s.dataDir, "sessions")

	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read sessions directory: %w", err)
	}

	for _, entry := range entries {
		sessionFile := filepath.Join(sessionsDir, entry.Name())
		if err := os.Remove(sessionFile); err != nil {
			logger.GetLogger().Warn("Failed to delete session file",
				zap.String("file", sessionFile),
				zap.Any("error", err),
			)
		}
	}

	logger.GetLogger().Info("All sessions cleaned")

	return nil
}
