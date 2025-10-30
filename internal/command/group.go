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

// HostGroup 主机分组
type HostGroup struct {
	Name      string    `json:"name"`
	Hosts     []string  `json:"hosts"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
}

// GroupManager 分组管理器
type GroupManager struct {
	groups map[string]*HostGroup
	mu     sync.RWMutex
}

// NewGroupManager 创建分组管理器
func NewGroupManager() *GroupManager {
	return &GroupManager{
		groups: make(map[string]*HostGroup),
	}
}

// Create 创建分组
func (m *GroupManager) Create(name string, hosts []string) (*HostGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.groups[name]; exists {
		return nil, fmt.Errorf("group already exists: %s", name)
	}

	group := &HostGroup{
		Name:      name,
		Hosts:     hosts,
		Tags:      []string{},
		CreatedAt: time.Now(),
	}

	m.groups[name] = group

	logger.GetLogger().Info("group created",
		zap.String("name", name),
		zap.Int("hosts", len(hosts)),
	)

	if err := m.Save(); err != nil {
		delete(m.groups, name)
		return nil, err
	}

	return group, nil
}

// Delete 删除分组
func (m *GroupManager) Delete(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.groups[name]; !exists {
		return fmt.Errorf("group not found: %s", name)
	}

	delete(m.groups, name)

	logger.GetLogger().Info("group deleted", zap.String("name", name))

	return m.Save()
}

// Get 获取分组
func (m *GroupManager) Get(name string) (*HostGroup, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, exists := m.groups[name]
	if !exists {
		return nil, fmt.Errorf("group not found: %s", name)
	}

	return group, nil
}

// List 列出所有分组
func (m *GroupManager) List() []*HostGroup {
	m.mu.RLock()
	defer m.mu.RUnlock()

	groups := make([]*HostGroup, 0, len(m.groups))
	for _, group := range m.groups {
		groups = append(groups, group)
	}

	return groups
}

// AddHost 添加主机到分组
func (m *GroupManager) AddHost(name, host string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, exists := m.groups[name]
	if !exists {
		return fmt.Errorf("group not found: %s", name)
	}

	for _, h := range group.Hosts {
		if h == host {
			return fmt.Errorf("host already in group: %s", host)
		}
	}

	group.Hosts = append(group.Hosts, host)

	logger.GetLogger().Info("host added to group",
		zap.String("group", name),
		zap.String("host", host),
	)

	return m.Save()
}

// RemoveHost 从分组移除主机
func (m *GroupManager) RemoveHost(name, host string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, exists := m.groups[name]
	if !exists {
		return fmt.Errorf("group not found: %s", name)
	}

	found := false
	for i, h := range group.Hosts {
		if h == host {
			group.Hosts = append(group.Hosts[:i], group.Hosts[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("host not in group: %s", host)
	}

	logger.GetLogger().Info("host removed from group",
		zap.String("group", name),
		zap.String("host", host),
	)

	return m.Save()
}

// Save 保存分组
func (m *GroupManager) Save() error {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".termbus", "groups.json")

	data, err := json.MarshalIndent(m.groups, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal groups: %w", err)
	}

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create groups directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save groups: %w", err)
	}

	return nil
}

// Load 加载分组
func (m *GroupManager) Load() error {
	homeDir, _ := os.UserHomeDir()
	configPath := filepath.Join(homeDir, ".termbus", "groups.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to load groups: %w", err)
	}

	if err := json.Unmarshal(data, &m.groups); err != nil {
		return fmt.Errorf("failed to unmarshal groups: %w", err)
	}

	return nil
}
