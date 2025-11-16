package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/termbus/termbus/internal/eventbus"
)

// PluginManifest describes a plugin manifest file.
type PluginManifest struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	Permissions  []string          `json:"permissions"`
	Commands     []string          `json:"commands"`
	ConfigSchema map[string]string `json:"config_schema"`
}

// PluginLoader handles plugin discovery and validation.
type PluginLoader struct {
	runtime  *PluginRuntime
	eventBus *eventbus.Manager
}

// NewLoader creates a plugin loader.
func NewLoader(runtime *PluginRuntime, eventBus *eventbus.Manager) *PluginLoader {
	return &PluginLoader{runtime: runtime, eventBus: eventBus}
}

// Discover finds plugin binaries in a directory.
func (l *PluginLoader) Discover(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		paths = append(paths, filepath.Join(dir, entry.Name()))
	}
	return paths, nil
}

// Validate validates plugin manifest at the path.
func (l *PluginLoader) Validate(path string) (*PluginManifest, error) {
	manifestPath := filepath.Join(filepath.Dir(path), "plugin.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("invalid plugin manifest: %w", err)
	}
	if manifest.Name == "" || manifest.Version == "" {
		return nil, fmt.Errorf("invalid manifest: name/version required")
	}
	return &manifest, nil
}

// LoadPlugin loads a plugin into runtime.
func (l *PluginLoader) LoadPlugin(path string) (*Plugin, error) {
	manifest, _ := l.Validate(path)
	plugin, err := l.runtime.Load(path)
	if err != nil {
		return nil, err
	}
	if manifest != nil {
		plugin.ID = manifest.Name
		plugin.Name = manifest.Name
		plugin.Version = manifest.Version
	}
	if l.eventBus != nil {
		l.eventBus.Publish("plugin.installed", plugin.ID)
	}
	return plugin, nil
}
