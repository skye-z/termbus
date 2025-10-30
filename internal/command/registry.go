package command

import (
	"fmt"
	"sync"
)

// Command 命令定义
type Command struct {
	Name        string           `json:"name"`
	Aliases     []string         `json:"aliases"`
	Description string           `json:"description"`
	Category    string           `json:"category"`
	Handler     CommandHandler   `json:"-"`
	Args        []ArgDefinition  `json:"args"`
	Flags       []FlagDefinition `json:"flags"`
	Examples    []string         `json:"examples"`
}

// ArgDefinition 参数定义
type ArgDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}

// FlagDefinition 标志定义
type FlagDefinition struct {
	Name        string `json:"name"`
	Short       string `json:"short,omitempty"`
	Description string `json:"description"`
	Default     string `json:"default,omitempty"`
}

// CommandHandler 命令处理器
type CommandHandler func(ctx *ExecutionContext) error

// ExecutionContext 执行上下文
type ExecutionContext struct {
	SessionID string
	Args      []string
	Flags     map[string]string
}

// CommandRegistry 命令注册表
type CommandRegistry struct {
	commands map[string]*Command
	aliases  map[string]string
	mu       sync.RWMutex
}

// NewCommandRegistry 创建命令注册表
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*Command),
		aliases:  make(map[string]string),
	}
}

// Register 注册命令
func (r *CommandRegistry) Register(cmd *Command) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.commands[cmd.Name]; exists {
		return fmt.Errorf("command already exists: %s", cmd.Name)
	}

	r.commands[cmd.Name] = cmd

	for _, alias := range cmd.Aliases {
		r.aliases[alias] = cmd.Name
	}

	return nil
}

// Unregister 注销命令
func (r *CommandRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd, exists := r.commands[name]
	if !exists {
		return fmt.Errorf("command not found: %s", name)
	}

	delete(r.commands, name)

	for _, alias := range cmd.Aliases {
		delete(r.aliases, alias)
	}

	return nil
}

// Get 获取命令
func (r *CommandRegistry) Get(name string) (*Command, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if cmdName, isAlias := r.aliases[name]; isAlias {
		name = cmdName
	}

	cmd, exists := r.commands[name]
	if !exists {
		return nil, fmt.Errorf("command not found: %s", name)
	}

	return cmd, nil
}

// List 列出命令
func (r *CommandRegistry) List(category string) []*Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Command
	for _, cmd := range r.commands {
		if category == "" || cmd.Category == category {
			result = append(result, cmd)
		}
	}

	return result
}

// Search 搜索命令
func (r *CommandRegistry) Search(keyword string) []*Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Command
	for _, cmd := range r.commands {
		if contains(cmd.Name, keyword) || contains(cmd.Description, keyword) {
			result = append(result, cmd)
		}
	}

	return result
}

// contains 检查字符串包含
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) >= len(substr))
}
