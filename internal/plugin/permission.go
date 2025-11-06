package plugin

import (
	"sync"

	"github.com/termbus/termbus/internal/config"
)

// Permission defines a plugin permission.
type Permission string

const (
	PermSSHConnect    Permission = "ssh.connect"
	PermSSHExecute    Permission = "ssh.execute"
	PermSFTPRead      Permission = "sftp.read"
	PermSFTPWrite     Permission = "sftp.write"
	PermSFTPDelete    Permission = "sftp.delete"
	PermTunnelCreate  Permission = "tunnel.create"
	PermTunnelManage  Permission = "tunnel.manage"
	PermConfigRead    Permission = "config.read"
	PermConfigWrite   Permission = "config.write"
	PermSystemExec    Permission = "system.exec"
	PermSystemNetwork Permission = "system.network"
)

// PermissionManager manages plugin permissions.
type PermissionManager struct {
	permissions map[string]map[Permission]bool
	config      *config.GlobalConfig
	mu          sync.RWMutex
}

// NewPermissionManager creates a permission manager.
func NewPermissionManager(cfg *config.GlobalConfig) *PermissionManager {
	return &PermissionManager{permissions: make(map[string]map[Permission]bool), config: cfg}
}

// Grant grants a permission.
func (m *PermissionManager) Grant(pluginID string, perm Permission) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.permissions[pluginID]; !ok {
		m.permissions[pluginID] = make(map[Permission]bool)
	}
	m.permissions[pluginID][perm] = true
	return nil
}

// Revoke revokes a permission.
func (m *PermissionManager) Revoke(pluginID string, perm Permission) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.permissions[pluginID]; ok {
		delete(m.permissions[pluginID], perm)
	}
	return nil
}

// Check checks if permission granted.
func (m *PermissionManager) Check(pluginID string, perm Permission) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if perms, ok := m.permissions[pluginID]; ok {
		return perms[perm]
	}
	return false
}

// List lists permissions for a plugin.
func (m *PermissionManager) List(pluginID string) []Permission {
	m.mu.RLock()
	defer m.mu.RUnlock()
	perms := make([]Permission, 0)
	if set, ok := m.permissions[pluginID]; ok {
		for perm := range set {
			perms = append(perms, perm)
		}
	}
	return perms
}

// Request requests a permission (stub for UI flow).
func (m *PermissionManager) Request(pluginID string, perm Permission) error {
	return m.Grant(pluginID, perm)
}
