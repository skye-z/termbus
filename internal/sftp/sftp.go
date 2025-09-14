package sftp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/sftp"
	"github.com/termbus/termbus/internal/logger"
	"github.com/termbus/termbus/pkg/interfaces"
	"github.com/termbus/termbus/pkg/types"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

var (
	ErrClientNotFound      = fmt.Errorf("sftp client not found")
	ErrSessionNotConnected = fmt.Errorf("session not connected")
)

type SFTPManager struct {
	sessionManager interfaces.SessionManager
	clients        map[string]*sftp.Client
	mu             sync.RWMutex
}

func NewSFTPManager(sessionManager interfaces.SessionManager) *SFTPManager {
	return &SFTPManager{
		sessionManager: sessionManager,
		clients:        make(map[string]*sftp.Client),
	}
}

func (m *SFTPManager) getSSHClient(sessionID string) (*ssh.Client, error) {
	return m.sessionManager.GetSSHClient(sessionID)
}

func (m *SFTPManager) getOrCreateClient(sessionID string) (*sftp.Client, error) {
	m.mu.RLock()
	client, exists := m.clients[sessionID]
	m.mu.RUnlock()

	if exists {
		return client, nil
	}

	sshClient, err := m.getSSHClient(sessionID)
	if err != nil {
		return nil, err
	}

	client, err = sftp.NewClient(sshClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create sftp client: %w", err)
	}

	m.mu.Lock()
	m.clients[sessionID] = client
	m.mu.Unlock()

	return client, nil
}

func (m *SFTPManager) List(sessionID, path string) ([]types.FileInfo, error) {
	client, err := m.getOrCreateClient(sessionID)
	if err != nil {
		return nil, err
	}

	entries, err := client.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read dir: %w", err)
	}

	var files []types.FileInfo
	for _, entry := range entries {
		info := types.FileInfo{
			Name:    entry.Name(),
			Size:    entry.Size(),
			Mode:    entry.Mode(),
			ModTime: entry.ModTime(),
			IsDir:   entry.IsDir(),
			Path:    filepath.Join(path, entry.Name()),
		}
		files = append(files, info)
	}

	return files, nil
}

func (m *SFTPManager) Download(sessionID, remotePath, localPath string, progress chan float64) error {
	client, err := m.getOrCreateClient(sessionID)
	if err != nil {
		return err
	}

	remoteFile, err := client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer remoteFile.Close()

	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	info, err := client.Stat(remotePath)
	if err != nil {
		return err
	}

	totalSize := info.Size()
	var transferred int64

	buffer := make([]byte, 32*1024)
	for {
		n, readErr := remoteFile.Read(buffer)
		if n > 0 {
			_, writeErr := localFile.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write: %w", writeErr)
			}
			transferred += int64(n)
			if progress != nil && totalSize > 0 {
				progress <- float64(transferred) / float64(totalSize) * 100
			}
		}
		if readErr != nil {
			break
		}
	}

	logger.GetLogger().Info("file downloaded",
		zap.String("session_id", sessionID),
		zap.String("remote", remotePath),
		zap.String("local", localPath),
	)

	return nil
}

func (m *SFTPManager) Upload(sessionID, localPath, remotePath string, progress chan float64) error {
	client, err := m.getOrCreateClient(sessionID)
	if err != nil {
		return err
	}

	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer localFile.Close()

	info, err := localFile.Stat()
	if err != nil {
		return err
	}

	remoteFile, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}
	defer remoteFile.Close()

	totalSize := info.Size()
	var transferred int64

	buffer := make([]byte, 32*1024)
	for {
		n, readErr := localFile.Read(buffer)
		if n > 0 {
			_, writeErr := remoteFile.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write: %w", writeErr)
			}
			transferred += int64(n)
			if progress != nil && totalSize > 0 {
				progress <- float64(transferred) / float64(totalSize) * 100
			}
		}
		if readErr != nil {
			break
		}
	}

	logger.GetLogger().Info("file uploaded",
		zap.String("session_id", sessionID),
		zap.String("local", localPath),
		zap.String("remote", remotePath),
	)

	return nil
}

func (m *SFTPManager) Delete(sessionID, path string) error {
	client, err := m.getOrCreateClient(sessionID)
	if err != nil {
		return err
	}

	info, err := client.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat: %w", err)
	}

	if info.IsDir() {
		err = client.RemoveDirectory(path)
	} else {
		err = client.Remove(path)
	}

	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	logger.GetLogger().Info("file deleted",
		zap.String("session_id", sessionID),
		zap.String("path", path),
	)

	return nil
}

func (m *SFTPManager) Rename(sessionID, oldPath, newPath string) error {
	client, err := m.getOrCreateClient(sessionID)
	if err != nil {
		return err
	}

	err = client.Rename(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("failed to rename: %w", err)
	}

	logger.GetLogger().Info("file renamed",
		zap.String("session_id", sessionID),
		zap.String("old_path", oldPath),
		zap.String("new_path", newPath),
	)

	return nil
}

func (m *SFTPManager) Mkdir(sessionID, path string) error {
	client, err := m.getOrCreateClient(sessionID)
	if err != nil {
		return err
	}

	err = client.MkdirAll(path)
	if err != nil {
		return fmt.Errorf("failed to mkdir: %w", err)
	}

	logger.GetLogger().Info("directory created",
		zap.String("session_id", sessionID),
		zap.String("path", path),
	)

	return nil
}

func (m *SFTPManager) ReadFile(sessionID, path string) (string, error) {
	client, err := m.getOrCreateClient(sessionID)
	if err != nil {
		return "", err
	}

	file, err := client.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

func (m *SFTPManager) WriteFile(sessionID, path, content string) error {
	client, err := m.getOrCreateClient(sessionID)
	if err != nil {
		return err
	}

	file, err := client.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	logger.GetLogger().Info("file written",
		zap.String("session_id", sessionID),
		zap.String("path", path),
	)

	return nil
}

func (m *SFTPManager) CloseClient(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if client, exists := m.clients[sessionID]; exists {
		client.Close()
		delete(m.clients, sessionID)
	}
}

func (m *SFTPManager) GetFileSize(sessionID, path string) (int64, error) {
	client, err := m.getOrCreateClient(sessionID)
	if err != nil {
		return 0, err
	}

	info, err := client.Stat(path)
	if err != nil {
		return 0, fmt.Errorf("failed to stat file: %w", err)
	}

	return info.Size(), nil
}

func (m *SFTPManager) ResumeUploadFromOffset(sessionID, localPath, remotePath string, offset, totalSize int64, progress chan float64) error {
	client, err := m.getOrCreateClient(sessionID)
	if err != nil {
		return err
	}

	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer localFile.Close()

	_, err = localFile.Seek(offset, 0)
	if err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	remoteFile, err := client.OpenFile(remotePath, os.O_RDWR|os.O_CREATE)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer remoteFile.Close()

	var transferred int64

	buffer := make([]byte, 32*1024)
	for {
		n, readErr := localFile.Read(buffer)
		if n > 0 {
			_, writeErr := remoteFile.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write: %w", writeErr)
			}
			transferred += int64(n)
			if progress != nil && totalSize > 0 {
				progress <- float64(offset+transferred) / float64(totalSize) * 100
			}
		}
		if readErr != nil {
			break
		}
	}

	logger.GetLogger().Info("file resumed uploaded",
		zap.String("session_id", sessionID),
		zap.String("local", localPath),
		zap.String("remote", remotePath),
		zap.Int64("offset", offset),
	)

	return nil
}

func (m *SFTPManager) ResumeDownloadFromOffset(sessionID, remotePath, localPath string, offset int64, progress chan float64) error {
	client, err := m.getOrCreateClient(sessionID)
	if err != nil {
		return err
	}

	remoteFile, err := client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer remoteFile.Close()

	localFile, err := os.OpenFile(localPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer localFile.Close()

	_, err = localFile.Seek(offset, 0)
	if err != nil {
		return fmt.Errorf("failed to seek: %w", err)
	}

	info, err := client.Stat(remotePath)
	if err != nil {
		return err
	}

	totalSize := info.Size()
	var transferred int64

	buffer := make([]byte, 32*1024)
	for {
		n, readErr := remoteFile.Read(buffer)
		if n > 0 {
			_, writeErr := localFile.Write(buffer[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write: %w", writeErr)
			}
			transferred += int64(n)
			if progress != nil && totalSize > 0 {
				progress <- float64(offset+transferred) / float64(totalSize) * 100
			}
		}
		if readErr != nil {
			break
		}
	}

	logger.GetLogger().Info("file resumed downloaded",
		zap.String("session_id", sessionID),
		zap.String("remote", remotePath),
		zap.String("local", localPath),
		zap.Int64("offset", offset),
	)

	return nil
}
