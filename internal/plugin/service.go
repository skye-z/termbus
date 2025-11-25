package plugin

import (
	"context"
	"fmt"

	"github.com/termbus/termbus/internal/logger"
	"go.uber.org/zap"
)

type InitResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type PluginExecuteResponse struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Error    string `json:"error"`
}

type StopResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type InfoResponse struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
}

type ManifestResponse struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	Permissions  []string          `json:"permissions"`
	Commands     []string          `json:"commands"`
	ConfigSchema map[string]string `json:"config_schema"`
}

type PluginService struct {
	version string
}

func NewPluginService(version string) *PluginService {
	return &PluginService{version: version}
}

func (s *PluginService) Init(ctx context.Context, args map[string]interface{}) (*InitResponse, error) {
	logger.GetLogger().Info("plugin init called",
		zap.String("version", s.version),
		zap.Any("args", args),
	)

	return &InitResponse{
		Success: true,
		Error:   "",
	}, nil
}

func (s *PluginService) Execute(ctx context.Context, args map[string]interface{}) (*PluginExecuteResponse, error) {
	command, _ := args["command"].(string)
	logger.GetLogger().Info("plugin execute called",
		zap.String("command", command),
	)

	return &PluginExecuteResponse{
		ExitCode: 0,
		Stdout:   fmt.Sprintf("Executed: %s", command),
		Stderr:   "",
		Error:    "",
	}, nil
}

func (s *PluginService) Stop(ctx context.Context, args map[string]interface{}) (*StopResponse, error) {
	logger.GetLogger().Info("plugin stop called")

	return &StopResponse{
		Success: true,
		Error:   "",
	}, nil
}

func (s *PluginService) Info(ctx context.Context) (*InfoResponse, error) {
	return &InfoResponse{
		Name:        "Termbus Plugin",
		Version:     s.version,
		Description: "Termbus plugin base service",
		Author:      "termbus",
	}, nil
}

func (s *PluginService) Manifest(ctx context.Context) (*ManifestResponse, error) {
	return &ManifestResponse{
		Name:         "Termbus Plugin",
		Version:      s.version,
		Description:  "Termbus plugin base service",
		Author:       "termbus",
		Permissions:  []string{"ssh.execute", "sftp.read"},
		Commands:     []string{"test"},
		ConfigSchema: map[string]string{"option1": "string"},
	}, nil
}
