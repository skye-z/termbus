// Package main provides tunnel management examples for Termbus
//
// This example demonstrates how to:
// - Initialize the tunnel manager
// - Create local port forwarding tunnels
// - Create remote port forwarding tunnels
// - Create dynamic SOCKS5 proxy tunnels
// - Start, stop, and delete tunnels
// - Monitor tunnel status
//
// Usage:
//
//	go run examples/tunnel_example.go <session-id>
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/logger"
	"github.com/termbus/termbus/internal/session"
	"github.com/termbus/termbus/internal/ssh"
	"github.com/termbus/termbus/internal/tunnel"
	"github.com/termbus/termbus/pkg/types"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run examples/tunnel_example.go <session-id>")
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

	eventBus := eventBus.New()

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
	tunnelMgr := tunnel.NewTunnelManager(sessionMgr)

	fmt.Println("Termbus Tunnel Management Examples")
	fmt.Println("=================================\n")

	s, err := sessionMgr.GetSession(sessionID)
	if err != nil {
		fmt.Printf("Failed to get session: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Session: %s\n", s.ID)
	fmt.Printf("Host: %s@%s:%d\n\n", s.HostConfig.User, s.HostConfig.HostName, s.HostConfig.Port)

	exampleLocalForward(tunnelMgr, sessionID)
	exampleRemoteForward(tunnelMgr, sessionID)
	exampleDynamicForward(tunnelMgr, sessionID)
	exampleListTunnels(tunnelMgr, sessionID)
}

func exampleLocalForward(manager *tunnel.TunnelManager, sessionID string) {
	fmt.Println("Example: Local Port Forwarding")
	fmt.Println("------------------------------\n")

	forwardTunnel := &types.ForwardTunnel{
		Type:       types.ForwardTypeLocal,
		LocalAddr:  "127.0.0.1:8080",
		RemoteAddr: "127.0.0.1:80",
		SessionID:  sessionID,
		AutoStart:  false,
	}

	err := manager.CreateTunnel(sessionID, forwardTunnel)
	if err != nil {
		fmt.Printf("Failed to create tunnel: %v\n", err)
		return
	}

	fmt.Println("✓ Tunnel created")
	fmt.Printf("  ID: %s\n", forwardTunnel.ID)
	fmt.Printf("  Type: %s\n", forwardTunnel.Type)
	fmt.Printf("  Local: %s -> Remote: %s\n", forwardTunnel.LocalAddr, forwardTunnel.RemoteAddr)

	fmt.Println("Starting tunnel...")
	err = manager.StartTunnel(forwardTunnel.ID)
	if err != nil {
		fmt.Printf("Failed to start tunnel: %v\n", err)
		return
	}

	tunnel, _ := manager.GetTunnel(forwardTunnel.ID)
	fmt.Printf("✓ Tunnel started (Status: %s)\n", tunnel.Status)

	time.Sleep(2 * time.Second)

	fmt.Println("Stopping tunnel...")
	manager.StopTunnel(forwardTunnel.ID)
	fmt.Println("✓ Tunnel stopped")

	manager.DeleteTunnel(forwardTunnel.ID)
	fmt.Println("✓ Tunnel deleted\n")
}

func exampleRemoteForward(manager *tunnel.TunnelManager, sessionID string) {
	fmt.Println("Example: Remote Port Forwarding")
	fmt.Println("-------------------------------\n")

	forwardTunnel := &types.ForwardTunnel{
		Type:       types.ForwardTypeRemote,
		LocalAddr:  "127.0.0.1:8081",
		RemoteAddr: "0.0.0.0:8081",
		SessionID:  sessionID,
		AutoStart:  false,
	}

	err := manager.CreateTunnel(sessionID, forwardTunnel)
	if err != nil {
		fmt.Printf("Failed to create tunnel: %v\n", err)
		return
	}

	fmt.Println("✓ Tunnel created")
	fmt.Printf("  ID: %s\n", forwardTunnel.ID)
	fmt.Printf("  Type: %s\n", forwardTunnel.Type)
	fmt.Printf("  Local: %s <- Remote: %s\n", forwardTunnel.LocalAddr, forwardTunnel.RemoteAddr)

	fmt.Println("Starting tunnel...")
	err = manager.StartTunnel(forwardTunnel.ID)
	if err != nil {
		fmt.Printf("Failed to start tunnel: %v\n", err)
		return
	}

	tunnel, _ := manager.GetTunnel(forwardTunnel.ID)
	fmt.Printf("✓ Tunnel started (Status: %s)\n", tunnel.Status)

	time.Sleep(2 * time.Second)

	fmt.Println("Stopping tunnel...")
	manager.StopTunnel(forwardTunnel.ID)
	fmt.Println("✓ Tunnel stopped")

	manager.DeleteTunnel(forwardTunnel.ID)
	fmt.Println("✓ Tunnel deleted\n")
}

func exampleDynamicForward(manager *tunnel.TunnelManager, sessionID string) {
	fmt.Println("Example: Dynamic SOCKS5 Proxy")
	fmt.Println("--------------------------------\n")

	forwardTunnel := &types.ForwardTunnel{
		Type:      types.ForwardTypeDynamic,
		LocalAddr: "127.0.0.1:1080",
		SessionID: sessionID,
		AutoStart: false,
	}

	err := manager.CreateTunnel(sessionID, forwardTunnel)
	if err != nil {
		fmt.Printf("Failed to create tunnel: %v\n", err)
		return
	}

	fmt.Println("✓ Tunnel created")
	fmt.Printf("  ID: %s\n", forwardTunnel.ID)
	fmt.Printf("  Type: %s (SOCKS5 Proxy)\n", forwardTunnel.Type)
	fmt.Printf("  Listen on: %s\n", forwardTunnel.LocalAddr)

	fmt.Println("Starting tunnel...")
	err = manager.StartTunnel(forwardTunnel.ID)
	if err != nil {
		fmt.Printf("Failed to start tunnel: %v\n", err)
		return
	}

	tunnel, _ := manager.GetTunnel(forwardTunnel.ID)
	fmt.Printf("✓ Tunnel started (Status: %s)\n", tunnel.Status)
	fmt.Println("  SOCKS5 proxy is now available on", forwardTunnel.LocalAddr)
	fmt.Println("  Configure your application to use", forwardTunnel.LocalAddr)

	time.Sleep(2 * time.Second)

	fmt.Println("Stopping tunnel...")
	manager.StopTunnel(forwardTunnel.ID)
	fmt.Println("✓ Tunnel stopped")

	manager.DeleteTunnel(forwardTunnel.ID)
	fmt.Println("✓ Tunnel deleted\n")
}

func exampleListTunnels(manager *tunnel.TunnelManager, sessionID string) {
	fmt.Println("Example: List Tunnels")
	fmt.Println("-------------------\n")

	forwardTunnel1 := &types.ForwardTunnel{
		Type:       types.ForwardTypeLocal,
		LocalAddr:  "127.0.0.1:9080",
		RemoteAddr: "127.0.0.1:90",
		SessionID:  sessionID,
	}

	forwardTunnel2 := &types.ForwardTunnel{
		Type:       types.ForwardTypeRemote,
		LocalAddr:  "127.0.0.1:9081",
		RemoteAddr: "0.0.0.0:9081",
		SessionID:  sessionID,
	}

	forwardTunnel3 := &types.ForwardTunnel{
		Type:      types.ForwardTypeDynamic,
		LocalAddr: "127.0.0.1:1081",
		SessionID: sessionID,
	}

	manager.CreateTunnel(sessionID, forwardTunnel1)
	manager.CreateTunnel(sessionID, forwardTunnel2)
	manager.CreateTunnel(sessionID, forwardTunnel3)

	tunnels := manager.ListTunnels(sessionID)

	fmt.Printf("✓ Found %d tunnels\n\n", len(tunnels))

	for _, tunnel := range tunnels {
		var typeIcon string
		switch tunnel.Type {
		case types.ForwardTypeLocal:
			typeIcon = "🔀"
		case types.ForwardTypeRemote:
			typeIcon = "📡"
		case types.ForwardTypeDynamic:
			typeIcon = "🌐"
		}

		var statusIcon string
		switch tunnel.Status {
		case types.TunnelStatusStopped:
			statusIcon = "⏹"
		case types.TunnelStatusRunning:
			statusIcon = "▶️"
		case types.TunnelStatusError:
			statusIcon = "❌"
		}

		fmt.Printf("%s %s [%s]\n", typeIcon, statusIcon, tunnel.ID)
		fmt.Printf("  Type: %s\n", tunnel.Type)
		fmt.Printf("  Local: %s\n", tunnel.LocalAddr)
		if tunnel.Type != types.ForwardTypeDynamic {
			fmt.Printf("  Remote: %s\n", tunnel.RemoteAddr)
		}
		fmt.Printf("  Status: %s\n", tunnel.Status)
		fmt.Printf("  AutoStart: %v\n", tunnel.AutoStart)
		fmt.Println()
	}

	for _, tunnel := range tunnels {
		manager.DeleteTunnel(tunnel.ID)
	}
	fmt.Println("✓ All tunnels cleaned up\n")
}
