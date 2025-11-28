package plugin

import (
	"fmt"
	"time"

	"github.com/termbus/termbus/internal/eventbus"
)

// PluginManager manages plugin lifecycle operations.
type PluginManager struct {
	runtime   *PluginRuntime
	loader    *PluginLoader
	installer *PluginInstaller
	store     *PluginStore
	eventBus  *eventbus.Manager
	audit     *AuditLogger
}

// NewManager creates a plugin manager.
func NewManager(runtime *PluginRuntime, loader *PluginLoader, installer *PluginInstaller, store *PluginStore, audit *AuditLogger, eventBus *eventbus.Manager) *PluginManager {
	return &PluginManager{runtime: runtime, loader: loader, installer: installer, store: store, audit: audit, eventBus: eventBus}
}

// Install installs a plugin from source.
func (m *PluginManager) Install(source string) (*Plugin, error) {
	if m.installer == nil {
		return nil, fmt.Errorf("installer not configured")
	}
	plug, err := m.installer.InstallFromURL(source)
	if err != nil {
		plug, err = m.installer.InstallFromFile(source)
	}
	if err != nil {
		plug, err = m.installer.InstallFromDir(source)
	}
	if err != nil {
		return nil, err
	}
	if m.store != nil {
		_ = m.store.Add(plug)
	}
	if m.audit != nil {
		_ = m.audit.Log(&AuditEntry{PluginID: plug.ID, Action: "install", Result: "success", Timestamp: time.Now()})
	}
	if m.eventBus != nil {
		m.eventBus.Publish("plugin.installed", plug.ID)
	}
	return plug, nil
}

// Uninstall removes a plugin.
func (m *PluginManager) Uninstall(id string) error {
	if m.runtime != nil {
		_ = m.runtime.Unload(id)
	}
	if m.installer != nil {
		_ = m.installer.Uninstall(id)
	}
	if m.store != nil {
		_ = m.store.Remove(id)
	}
	if m.audit != nil {
		_ = m.audit.Log(&AuditEntry{PluginID: id, Action: "uninstall", Result: "success", Timestamp: time.Now()})
	}
	if m.eventBus != nil {
		m.eventBus.Publish("plugin.uninstalled", id)
	}
	return nil
}

// Enable enables a plugin.
func (m *PluginManager) Enable(id string) error {
	if m.runtime == nil {
		return fmt.Errorf("runtime not configured")
	}
	if err := m.runtime.Start(id); err != nil {
		return err
	}
	if m.eventBus != nil {
		m.eventBus.Publish("plugin.enabled", id)
	}
	return nil
}

// Disable disables a plugin.
func (m *PluginManager) Disable(id string) error {
	if m.runtime == nil {
		return fmt.Errorf("runtime not configured")
	}
	if err := m.runtime.Stop(id); err != nil {
		return err
	}
	if m.eventBus != nil {
		m.eventBus.Publish("plugin.disabled", id)
	}
	return nil
}

// Update restarts a plugin after reloading.
func (m *PluginManager) Update(id string) error {
	if m.runtime == nil {
		return fmt.Errorf("runtime not configured")
	}
	return m.runtime.Restart(id)
}

// Reload reloads a plugin.
func (m *PluginManager) Reload(id string) error {
	if m.runtime == nil {
		return fmt.Errorf("runtime not configured")
	}
	return m.runtime.Restart(id)
}

// Info returns plugin info.
func (m *PluginManager) Info(id string) (*Plugin, error) {
	return m.Get(id)
}

// List returns all plugins.
func (m *PluginManager) List() []*Plugin {
	if m.runtime == nil {
		return nil
	}
	return m.runtime.List()
}

// Get returns a plugin by ID.
func (m *PluginManager) Get(id string) (*Plugin, error) {
	if m.runtime == nil {
		return nil, fmt.Errorf("runtime not configured")
	}
	return m.runtime.Get(id)
}
