package api

import (
	"context"
	"io"
)

// Plugin defines the plugin interface.
type Plugin interface {
	Init(ctx context.Context, config map[string]string) error
	Execute(ctx context.Context, cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error)
	Stop(ctx context.Context) error
	Name() string
	Version() string
	Description() string
	Author() string
	Permissions() []string
	Commands() []string
}

// BasePlugin provides default plugin fields.
type BasePlugin struct {
	Name        string
	Version     string
	Description string
	Author      string
	Config      map[string]string
}

// Init stores configuration.
func (p *BasePlugin) Init(ctx context.Context, config map[string]string) error {
	p.Config = config
	return nil
}

// Stop performs cleanup.
func (p *BasePlugin) Stop(ctx context.Context) error {
	return nil
}
