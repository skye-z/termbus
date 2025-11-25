package command

import (
	"fmt"
	"sync"

	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/session"
)

type Executor struct {
	registry   *CommandRegistry
	sessionMgr session.SessionManager
	eventBus   *eventbus.Manager
	mu         sync.RWMutex
}

// NewExecutor 创建命令执行器
func NewExecutor(registry *CommandRegistry, sessionMgr session.SessionManager, eventBus *eventbus.Manager) *Executor {
	return &Executor{
		registry:   registry,
		sessionMgr: sessionMgr,
		eventBus:   eventBus,
	}
}

// Execute 执行命令
func (e *Executor) Execute(cmdName string, args []string, flags map[string]string) error {
	cmd, err := e.registry.Get(cmdName)
	if err != nil {
		return err
	}

	ctx := &ExecutionContext{
		SessionID: flags["session"],
		Args:      args,
		Flags:     flags,
	}

	err = cmd.Handler(ctx)
	if err != nil {
		e.eventBus.Publish("command.failed", cmdName, err)
		return err
	}

	e.eventBus.Publish("command.executed", cmdName, ctx)
	return nil
}

// ExecuteBatch 批量执行命令
func (e *Executor) ExecuteBatch(commands []*BatchCommand) error {
	for _, batchCmd := range commands {
		err := e.Execute(batchCmd.Command, batchCmd.Args, batchCmd.Flags)
		if err != nil {
			return err
		}
	}
	return nil
}

// RegisterCommand 注册命令
func (e *Executor) RegisterCommand(cmd *Command) error {
	return e.registry.Register(cmd)
}

// GetCommand 获取命令
func (e *Executor) GetCommand(name string) (*Command, error) {
	return e.registry.Get(name)
}

// ListCommands 列出命令
func (e *Executor) ListCommands(category string) []*Command {
	return e.registry.List(category)
}

// SearchCommands 搜索命令
func (e *Executor) SearchCommands(keyword string) []*Command {
	return e.registry.Search(keyword)
}

// ExecuteOnSessions 在指定会话上执行命令
func (e *Executor) ExecuteOnSessions(sessionIDs []string, cmdName string, args []string) []*CommandResult {
	var results []*CommandResult

	for _, sessionID := range sessionIDs {
		session, err := e.sessionMgr.GetSession(sessionID)
		if err != nil {
			results = append(results, &CommandResult{
				SessionID: sessionID,
				Command:   cmdName,
				Error:     err.Error(),
			})
			continue
		}

		err = e.Execute(cmdName, args, map[string]string{"session": sessionID})
		results = append(results, &CommandResult{
			SessionID: sessionID,
			Command:   cmdName,
			Output:    fmt.Sprintf("Executed on %s (%s)", session.HostConfig.HostName, session.HostConfig.Host),
			Error: func() string {
				if err != nil {
					return err.Error()
				} else {
					return ""
				}
			}(),
		})
	}

	return results
}

// CommandResult 命令执行结果
type CommandResult struct {
	SessionID string
	Command   string
	Output    string
	Error     string
	Duration  int64
}
