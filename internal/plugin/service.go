package plugin

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/termbus/termbus/internal/logger"
	"go.uber.org/zap"
)

type PluginService struct {
	version string
}

// Serve 启动插件服务
func Serve(version string) {
	plugin.Serve(&PluginService{
		version: version,
	})
}

func (s *PluginService) Init(args map[string]interface{}) (struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}, error) {
	logger.GetLogger().Info("plugin init called",
		zap.String("version", s.version),
		zap.Any("args", args),
	)

	return struct {
		Success: true,
		Error:   "",
	}, nil
}

func (s *PluginService) Execute(args map[string]interface{}) (struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Error    string `json:"error"`
}, error) {
	command := args["command"].(string)
	logger.GetLogger().Info("plugin execute called",
		zap.String("command", command),
	)

	return struct {
		ExitCode: 0,
		Stdout:   fmt.Sprintf("Executed: %s", command),
		Stderr:   "",
		Error:    "",
	}, nil
}

func (s *PluginService) Stop(args map[string]interface{}) (struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}, error) {
	logger.GetLogger().Info("plugin stop called")

	return struct {
		Success: true,
		Error:   "",
	}, nil
}

func (s *PluginService) Info(args map[string]interface{}) (struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
}, error) {
	return struct {
		Name:        "Termbus Plugin",
		Version:     s.version,
		Description: "Termbus plugin base service",
		Author:      "termbus",
	}, nil
}

func (s *PluginService) Manifest(args map[string]interface{}) (struct {
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	Author      string          `json:"author"`
	Permissions []string        `json:"permissions"`
	Commands    []string        `json:"commands"`
	ConfigSchema map[string]schema `json:"config_schema"`
}, error) {
	return struct {
		Name:        "Termbus Plugin",
		Version:     s.version,
		Description: "Termbus plugin base service",
		Author:      "termbus",
		Permissions: []string{"ssh.execute", "sftp.read"},
		Commands:    []string{"test"},
		ConfigSchema: map[string]schema{
			"option1": {
				Type:        "string",
				Default:     "default",
				Description: "Option 1 description",
			},
		},
	}, nil
}

type schema struct {
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}
