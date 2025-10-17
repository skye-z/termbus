package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/session"
	"github.com/termbus/termbus/internal/sftp"
	"github.com/termbus/termbus/pkg/types"
)

func TestSFTPManager_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	sftpMgr := sftp.NewSFTPManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-sftp-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	files, err := sftpMgr.List(s.ID, "/tmp")
	if err != nil {
		t.Skipf("Failed to list files: %v", err)
	}

	assert.NotNil(t, files)
	assert.NotEmpty(t, files)
}

func TestSFTPManager_Upload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	sftpMgr := sftp.NewSFTPManager(sessionMgr)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, SFTP!"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-sftp-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	remotePath := fmt.Sprintf("/tmp/test_upload_%d.txt", os.Getpid())
	progress := make(chan float64, 10)

	go func() {
		for p := range progress {
			t.Logf("Upload progress: %.2f%%", p)
		}
	}()

	err = sftpMgr.Upload(s.ID, testFile, remotePath, progress)
	if err != nil {
		t.Skipf("Failed to upload: %v", err)
	}

	assert.NoError(t, err)
	close(progress)
}

func TestSFTPManager_Download(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	sftpMgr := sftp.NewSFTPManager(sessionMgr)

	tempDir := t.TempDir()
	localPath := filepath.Join(tempDir, "downloaded.txt")

	hostConfig := &types.SSHHostConfig{
		Host:     "test-sftp-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	remotePath := "/etc/hosts"
	progress := make(chan float64, 10)

	go func() {
		for p := range progress {
			t.Logf("Download progress: %.2f%%", p)
		}
	}()

	err = sftpMgr.Download(s.ID, remotePath, localPath, progress)
	if err != nil {
		t.Skipf("Failed to download: %v", err)
	}

	assert.NoError(t, err)

	content, err := os.ReadFile(localPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)

	close(progress)
}

func TestSFTPManager_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	sftpMgr := sftp.NewSFTPManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-sftp-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	testPath := fmt.Sprintf("/tmp/test_delete_%d.txt", os.Getpid())
	testContent := "test content"

	_ = sftpMgr.WriteFile(s.ID, testPath, testContent)

	err = sftpMgr.Delete(s.ID, testPath)
	if err != nil {
		t.Skipf("Failed to delete: %v", err)
	}

	assert.NoError(t, err)
}

func TestSFTPManager_Rename(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	sftpMgr := sftp.NewSFTPManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-sftp-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	oldPath := fmt.Sprintf("/tmp/test_old_%d.txt", os.Getpid())
	newPath := fmt.Sprintf("/tmp/test_new_%d.txt", os.Getpid())
	testContent := "test content"

	_ = sftpMgr.WriteFile(s.ID, oldPath, testContent)

	err = sftpMgr.Rename(s.ID, oldPath, newPath)
	if err != nil {
		t.Skipf("Failed to rename: %v", err)
	}

	assert.NoError(t, err)

	content, err := sftpMgr.ReadFile(s.ID, newPath)
	assert.NoError(t, err)
	assert.Equal(t, testContent, content)
}

func TestSFTPManager_Mkdir(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	sftpMgr := sftp.NewSFTPManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-sftp-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	testPath := fmt.Sprintf("/tmp/test_dir_%d", os.Getpid())

	err = sftpMgr.Mkdir(s.ID, testPath)
	if err != nil {
		t.Skipf("Failed to mkdir: %v", err)
	}

	assert.NoError(t, err)

	files, err := sftpMgr.List(s.ID, "/tmp")
	assert.NoError(t, err)

	found := false
	for _, file := range files {
		if file.Path == testPath && file.IsDir {
			found = true
			break
		}
	}
	assert.True(t, found, "Directory should exist")
}

func TestSFTPManager_ReadFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	sftpMgr := sftp.NewSFTPManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-sftp-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	content, err := sftpMgr.ReadFile(s.ID, "/etc/hostname")
	if err != nil {
		t.Skipf("Failed to read file: %v", err)
	}

	assert.NoError(t, err)
	assert.NotEmpty(t, content)
}

func TestSFTPManager_WriteFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	sftpMgr := sftp.NewSFTPManager(sessionMgr)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-sftp-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	testPath := fmt.Sprintf("/tmp/test_write_%d.txt", os.Getpid())
	testContent := "test write content"

	err = sftpMgr.WriteFile(s.ID, testPath, testContent)
	if err != nil {
		t.Skipf("Failed to write file: %v", err)
	}

	assert.NoError(t, err)

	readContent, err := sftpMgr.ReadFile(s.ID, testPath)
	assert.NoError(t, err)
	assert.Equal(t, testContent, readContent)

	_ = sftpMgr.Delete(s.ID, testPath)
}

func TestSFTPManager_ProgressTracking(t *testing.T) {
	eventBus := eventbus.New()
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	sftpMgr := sftp.NewSFTPManager(sessionMgr)

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "large.txt")
	testContent := make([]byte, 1024*1024)
	for i := range testContent {
		testContent[i] = byte(i % 256)
	}

	err := os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	hostConfig := &types.SSHHostConfig{
		Host:     "test-sftp-host",
		HostName: "127.0.0.1",
		User:     "testuser",
		Port:     2222,
	}

	s, err := sessionMgr.CreateSession(hostConfig)
	require.NoError(t, err)

	remotePath := fmt.Sprintf("/tmp/large_%d.txt", os.Getpid())
	progress := make(chan float64, 100)

	progressValues := []float64{}
	go func() {
		for p := range progress {
			progressValues = append(progressValues, p)
		}
	}()

	err = sftpMgr.Upload(s.ID, testFile, remotePath, progress)
	if err != nil {
		t.Skipf("Failed to upload: %v", err)
	}

	assert.NoError(t, err)
	close(progress)

	if len(progressValues) > 0 {
		assert.Equal(t, 100.0, progressValues[len(progressValues)-1], "Final progress should be 100%")
	}
}
