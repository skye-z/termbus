// Package main provides SFTP file management examples for Termbus
//
// This example demonstrates how to:
// - Initialize the SFTP manager
// - List files and directories
// - Upload files with progress tracking
// - Download files with progress tracking
// - Create, delete, rename files and directories
// - Read and edit remote files
//
// Usage:
//
//	go run examples/sftp_example.go <session-id>
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/logger"
	"github.com/termbus/termbus/internal/session"
	"github.com/termbus/termbus/internal/sftp"
	"github.com/termbus/termbus/internal/ssh"
	"github.com/termbus/termbus/pkg/types"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run examples/sftp_example.go <session-id>")
		os.Exit(1)
	}

	sessionID := os.Args[1]

	if err := logger.Init(&logger.LogConfig{
		Level:      "info",
		OutputPath: "",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	eventBus := eventbus.New()

	cfg := &ssh.SSHConfig{
		ConfigPath:        "",
		KnownHostsPath:    "",
		DefaultTimeout:    30,
		KeepaliveEnabled:  true,
		KeepaliveInterval: 60,
	}

	sshManager := ssh.NewSSHManager(cfg, eventBus)
	sshPool := session.NewSSHConnectionPool()
	sessionMgr := session.New(eventBus, sshPool, nil)
	sftpMgr := sftp.NewSFTPManager(sessionMgr)

	fmt.Println("Termbus SFTP File Management Examples")
	fmt.Println("======================================\n")

	s, err := sessionMgr.GetSession(sessionID)
	if err != nil {
		fmt.Printf("Failed to get session: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Session: %s\n", s.ID)
	fmt.Printf("Host: %s@%s:%d\n\n", s.HostConfig.User, s.HostConfig.HostName, s.HostConfig.Port)

	exampleListFiles(sftpMgr, sessionID)
	exampleUploadFile(sftpMgr, sessionID)
	exampleDownloadFile(sftpMgr, sessionID)
	exampleCreateDirectory(sftpMgr, sessionID)
	exampleDeleteFile(sftpMgr, sessionID)
	exampleRenameFile(sftpMgr, sessionID)
	exampleReadFile(sftpMgr, sessionID)
	exampleWriteFile(sftpMgr, sessionID)
}

func exampleListFiles(manager *sftp.SFTPManager, sessionID string) {
	fmt.Println("Example: List Files")
	fmt.Println("------------------\n")

	files, err := manager.List(sessionID, "/tmp")
	if err != nil {
		fmt.Printf("Failed to list files: %v\n", err)
		return
	}

	fmt.Printf("✓ Found %d items in /tmp\n\n", len(files))

	for _, file := range files {
		icon := "📄"
		if file.IsDir {
			icon = "📁"
		}
		fmt.Printf("%s %-40s %10d %s\n", icon, file.Name, file.Size, file.ModTime.Format("2006-01-02 15:04:05"))
	}
	fmt.Println()
}

func exampleUploadFile(manager *sftp.SFTPManager, sessionID string) {
	fmt.Println("Example: Upload File")
	fmt.Println("--------------------\n")

	tempDir := os.TempDir()
	localFile := tempDir + "/test_upload.txt"
	content := "Hello from Termbus SFTP!"

	if err := os.WriteFile(localFile, []byte(content), 0644); err != nil {
		fmt.Printf("Failed to create local file: %v\n", err)
		return
	}
	defer os.Remove(localFile)

	remotePath := fmt.Sprintf("/tmp/termbus_upload_%d.txt", time.Now().Unix())
	progress := make(chan float64, 10)

	done := make(chan bool)
	go func() {
		for p := range progress {
			fmt.Printf("\r  Upload progress: %.2f%%", p)
		}
		done <- true
	}()

	err := manager.Upload(sessionID, localFile, remotePath, progress)
	close(progress)
	<-done
	fmt.Println()

	if err != nil {
		fmt.Printf("Failed to upload: %v\n", err)
		return
	}

	fmt.Println("✓ File uploaded successfully")
	fmt.Printf("  Local:  %s\n", localFile)
	fmt.Printf("  Remote: %s\n", remotePath)

	_ = manager.Delete(sessionID, remotePath)
	fmt.Println("✓ Remote file cleaned up\n")
}

func exampleDownloadFile(manager *sftp.SFTPManager, sessionID string) {
	fmt.Println("Example: Download File")
	fmt.Println("----------------------\n")

	remotePath := "/etc/hostname"
	tempDir := os.TempDir()
	localFile := tempDir + "/hostname"

	progress := make(chan float64, 10)

	done := make(chan bool)
	go func() {
		for p := range progress {
			fmt.Printf("\r  Download progress: %.2f%%", p)
		}
		done <- true
	}()

	err := manager.Download(sessionID, remotePath, localFile, progress)
	close(progress)
	<-done
	fmt.Println()

	if err != nil {
		fmt.Printf("Failed to download: %v\n", err)
		return
	}

	fmt.Println("✓ File downloaded successfully")
	fmt.Printf("  Remote: %s\n", remotePath)
	fmt.Printf("  Local:  %s\n", localFile)

	content, _ := os.ReadFile(localFile)
	fmt.Printf("  Content: %s\n", string(content))

	os.Remove(localFile)
	fmt.Println("✓ Local file cleaned up\n")
}

func exampleCreateDirectory(manager *sftp.SFTPManager, sessionID string) {
	fmt.Println("Example: Create Directory")
	fmt.Println("-------------------------\n")

	testDir := fmt.Sprintf("/tmp/termbus_test_%d", time.Now().Unix())

	err := manager.Mkdir(sessionID, testDir)
	if err != nil {
		fmt.Printf("Failed to create directory: %v\n", err)
		return
	}

	fmt.Println("✓ Directory created successfully")
	fmt.Printf("  Path: %s\n", testDir)

	_ = manager.Delete(sessionID, testDir)
	fmt.Println("✓ Directory cleaned up\n")
}

func exampleDeleteFile(manager *sftp.SFTPManager, sessionID string) {
	fmt.Println("Example: Delete File")
	fmt.Println("-------------------\n")

	testFile := fmt.Sprintf("/tmp/termbus_delete_%d.txt", time.Now().Unix())
	testContent := "test content"

	_ = manager.WriteFile(sessionID, testFile, testContent)

	err := manager.Delete(sessionID, testFile)
	if err != nil {
		fmt.Printf("Failed to delete: %v\n", err)
		return
	}

	fmt.Println("✓ File deleted successfully")
	fmt.Printf("  Path: %s\n", testFile)
}

func exampleRenameFile(manager *sftp.SFTPManager, sessionID string) {
	fmt.Println("Example: Rename File")
	fmt.Println("-------------------\n")

	oldPath := fmt.Sprintf("/tmp/termbus_old_%d.txt", time.Now().Unix())
	newPath := fmt.Sprintf("/tmp/termbus_new_%d.txt", time.Now().Unix())
	testContent := "test content"

	_ = manager.WriteFile(sessionID, oldPath, testContent)

	err := manager.Rename(sessionID, oldPath, newPath)
	if err != nil {
		fmt.Printf("Failed to rename: %v\n", err)
		return
	}

	fmt.Println("✓ File renamed successfully")
	fmt.Printf("  Old: %s\n", oldPath)
	fmt.Printf("  New: %s\n", newPath)

	_ = manager.Delete(sessionID, newPath)
	fmt.Println("✓ File cleaned up\n")
}

func exampleReadFile(manager *sftp.SFTPManager, sessionID string) {
	fmt.Println("Example: Read File")
	fmt.Println("-----------------\n")

	content, err := manager.ReadFile(sessionID, "/etc/hostname")
	if err != nil {
		fmt.Printf("Failed to read file: %v\n", err)
		return
	}

	fmt.Println("✓ File read successfully")
	fmt.Printf("  Path: /etc/hostname\n")
	fmt.Printf("  Content: %s\n", content)
}

func exampleWriteFile(manager *sftp.SFTPManager, sessionID string) {
	fmt.Println("Example: Write File")
	fmt.Println("-----------------\n")

	testPath := fmt.Sprintf("/tmp/termbus_write_%d.txt", time.Now().Unix())
	testContent := "Hello from Termbus!\nWritten at " + time.Now().Format(time.RFC3339)

	err := manager.WriteFile(sessionID, testPath, testContent)
	if err != nil {
		fmt.Printf("Failed to write: %v\n", err)
		return
	}

	fmt.Println("✓ File written successfully")
	fmt.Printf("  Path: %s\n", testPath)
	fmt.Printf("  Content: %s\n", testContent)

	_ = manager.Delete(sessionID, testPath)
	fmt.Println("✓ File cleaned up\n")
}
