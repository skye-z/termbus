package sftp

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/termbus/termbus/internal/logger"
	"go.uber.org/zap"
)

type ResumeManager struct {
	stateDir string
	states   map[string]*ResumeState
	mu       sync.RWMutex
}

type ResumeState struct {
	SessionID   string    `json:"session_id"`
	LocalPath   string    `json:"local_path"`
	RemotePath  string    `json:"remote_path"`
	Size        int64     `json:"size"`
	Transferred int64     `json:"transferred"`
	Checksum    string    `json:"checksum"`
	LastUpdated time.Time `json:"last_updated"`
	Direction   string    `json:"direction"`
}

func NewResumeManager(stateDir string) (*ResumeManager, error) {
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state dir: %w", err)
	}

	return &ResumeManager{
		stateDir: stateDir,
		states:   make(map[string]*ResumeState),
	}, nil
}

func (m *ResumeManager) SaveState(state *ResumeState) error {
	state.LastUpdated = time.Now()

	m.mu.Lock()
	m.states[state.Key()] = state
	m.mu.Unlock()

	filename := filepath.Join(m.stateDir, state.Key()+".json")
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	logger.GetLogger().Debug("resume state saved",
		zap.String("key", state.Key()),
		zap.Int64("transferred", state.Transferred),
	)
	return nil
}

func (m *ResumeManager) LoadState(key string) (*ResumeState, error) {
	m.mu.RLock()
	if state, exists := m.states[key]; exists {
		m.mu.RUnlock()
		return state, nil
	}
	m.mu.RUnlock()

	filename := filepath.Join(m.stateDir, key+".json")
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("state not found")
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state ResumeState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	m.mu.Lock()
	m.states[key] = &state
	m.mu.Unlock()

	return &state, nil
}

func (m *ResumeManager) DeleteState(key string) error {
	m.mu.Lock()
	delete(m.states, key)
	m.mu.Unlock()

	filename := filepath.Join(m.stateDir, key+".json")
	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete state file: %w", err)
	}

	return nil
}

func (s *ResumeState) Key() string {
	return fmt.Sprintf("%s_%s_%s_%s", s.SessionID, s.Direction, s.LocalPath, s.RemotePath)
}

func (m *ResumeManager) CalculateChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	buffer := make([]byte, 32*1024)

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			hash.Write(buffer[:n])
		}
		if err != nil {
			break
		}
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (m *ResumeManager) GetFileInfo(path string) (int64, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, "", fmt.Errorf("failed to stat file: %w", err)
	}

	checksum, err := m.CalculateChecksum(path)
	if err != nil {
		return 0, "", err
	}

	return info.Size(), checksum, nil
}

func (m *ResumeManager) ResumeUpload(sftpManager *SFTPManager, sessionID, localPath, remotePath string, progress chan float64) error {
	state, err := m.LoadState(sessionID + "_upload_" + localPath + "_" + remotePath)
	if err != nil {
		return fmt.Errorf("no resume state found: %w", err)
	}

	size, checksum, err := m.GetFileInfo(localPath)
	if err != nil {
		return err
	}

	if state.Checksum != checksum {
		logger.GetLogger().Warn("file changed, restarting transfer",
			zap.String("old_checksum", state.Checksum),
			zap.String("new_checksum", checksum),
		)
		m.DeleteState(state.Key())
		return fmt.Errorf("file changed, cannot resume")
	}

	if state.Transferred >= state.Size {
		logger.GetLogger().Info("file already transferred")
		return nil
	}

	return sftpManager.ResumeUploadFromOffset(sessionID, localPath, remotePath, state.Transferred, size, progress)
}

func (m *ResumeManager) ResumeDownload(sftpManager *SFTPManager, sessionID, remotePath, localPath string, progress chan float64) error {
	state, err := m.LoadState(sessionID + "_download_" + remotePath + "_" + localPath)
	if err != nil {
		return fmt.Errorf("no resume state found: %w", err)
	}

	remoteSize, err := sftpManager.GetFileSize(sessionID, remotePath)
	if err != nil {
		return err
	}

	if state.Transferred >= state.Size && state.Size != remoteSize {
		logger.GetLogger().Warn("remote file changed, restarting transfer")
		m.DeleteState(state.Key())
		return fmt.Errorf("remote file changed, cannot resume")
	}

	return sftpManager.ResumeDownloadFromOffset(sessionID, remotePath, localPath, state.Transferred, progress)
}

func (m *ResumeManager) CreateResumeState(sessionID, localPath, remotePath, direction string, size int64) *ResumeState {
	checksum, _ := m.CalculateChecksum(localPath)

	state := &ResumeState{
		SessionID:   sessionID,
		LocalPath:   localPath,
		RemotePath:  remotePath,
		Size:        size,
		Transferred: 0,
		Checksum:    checksum,
		LastUpdated: time.Now(),
		Direction:   direction,
	}

	m.SaveState(state)
	return state
}

func (m *ResumeManager) UpdateProgress(key string, transferred int64) {
	m.mu.RLock()
	state, exists := m.states[key]
	m.mu.RUnlock()

	if !exists {
		return
	}

	state.Transferred = transferred
	m.SaveState(state)
}

type Client interface {
	ResumeUploadFromOffset(localPath, remotePath string, offset, totalSize int64, progress chan float64) error
	ResumeDownloadFromOffset(remotePath, localPath string, offset int64, progress chan float64) error
	GetFileSize(path string) (int64, error)
}
