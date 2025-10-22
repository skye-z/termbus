// Package main provides basic SSH connection examples for Termbus
//
// This example demonstrates how to:
// - Initialize the SSH manager
// - Connect to a remote host using password authentication
// - Connect to a remote host using public key authentication
// - Use SSH agent for authentication
// - Parse SSH config files
// - Manage connection pooling
//
// Usage:
//
//	go run examples/ssh_example.go
package main

import (
	"fmt"
	"os"

	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/logger"
	"github.com/termbus/termbus/internal/ssh"
)

func main() {
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

	fmt.Println("Termbus SSH Connection Examples")
	fmt.Println("================================\n")

	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run examples/ssh_example.go <example>")
		fmt.Println("\nExamples:")
		fmt.Println("  password    - Password authentication example")
		fmt.Println("  pubkey      - Public key authentication example")
		fmt.Println("  agent       - SSH agent authentication example")
		fmt.Println("  config      - SSH config parsing example")
		fmt.Println("  pooling     - Connection pooling example")
		os.Exit(0)
	}

	example := os.Args[1]

	switch example {
	case "password":
		examplePasswordAuth(sshManager)
	case "pubkey":
		examplePubkeyAuth(sshManager)
	case "agent":
		exampleAgentAuth(sshManager)
	case "config":
		exampleConfigParsing(sshManager)
	case "pooling":
		exampleConnectionPooling(sshManager)
	default:
		fmt.Printf("Unknown example: %s\n", example)
		os.Exit(1)
	}
}

func examplePasswordAuth(manager *ssh.SSHManager) {
	fmt.Println("Example: Password Authentication")
	fmt.Println("-----------------------------\n")

	hostConfig := &ssh.HostConfig{
		Host:     "example-host",
		HostName: "127.0.0.1",
		User:     "username",
		Port:     22,
		Password: "your-password",
	}

	client, err := manager.Connect(hostConfig, hostConfig.Password)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer client.Close()

	fmt.Println("✓ Connected successfully")
	fmt.Printf("✓ Session established with %s@%s:%d\n", hostConfig.User, hostConfig.HostName, hostConfig.Port)

	manager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)
	fmt.Println("✓ Disconnected")
}

func examplePubkeyAuth(manager *ssh.SSHManager) {
	fmt.Println("Example: Public Key Authentication")
	fmt.Println("----------------------------------\n")

	homeDir, _ := os.UserHomeDir()
	keyPath := homeDir + "/.ssh/id_rsa"

	hostConfig := &ssh.HostConfig{
		Host:         "example-host",
		HostName:     "127.0.0.1",
		User:         "username",
		Port:         22,
		IdentityFile: keyPath,
		Password:     "", // Empty for public key auth
	}

	client, err := manager.Connect(hostConfig, "")
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer client.Close()

	fmt.Println("✓ Connected successfully using public key")
	fmt.Printf("✓ Using identity file: %s\n", keyPath)

	manager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)
	fmt.Println("✓ Disconnected")
}

func exampleAgentAuth(manager *ssh.SSHManager) {
	fmt.Println("Example: SSH Agent Authentication")
	fmt.Println("--------------------------------\n")

	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSock == "" {
		fmt.Println("SSH_AUTH_SOCK environment variable not set")
		fmt.Println("Start SSH agent with: eval $(ssh-agent -s)")
		fmt.Println("Add key with: ssh-add ~/.ssh/id_rsa")
		return
	}

	fmt.Printf("✓ SSH agent socket: %s\n", sshAuthSock)

	hostConfig := &ssh.HostConfig{
		Host:     "example-host",
		HostName: "127.0.0.1",
		User:     "username",
		Port:     22,
	}

	client, err := manager.Connect(hostConfig, "")
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer client.Close()

	fmt.Println("✓ Connected successfully using SSH agent")
	fmt.Println("✓ Agent forwarding enabled")

	manager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)
	fmt.Println("✓ Disconnected")
}

func exampleConfigParsing(manager *ssh.SSHManager) {
	fmt.Println("Example: SSH Config Parsing")
	fmt.Println("-----------------------------\n")

	homeDir, _ := os.UserHomeDir()
	configPath := homeDir + "/.ssh/config"

	cfg := &ssh.SSHConfig{
		ConfigPath:        configPath,
		KnownHostsPath:    "",
		DefaultTimeout:    30,
		KeepaliveEnabled:  true,
		KeepaliveInterval: 60,
	}

	manager = ssh.NewSSHManager(cfg, manager.(*ssh.SSHManager).EventBus())

	configs, err := manager.ScanSSHConfigs()
	if err != nil {
		fmt.Printf("Failed to parse SSH config: %v\n", err)
		return
	}

	fmt.Printf("✓ Found %d hosts in SSH config\n", len(configs))
	fmt.Println()

	for _, config := range configs {
		fmt.Printf("Host: %s\n", config.Host)
		fmt.Printf("  HostName: %s\n", config.HostName)
		fmt.Printf("  User: %s\n", config.User)
		fmt.Printf("  Port: %d\n", config.Port)
		if config.ProxyJump != "" {
			fmt.Printf("  ProxyJump: %s\n", config.ProxyJump)
		}
		if config.ProxyCommand != "" {
			fmt.Printf("  ProxyCommand: %s\n", config.ProxyCommand)
		}
		fmt.Println()
	}
}

func exampleConnectionPooling(manager *ssh.SSHManager) {
	fmt.Println("Example: Connection Pooling")
	fmt.Println("---------------------------\n")

	hostConfig := &ssh.HostConfig{
		Host:     "example-host",
		HostName: "127.0.0.1",
		User:     "username",
		Port:     22,
		Password: "your-password",
	}

	fmt.Println("Creating first connection...")
	client1, err := manager.Connect(hostConfig, hostConfig.Password)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer client1.Close()
	fmt.Println("✓ First connection established")

	fmt.Println("Creating second connection (should reuse)...")
	client2, err := manager.Connect(hostConfig, hostConfig.Password)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		return
	}
	defer client2.Close()

	if client1 == client2 {
		fmt.Println("✓ Connection pooled and reused")
	} else {
		fmt.Println("✗ New connection created (pooling not working)")
	}

	manager.Disconnect(hostConfig.HostName, hostConfig.User, hostConfig.Port)
	fmt.Println("✓ Connection released from pool")
}
