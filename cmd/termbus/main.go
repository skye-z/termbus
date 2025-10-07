package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/termbus/termbus/internal/config"
	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/logger"
	"github.com/termbus/termbus/internal/session"
	"github.com/termbus/termbus/internal/ssh"
	"github.com/termbus/termbus/tui/views"
	"go.uber.org/zap"
)

var (
	version = "0.1.0"
	commit  = "dev"
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "termbus",
		Short:   "Terminal-First DevOps Tool",
		Long:    "Termbus - A Terminal-First DevOps Tool with SSH protocol support",
		Version: fmt.Sprintf("%s (commit: %s)", version, commit),
		Run:     run,
	}

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Termbus version %s (commit: %s)\n", version, commit)
		},
	})

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	cfgManager, err := config.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	cfg := cfgManager.Get()

	if err := logger.Init(&logger.LogConfig{
		Level:      cfg.Log.Level,
		OutputPath: cfg.Log.OutputPath,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	eventBus := eventbus.New()

	sshPool := session.NewSSHConnectionPool()
	store, err := session.NewSessionStore(cfg.General.DataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize session store: %v\n", err)
		os.Exit(1)
	}

	sessionManager := session.New(eventBus, sshPool, store)
	sshManager := ssh.NewSSHManager(&ssh.SSHConfig{
		ConfigPath:        cfg.SSH.ConfigPath,
		KnownHostsPath:    cfg.SSH.KnownHostsPath,
		DefaultTimeout:    cfg.SSH.DefaultTimeout,
		KeepaliveEnabled:  cfg.SSH.KeepaliveEnabled,
		KeepaliveInterval: cfg.SSH.KeepaliveInterval,
	}, eventBus)

	logger.GetLogger().Info("Termbus started",
		zap.String("version", version),
		zap.String("commit", commit),
	)

	app := views.NewApp(eventBus, sessionManager, sshManager, 120, 40)
	program := tea.NewProgram(app, tea.WithAltScreen())
	go func() {
		if err := program.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start TUI: %v\n", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	program.Quit()

	logger.GetLogger().Info("Termbus shutting down")
}
