package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/termbus/termbus/internal/config"
)

// AuditEntry represents plugin audit event.
type AuditEntry struct {
	ID        string    `json:"id"`
	PluginID  string    `json:"plugin_id"`
	Action    string    `json:"action"`
	User      string    `json:"user"`
	Timestamp time.Time `json:"timestamp"`
	Result    string    `json:"result"`
	Details   string    `json:"details"`
}

// AuditLogger records audit events.
type AuditLogger struct {
	entries map[string][]*AuditEntry
	config  *config.GlobalConfig
	mu      sync.RWMutex
}

// NewAuditLogger creates a new audit logger.
func NewAuditLogger(cfg *config.GlobalConfig) *AuditLogger {
	return &AuditLogger{entries: make(map[string][]*AuditEntry), config: cfg}
}

// Log logs an audit entry.
func (l *AuditLogger) Log(entry *AuditEntry) error {
	if entry == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries[entry.PluginID] = append(l.entries[entry.PluginID], entry)
	return l.persist(entry.PluginID)
}

// Query returns audit entries between time ranges.
func (l *AuditLogger) Query(pluginID string, start, end time.Time) []*AuditEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	entries := make([]*AuditEntry, 0)
	for _, entry := range l.entries[pluginID] {
		if entry.Timestamp.After(start) && entry.Timestamp.Before(end) {
			entries = append(entries, entry)
		}
	}
	return entries
}

// Export exports audit entries.
func (l *AuditLogger) Export(pluginID string, format string, output string) error {
	l.mu.RLock()
	entries := l.entries[pluginID]
	l.mu.RUnlock()
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(output, data, 0600)
}

// Search searches audit entries by keyword.
func (l *AuditLogger) Search(keyword string) []*AuditEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	results := make([]*AuditEntry, 0)
	for _, entries := range l.entries {
		for _, entry := range entries {
			if strings.Contains(entry.Action, keyword) || strings.Contains(entry.Details, keyword) || strings.Contains(entry.Result, keyword) {
				results = append(results, entry)
			}
		}
	}
	return results
}

func (l *AuditLogger) persist(pluginID string) error {
	if l.config == nil {
		return nil
	}
	path := filepath.Join(l.config.General.DataDir, "audit", pluginID+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(l.entries[pluginID], "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
