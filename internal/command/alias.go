package command

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/termbus/termbus/internal/logger"
	"go.uber.org/zap"
)

// AliasManager 别名管理器
type AliasManager struct {
	aliases map[string]string
	mu      sync.RWMutex
}

// NewAliasManager 创建别名管理器
func NewAliasManager() *AliasManager {
	return &AliasManager{
		aliases: make(map[string]string),
	}
}

// Add 添加别名
func (m *AliasManager) Add(name, cmd string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.aliases[name] = cmd
	logger.GetLogger().Info("alias added",
		zap.String("name", name),
		zap.String("command", cmd),
	)

	return m.Save()
}

// Remove 删除别名
func (m *AliasManager) Remove(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.aliases[name]; !exists {
		return fmt.Errorf("alias not found: %s", name)
	}

	delete(m.aliases, name)
	logger.GetLogger().Info("alias removed",
		zap.String("name", name),
	)

	return m.Save()
}

// Get 获取别名
func (m *AliasManager) Get(name string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cmd, exists := m.aliases[name]
	if !exists {
		return "", fmt.Errorf("alias not found: %s", name)
	}

	return cmd, nil
}

// Expand 展开别名
func (m *AliasManager) Expand(input string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	words := splitWords(input)
	if len(words) == 0 {
		return input
	}

	if cmd, exists := m.aliases[words[0]]; exists {
		args := words[1:]
		if len(args) > 0 {
			return fmt.Sprintf("%s %s", cmd, joinWords(args))
		}
		return cmd
	}

	return input
}

// List 列出所有别名
func (m *AliasManager) List() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]string)
	for k, v := range m.aliases {
		result[k] = v
	}

	return result
}

// Save 保存别名
func (m *AliasManager) Save() error {
	configPath := filepath.Join(getConfigDir(), "aliases.json")

	data, err := json.MarshalIndent(m.aliases, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal aliases: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save aliases: %w", err)
	}

	return nil
}

// Load 加载别名
func (m *AliasManager) Load() error {
	configPath := filepath.Join(getConfigDir(), "aliases.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if err := json.Unmarshal(data, &m.aliases); err != nil {
		return fmt.Errorf("failed to unmarshal aliases: %w", err)
	}

	return nil
}

// getConfigDir 获取配置目录
func getConfigDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".termbus")
}

// splitWords 分割单词
func splitWords(input string) []string {
	var words []string
	var current strings.Builder
	inQuote := false

	for _, r := range input {
		switch r {
		case '"':
			inQuote = !inQuote
		case ' ':
			if !inQuote && current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		words = append(words, current.String())
	}

	return words
}

// joinWords 连接单词
func joinWords(words []string) string {
	return fmt.Sprintf("'%s'", strings.Join(words, "' '"))
}
