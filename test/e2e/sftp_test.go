package e2e

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
	"github.com/termbus/termbus/internal/ssh"
	"github.com/termbus/termbus/pkg/types"
)

func TestE2E_SFTPList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	config := loadTestConfig(t)
	t.Logf("INPUT: Listing SFTP files in %s", config.TestPaths.RemoteTmp)
	eventBus := eventbus.New()

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

	sshPool := session.NewSSHConnectionPool()
	hostKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)
	sshPool.SetConnection(hostKey, client)

	sessionMgr := session.New(eventBus, sshPool, nil)
	sftpMgr := sftp.NewSFTPManager(sessionMgr)

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

	files, err := sftpMgr.List(s.ID, config.TestPaths.RemoteTmp)
	require.NoError(t, err)
	assert.NotNil(t, files)
	t.Logf("OUTPUT: Listed %d files", len(files))

	sessionMgr.DisconnectSession(s.ID)
	sessionMgr.DeleteSession(s.ID)
	sshManager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)
}

func TestE2E_SFTPUploadDownload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	config := loadTestConfig(t)
	t.Logf("INPUT: Testing SFTP upload/download on %s", config.Hosts.SSHServer.Hostname)
	eventBus := eventbus.New()

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

	sshPool := session.NewSSHConnectionPool()
	hostKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)
	sshPool.SetConnection(hostKey, client)

	sessionMgr := session.New(eventBus, sshPool, nil)
	sftpMgr := sftp.NewSFTPManager(sessionMgr)

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

	tempDir := t.TempDir()
	localFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello from E2E SFTP test!"
	err = os.WriteFile(localFile, []byte(testContent), 0644)
	require.NoError(t, err)

	remotePath := fmt.Sprintf("/tmp/e2e_test_%d.txt", os.Getpid())
	t.Logf("INPUT: Uploading file %s to %s", localFile, remotePath)

	err = sftpMgr.Upload(s.ID, localFile, remotePath, nil)
	require.NoError(t, err)
	t.Logf("OUTPUT: File uploaded successfully")

	content, err := sftpMgr.ReadFile(s.ID, remotePath)
	require.NoError(t, err)
	assert.Equal(t, testContent, content)
	t.Logf("OUTPUT: File content verified: %s", content)

	downloadPath := filepath.Join(tempDir, "downloaded.txt")
	t.Logf("INPUT: Downloading file from %s to %s", remotePath, downloadPath)

	err = sftpMgr.Download(s.ID, remotePath, downloadPath, nil)
	require.NoError(t, err)
	t.Logf("OUTPUT: File downloaded successfully")

	downloadedContent, err := os.ReadFile(downloadPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(downloadedContent))

	sftpMgr.Delete(s.ID, remotePath)

	sessionMgr.DisconnectSession(s.ID)
	sessionMgr.DeleteSession(s.ID)
	sshManager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)
}

func TestE2E_SFTPMkdirDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	config := loadTestConfig(t)
	eventBus := eventbus.New()

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

	sshPool := session.NewSSHConnectionPool()
	hostKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)
	sshPool.SetConnection(hostKey, client)

	sessionMgr := session.New(eventBus, sshPool, nil)
	sftpMgr := sftp.NewSFTPManager(sessionMgr)

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

	testDir := fmt.Sprintf("/tmp/e2e_test_dir_%d", os.Getpid())

	err = sftpMgr.Mkdir(s.ID, testDir)
	require.NoError(t, err)

	files, err := sftpMgr.List(s.ID, "/tmp")
	require.NoError(t, err)

	found := false
	for _, file := range files {
		if file.Path == testDir && file.IsDir {
			found = true
			break
		}
	}
	assert.True(t, found, "Directory should exist")

	err = sftpMgr.Delete(s.ID, testDir)
	require.NoError(t, err)

	sessionMgr.DisconnectSession(s.ID)
	sessionMgr.DeleteSession(s.ID)
	sshManager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)
}

func TestE2E_SFTPRename(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	config := loadTestConfig(t)
	eventBus := eventbus.New()

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

	sshPool := session.NewSSHConnectionPool()
	hostKey := fmt.Sprintf("%s@%s:%d", hostConfig.User, hostConfig.HostName, hostConfig.Port)
	sshPool.SetConnection(hostKey, client)

	sessionMgr := session.New(eventBus, sshPool, nil)
	sftpMgr := sftp.NewSFTPManager(sessionMgr)

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

	oldPath := fmt.Sprintf("/tmp/e2e_old_%d.txt", os.Getpid())
	newPath := fmt.Sprintf("/tmp/e2e_new_%d.txt", os.Getpid())
	testContent := "test rename content"

	err = sftpMgr.WriteFile(s.ID, oldPath, testContent)
	require.NoError(t, err)

	err = sftpMgr.Rename(s.ID, oldPath, newPath)
	require.NoError(t, err)

	content, err := sftpMgr.ReadFile(s.ID, newPath)
	require.NoError(t, err)
	assert.Equal(t, testContent, content)

	sftpMgr.Delete(s.ID, newPath)

	sessionMgr.DisconnectSession(s.ID)
	sessionMgr.DeleteSession(s.ID)
	sshManager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)
}
