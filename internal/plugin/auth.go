package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/termbus/termbus/internal/eventbus"
)

// AuthorizationRequest represents a permission request.
type AuthorizationRequest struct {
	PluginID    string       `json:"plugin_id"`
	Permissions []Permission `json:"permissions"`
	Reason      string       `json:"reason"`
	Timestamp   time.Time    `json:"timestamp"`
}

// AuthorizationDecision represents a permission decision.
type AuthorizationDecision struct {
	Granted     bool         `json:"granted"`
	Permissions []Permission `json:"permissions"`
	Expiry      time.Time    `json:"expiry"`
	Timestamp   time.Time    `json:"timestamp"`
}

// Authorizer manages authorization flow.
type Authorizer struct {
	permMgr   *PermissionManager
	eventBus  *eventbus.Manager
	decisions map[string]*AuthorizationDecision
	storePath string
	mu        sync.RWMutex
}

// NewAuthorizer creates an authorizer.
func NewAuthorizer(permMgr *PermissionManager, eventBus *eventbus.Manager, storePath string) *Authorizer {
	authorizer := &Authorizer{permMgr: permMgr, eventBus: eventBus, decisions: make(map[string]*AuthorizationDecision), storePath: storePath}
	_ = authorizer.Load()
	if eventBus != nil {
		eventBus.Subscribe("plugin.permission.granted", func(args ...interface{}) {
			if len(args) == 0 {
				return
			}
			pluginID, ok := args[0].(string)
			if !ok {
				return
			}
			authorizer.GrantAll(pluginID)
		})
		eventBus.Subscribe("plugin.permission.revoked", func(args ...interface{}) {
			if len(args) == 0 {
				return
			}
			pluginID, ok := args[0].(string)
			if !ok {
				return
			}
			_ = authorizer.RevokeAuthorization(pluginID)
		})
	}
	return authorizer
}

// RequestAuthorization requests permissions.
func (a *Authorizer) RequestAuthorization(req *AuthorizationRequest) (*AuthorizationDecision, error) {
	if a.eventBus != nil {
		a.eventBus.Publish("plugin.permission.requested", req.PluginID, req.Permissions)
	}
	decision := &AuthorizationDecision{Granted: false, Permissions: req.Permissions, Timestamp: time.Now()}
	a.mu.Lock()
	a.decisions[req.PluginID] = decision
	a.mu.Unlock()
	_ = a.Save()
	return decision, nil
}

// GrantAll grants permissions from the last decision.
func (a *Authorizer) GrantAll(pluginID string) {
	a.mu.RLock()
	decision := a.decisions[pluginID]
	a.mu.RUnlock()
	if decision == nil {
		return
	}
	decision.Granted = true
	for _, perm := range decision.Permissions {
		_ = a.permMgr.Grant(pluginID, perm)
	}
	_ = a.Save()
}

// RevokeAuthorization revokes a plugin authorization.
func (a *Authorizer) RevokeAuthorization(pluginID string) error {
	a.mu.Lock()
	delete(a.decisions, pluginID)
	a.mu.Unlock()
	_ = a.Save()
	if a.eventBus != nil {
		a.eventBus.Publish("plugin.permission.revoked", pluginID)
	}
	return nil
}

// Save persists authorization decisions.
func (a *Authorizer) Save() error {
	if a.storePath == "" {
		return nil
	}
	data, err := json.MarshalIndent(a.decisions, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(a.storePath), 0755); err != nil {
		return err
	}
	return os.WriteFile(a.storePath, data, 0600)
}

// Load loads authorization decisions from disk.
func (a *Authorizer) Load() error {
	if a.storePath == "" {
		return nil
	}
	data, err := os.ReadFile(a.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	decisions := make(map[string]*AuthorizationDecision)
	if err := json.Unmarshal(data, &decisions); err != nil {
		return err
	}
	a.mu.Lock()
	a.decisions = decisions
	a.mu.Unlock()
	return nil
}

// ListAuthorizations lists authorizations for a plugin.
func (a *Authorizer) ListAuthorizations(pluginID string) []*AuthorizationDecision {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if decision, ok := a.decisions[pluginID]; ok {
		return []*AuthorizationDecision{decision}
	}
	return nil
}
