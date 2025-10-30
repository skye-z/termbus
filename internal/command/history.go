package command

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/termbus/termbus/internal/logger"
	"go.uber.org/zap"
)

// HistoryEntry 历史条目
type HistoryEntry struct {
	ID        int64     `json:"id"`
	Command   string    `json:"command"`
	SessionID string    `json:"session_id"`
	Host      string    `json:"host"`
	Timestamp time.Time `json:"timestamp"`
	ExitCode  int       `json:"exit_code"`
}

// HistoryManager 历史管理器
type HistoryManager struct {
	entries map[string][]*HistoryEntry
	global  []*HistoryEntry
	config  *HistoryConfig
	mu      sync.RWMutex
	nextID  int64
}

// HistoryConfig 历史配置
type HistoryConfig struct {
	MaxSize   int    `json:"max_size"`
	SavePath  string `json:"save_path"`
	SessionID string `json:"session_id"`
}

// NewHistoryManager 创建历史管理器
func NewHistoryManager(cfg *HistoryConfig) *HistoryManager {
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 1000
	}

	return &HistoryManager{
		entries: make(map[string][]*HistoryEntry),
		global:  make([]*HistoryEntry, 0),
		config:  cfg,
		nextID:  1,
	}
}

// Add 添加历史
func (m *HistoryManager) Add(cmd string, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := &HistoryEntry{
		ID:        m.nextID,
		Command:   cmd,
		SessionID: sessionID,
		Timestamp: time.Now(),
		ExitCode:  0,
	}

	m.nextID++

	if sessionID != "" {
		if m.entries[sessionID] == nil {
			m.entries[sessionID] = make([]*HistoryEntry, 0)
		}
		m.entries[sessionID] = append(m.entries[sessionID], entry)

		if len(m.entries[sessionID]) > m.config.MaxSize {
			m.entries[sessionID] = m.entries[sessionID][1:]
		}
	}

	m.global = append(m.global, entry)
	if len(m.global) > m.config.MaxSize {
		m.global = m.global[1:]
	}

	logger.GetLogger().Info("history added",
		zap.String("command", cmd),
		zap.String("session_id", sessionID),
	)

	return m.Save()
}

// Get 获取历史
func (m *HistoryManager) Get(sessionID string, limit int) []*HistoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var entries []*HistoryEntry

	if sessionID == "" {
		entries = m.global
	} else {
		entries = m.entries[sessionID]
	}

	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	return entries
}

// GetGlobal 获取全局历史
func (m *HistoryManager) GetGlobal(limit int) []*HistoryEntry {
	return m.Get("", limit)
}

// Search 搜索历史
func (m *HistoryManager) Search(keyword string, sessionID string) []*HistoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var entries []*HistoryEntry
	target := m.entries[sessionID]

	if sessionID == "" {
		target = m.global
	}

	for _, entry := range target {
		if contains(entry.Command, keyword) || contains(entry.Host, keyword) {
			entries = append(entries, entry)
		}
	}

	return entries
}

// SearchByTime 按时间搜索
func (m *HistoryManager) SearchByTime(start, end time.Time, sessionID string) []*HistoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var entries []*HistoryEntry
	target := m.entries[sessionID]

	if sessionID == "" {
		target = m.global
	}

	for _, entry := range target {
		if (entry.Timestamp.Equal(start) || entry.Timestamp.After(start)) &&
			(entry.Timestamp.Equal(end) || entry.Timestamp.Before(end)) {
			entries = append(entries, entry)
		}
	}

	return entries
}

// SearchByExitCode 按退出码搜索
func (m *HistoryManager) SearchByExitCode(code int, sessionID string) []*HistoryEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var entries []*HistoryEntry
	target := m.entries[sessionID]

	if sessionID == "" {
		target = m.global
	}

	for _, entry := range target {
		if entry.ExitCode == code {
			entries = append(entries, entry)
		}
	}

	return entries
}

// Clear 清除历史
func (m *HistoryManager) Clear(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sessionID == "" {
		m.entries = make(map[string][]*HistoryEntry)
		m.global = make([]*HistoryEntry, 0)
	} else {
		delete(m.entries, sessionID)
	}

	return m.Save()
}

// Save 保存历史
func (m *HistoryManager) Save() error {
	if m.config.SavePath == "" {
		homeDir, _ := os.UserHomeDir()
		m.config.SavePath = filepath.Join(homeDir, ".termbus", "history.json")
	}

	data, err := json.MarshalIndent(m.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	dir := filepath.Dir(m.config.SavePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	if err := os.WriteFile(m.config.SavePath, data, 0644); err != nil {
		return fmt.Errorf("failed to save history: %w", err)
	}

	return nil
}

// Load 加载历史
func (m *HistoryManager) Load() error {
	if m.config.SavePath == "" {
		homeDir, _ := os.UserHomeDir()
		m.config.SavePath = filepath.Join(homeDir, ".termbus", "history.json")
	}

	data, err := os.ReadFile(m.config.SavePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to load history: %w", err)
	}

	if err := json.Unmarshal(data, &m.entries); err != nil {
		return fmt.Errorf("failed to unmarshal history: %w", err)
	}

	maxID := int64(0)
	for _, entries := range m.entries {
		for _, entry := range entries {
			if entry.ID > maxID {
				maxID = entry.ID
			}
		}
	}
	m.nextID = maxID + 1

	return nil
}

// Export 导出历史
func (m *HistoryManager) Export(sessionID string, format string, output string) error {
	entries := m.Get(sessionID, 0)

	var content string
	switch format {
	case "json":
		data, _ := json.MarshalIndent(entries, "", "  ")
		content = string(data)
	case "csv":
		content = "ID,Command,SessionID,Host,Timestamp,ExitCode\n"
		for _, entry := range entries {
			content += fmt.Sprintf("%d,\"%s\",\"%s\",\"%s\",\"%s\",%d\n",
				entry.ID, entry.Command, entry.SessionID, entry.Host,
				entry.Timestamp.Format(time.RFC3339), entry.ExitCode)
		}
	default:
		for _, entry := range entries {
			content += fmt.Sprintf("%s [%s] %s\n",
				entry.Timestamp.Format("2006-01-02 15:04:05"),
				entry.SessionID, entry.Command)
		}
	}

	return os.WriteFile(output, []byte(content), 0644)
}
