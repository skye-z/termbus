package plugin

import (
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
	mu        sync.RWMutex
}

// NewAuthorizer creates an authorizer.
func NewAuthorizer(permMgr *PermissionManager, eventBus *eventbus.Manager) *Authorizer {
	return &Authorizer{permMgr: permMgr, eventBus: eventBus, decisions: make(map[string]*AuthorizationDecision)}
}

// RequestAuthorization requests permissions.
func (a *Authorizer) RequestAuthorization(req *AuthorizationRequest) (*AuthorizationDecision, error) {
	decision := &AuthorizationDecision{Granted: true, Permissions: req.Permissions, Timestamp: time.Now()}
	a.mu.Lock()
	a.decisions[req.PluginID] = decision
	a.mu.Unlock()
	for _, perm := range req.Permissions {
		_ = a.permMgr.Grant(req.PluginID, perm)
	}
	if a.eventBus != nil {
		a.eventBus.Publish("plugin.permission.granted", req.PluginID)
	}
	return decision, nil
}

// RevokeAuthorization revokes a plugin authorization.
func (a *Authorizer) RevokeAuthorization(pluginID string) error {
	a.mu.Lock()
	delete(a.decisions, pluginID)
	a.mu.Unlock()
	if a.eventBus != nil {
		a.eventBus.Publish("plugin.permission.revoked", pluginID)
	}
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
