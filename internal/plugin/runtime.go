package plugin

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	plugin "github.com/hashicorp/go-plugin"
	"github.com/termbus/termbus/internal/config"
	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/logger"
	pluginrpc "github.com/termbus/termbus/internal/plugin/grpc"
	"go.uber.org/zap"
)

// Plugin represents a loaded plugin instance.
type Plugin struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	Path      string            `json:"path"`
	Enabled   bool              `json:"enabled"`
	Config    map[string]string `json:"config"`
	Process   *exec.Cmd         `json:"-"`
	Client    *plugin.Client    `json:"-"`
	PID       int               `json:"pid"`
	StartedAt time.Time         `json:"started_at"`
}

// PluginRuntime manages plugin lifecycle and processes.
type PluginRuntime struct {
	plugins  map[string]*Plugin
	config   *config.GlobalConfig
	eventBus *eventbus.Manager
	mu       sync.RWMutex
}

// NewRuntime creates a plugin runtime.
func NewRuntime(cfg *config.GlobalConfig, eventBus *eventbus.Manager) *PluginRuntime {
	return &PluginRuntime{
		plugins:  make(map[string]*Plugin),
		config:   cfg,
		eventBus: eventBus,
	}
}

// Load registers a plugin path with runtime.
func (r *PluginRuntime) Load(path string) (*Plugin, error) {
	if path == "" {
		return nil, fmt.Errorf("plugin path is empty")
	}
	plugin := &Plugin{Path: path, Enabled: true, Config: map[string]string{}}
	r.mu.Lock()
	r.plugins[path] = plugin
	r.mu.Unlock()
	return plugin, nil
}

// Unload stops and removes a plugin.
func (r *PluginRuntime) Unload(id string) error {
	plug, err := r.Get(id)
	if err != nil {
		return err
	}
	if plug.Enabled {
		_ = r.Stop(id)
	}
	r.mu.Lock()
	delete(r.plugins, id)
	r.mu.Unlock()
	if r.eventBus != nil {
		r.eventBus.Publish("plugin.uninstalled", id)
	}
	return nil
}

// Start launches the plugin process.
func (r *PluginRuntime) Start(id string) error {
	plug, err := r.Get(id)
	if err != nil {
		return err
	}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "TERMBUS_PLUGIN",
			MagicCookieValue: "1",
		},
		Cmd:              exec.Command(plug.Path),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
	})

	plug.Client = client
	plug.Process = client.Cmd()
	if plug.Process != nil && plug.Process.Process != nil {
		plug.PID = plug.Process.Process.Pid
	}
	plug.StartedAt = time.Now()
	plug.Enabled = true

	if r.eventBus != nil {
		r.eventBus.Publish("plugin.started", plug.ID)
	}

	logger.GetLogger().Info("plugin started",
		zap.String("plugin", plug.Path),
	)

	return nil
}

// Stop stops a running plugin process.
func (r *PluginRuntime) Stop(id string) error {
	plug, err := r.Get(id)
	if err != nil {
		return err
	}
	if plug.Client != nil {
		plug.Client.Kill()
		plug.Client = nil
	}
	plug.Enabled = false
	if r.eventBus != nil {
		r.eventBus.Publish("plugin.stopped", plug.ID)
	}
	return nil
}

// Restart restarts a plugin.
func (r *PluginRuntime) Restart(id string) error {
	if err := r.Stop(id); err != nil {
		return err
	}
	return r.Start(id)
}

// Get returns a plugin by ID.
func (r *PluginRuntime) Get(id string) (*Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	plugin, ok := r.plugins[id]
	if !ok {
		return nil, fmt.Errorf("plugin not found")
	}
	return plugin, nil
}

// List returns all plugins.
func (r *PluginRuntime) List() []*Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*Plugin, 0, len(r.plugins))
	for _, plug := range r.plugins {
		list = append(list, plug)
	}
	return list
}

// ListEnabled returns enabled plugins.
func (r *PluginRuntime) ListEnabled() []*Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*Plugin, 0)
	for _, plug := range r.plugins {
		if plug.Enabled {
			list = append(list, plug)
		}
	}
	return list
}

// Execute runs a plugin command via gRPC.
func (r *PluginRuntime) Execute(id string, command string, args []string, env map[string]string) (*pluginrpc.ExecuteResponse, error) {
	plug, err := r.Get(id)
	if err != nil {
		return nil, err
	}
	if plug.Client == nil {
		return nil, fmt.Errorf("plugin client not started")
	}

	clientProtocol, err := plug.Client.Client()
	if err != nil {
		return nil, err
	}

	rpc, err := clientProtocol.Dispense("plugin")
	if err != nil {
		return nil, err
	}

	service, ok := rpc.(pluginrpc.PluginClient)
	if !ok {
		return nil, fmt.Errorf("invalid plugin client")
	}

	resp, err := service.Execute(context.Background(), &pluginrpc.ExecuteRequest{Command: command, Args: args, Env: env})
	if err != nil {
		if r.eventBus != nil {
			r.eventBus.Publish("plugin.failed", id, err)
		}
		return nil, err
	}

	if r.eventBus != nil {
		r.eventBus.Publish("plugin.executed", id, command)
	}

	return resp, nil
}
