package plugin

import (
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/termbus/termbus/internal/config"
	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/logger"
	"go.uber.org/zap"
)

type ExecuteResponse struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Error    string `json:"error"`
}

type Plugin struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	Path      string            `json:"path"`
	Enabled   bool              `json:"enabled"`
	Config    map[string]string `json:"config"`
	Process   *exec.Cmd         `json:"-"`
	PID       int               `json:"pid"`
	StartedAt time.Time         `json:"started_at"`
}

type PluginRuntime struct {
	plugins  map[string]*Plugin
	config   *config.GlobalConfig
	eventBus *eventbus.Manager
	mu       sync.RWMutex
}

func NewRuntime(cfg *config.GlobalConfig, eventBus *eventbus.Manager) *PluginRuntime {
	return &PluginRuntime{
		plugins:  make(map[string]*Plugin),
		config:   cfg,
		eventBus: eventBus,
	}
}

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

func (r *PluginRuntime) Start(id string) error {
	plug, err := r.Get(id)
	if err != nil {
		return err
	}

	cmd := exec.Command(plug.Path)
	cmd.Start()
	plug.Process = cmd
	plug.PID = cmd.Process.Pid
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

func (r *PluginRuntime) Stop(id string) error {
	plug, err := r.Get(id)
	if err != nil {
		return err
	}
	if plug.Process != nil && plug.Process.Process != nil {
		_ = plug.Process.Process.Kill()
	}
	plug.Enabled = false
	if r.eventBus != nil {
		r.eventBus.Publish("plugin.stopped", plug.ID)
	}
	return nil
}

func (r *PluginRuntime) Restart(id string) error {
	if err := r.Stop(id); err != nil {
		return err
	}
	return r.Start(id)
}

func (r *PluginRuntime) Get(id string) (*Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	plugin, ok := r.plugins[id]
	if !ok {
		return nil, fmt.Errorf("plugin not found")
	}
	return plugin, nil
}

func (r *PluginRuntime) List() []*Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	list := make([]*Plugin, 0, len(r.plugins))
	for _, plug := range r.plugins {
		list = append(list, plug)
	}
	return list
}

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

func (r *PluginRuntime) Execute(id string, command string, args []string, env map[string]string) (*ExecuteResponse, error) {
	plug, err := r.Get(id)
	if err != nil {
		return nil, err
	}
	if !plug.Enabled || plug.Process == nil {
		return nil, fmt.Errorf("plugin not running")
	}

	logger.GetLogger().Info("plugin execute",
		zap.String("plugin", plug.Path),
		zap.String("command", command),
	)

	return &ExecuteResponse{
		ExitCode: 0,
		Stdout:   fmt.Sprintf("Executed: %s on plugin %s", command, plug.Name),
		Stderr:   "",
		Error:    "",
	}, nil
}
