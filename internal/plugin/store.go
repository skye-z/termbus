package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/termbus/termbus/internal/config"
)

// PluginStore stores plugin metadata.
type PluginStore struct {
	plugins map[string]*Plugin
	config  *config.GlobalConfig
	mu      sync.RWMutex
}

// NewStore creates a plugin store.
func NewStore(cfg *config.GlobalConfig) *PluginStore {
	return &PluginStore{plugins: make(map[string]*Plugin), config: cfg}
}

// Add adds a plugin to store.
func (s *PluginStore) Add(plugin *Plugin) error {
	if plugin == nil || plugin.Path == "" {
		return fmt.Errorf("invalid plugin")
	}
	s.mu.Lock()
	s.plugins[plugin.Path] = plugin
	s.mu.Unlock()
	return s.Save()
}

// Remove removes a plugin from store.
func (s *PluginStore) Remove(id string) error {
	s.mu.Lock()
	delete(s.plugins, id)
	s.mu.Unlock()
	return s.Save()
}

// Get returns a plugin by ID.
func (s *PluginStore) Get(id string) (*Plugin, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	plugin, ok := s.plugins[id]
	if !ok {
		return nil, fmt.Errorf("plugin not found")
	}
	return plugin, nil
}

// List returns all plugins.
func (s *PluginStore) List() []*Plugin {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*Plugin, 0, len(s.plugins))
	for _, plug := range s.plugins {
		list = append(list, plug)
	}
	return list
}

// Save persists plugin metadata.
func (s *PluginStore) Save() error {
	if s.config == nil {
		return nil
	}
	path := filepath.Join(s.config.General.DataDir, "plugins.json")
	data, err := json.MarshalIndent(s.plugins, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// Load loads plugin metadata.
func (s *PluginStore) Load() error {
	if s.config == nil {
		return nil
	}
	path := filepath.Join(s.config.General.DataDir, "plugins.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var plugins map[string]*Plugin
	if err := json.Unmarshal(data, &plugins); err != nil {
		return err
	}
	s.mu.Lock()
	s.plugins = plugins
	s.mu.Unlock()
	return nil
}
