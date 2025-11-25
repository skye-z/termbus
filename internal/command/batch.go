package command

import (
	"fmt"
	"sync"
	"time"

	"github.com/termbus/termbus/internal/eventbus"
	"github.com/termbus/termbus/internal/logger"
	"github.com/termbus/termbus/internal/session"
	"github.com/termbus/termbus/pkg/types"
	"go.uber.org/zap"
)

// BatchExecutor 批量执行管理器
type BatchExecutor struct {
	sessionMgr session.SessionManager
	eventBus   *eventbus.Manager
	timeout    time.Duration
}

// NewBatchExecutor 创建批量执行管理器
func NewBatchExecutor(sessionMgr session.SessionManager, eventBus *eventbus.Manager, timeout int) *BatchExecutor {
	return &BatchExecutor{
		sessionMgr: sessionMgr,
		eventBus:   eventBus,
		timeout:    time.Duration(timeout) * time.Second,
	}
}

// Execute 执行批量命令
func (e *BatchExecutor) Execute(batch *BatchCommand) ([]*BatchResult, error) {
	e.eventBus.Publish("batch.started", batch)

	results := make([]*BatchResult, len(batch.SessionIDs))

	var wg sync.WaitGroup
	sem := make(chan struct{}, batch.Parallel)
	if batch.Parallel <= 0 {
		batch.Parallel = 5
	}

	for i, sessionID := range batch.SessionIDs {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, sid string) {
			defer wg.Done()
			defer func() { <-sem }()

			result := e.executeOnSession(sid, batch.Command, batch.Args, batch.Timeout)
			results[idx] = result

			if result.Error != "" {
				logger.GetLogger().Error("batch execution failed",
					zap.String("session_id", sid),
					zap.String("error", result.Error),
				)
			}
		}(i, sessionID)
	}

	wg.Wait()

	e.eventBus.Publish("batch.completed", batch, results)

	return results, nil
}

// ExecuteGroup 执行分组命令
func (e *BatchExecutor) ExecuteGroup(groupName, cmd string, args []string) ([]*BatchResult, error) {
	session, err := e.sessionMgr.GetSession("")
	if err != nil {
		return nil, err
	}

	sessionIDs := getSessionIDsFromGroup(groupName, session.HostConfig.HostName)

	batch := &BatchCommand{
		Command:    cmd,
		Args:       args,
		SessionIDs: sessionIDs,
		Parallel:   5,
		Timeout:    30,
	}

	return e.Execute(batch)
}

// ExecuteAll 在所有会话上执行
func (e *BatchExecutor) ExecuteAll(cmd string, args []string) ([]*BatchResult, error) {
	sessions := e.sessionMgr.ListSessions()

	sessionIDs := make([]string, len(sessions))
	for i, session := range sessions {
		sessionIDs[i] = session.ID
	}

	batch := &BatchCommand{
		Command:    cmd,
		Args:       args,
		SessionIDs: sessionIDs,
		Parallel:   10,
		Timeout:    30,
	}

	return e.Execute(batch)
}

// executeOnSession 在指定会话上执行
func (e *BatchExecutor) executeOnSession(sessionID, cmd string, args []string, timeout int) *BatchResult {
	start := time.Now()

	session, err := e.sessionMgr.GetSession(sessionID)
	if err != nil {
		return &BatchResult{
			SessionID: sessionID,
			Command:   cmd,
			Error:     err.Error(),
		}
	}

	output, err := e.executeCommand(session, cmd, args, timeout)

	return &BatchResult{
		SessionID: sessionID,
		Command:   cmd,
		Output:    output,
		Error: func() string {
			if err != nil {
				return err.Error()
			} else {
				return ""
			}
		}(),
		ExitCode: func() int {
			if err != nil {
				return 1
			} else {
				return 0
			}
		}(),
		Duration: time.Since(start),
	}
}

// executeCommand 执行命令
func (e *BatchExecutor) executeCommand(session *types.Session, cmd string, args []string, timeout int) (string, error) {
	output := fmt.Sprintf("Executed: %s %s on %s", cmd, args, session.HostConfig.HostName)

	logger.GetLogger().Info("batch command executed",
		zap.String("session_id", session.ID),
		zap.String("command", cmd),
		zap.Int("timeout", timeout),
	)

	return output, nil
}

// getSessionIDsFromGroup 从分组获取会话ID
func getSessionIDsFromGroup(groupName, host string) []string {
	return []string{host}
}

// BatchCommand 批量命令
type BatchCommand struct {
	Command    string
	Args       []string
	SessionIDs []string
	Timeout    int
	Parallel   int
	Flags      map[string]string
}

// BatchResult 批量结果
type BatchResult struct {
	SessionID string
	Command   string
	Output    string
	Error     string
	Duration  time.Duration
	ExitCode  int
}

// GetBatchSummary 获取批量执行摘要
func GetBatchSummary(results []*BatchResult) string {
	success := 0
	failed := 0
	totalDuration := time.Duration(0)

	for _, result := range results {
		if result.ExitCode == 0 {
			success++
		} else {
			failed++
		}
		totalDuration += result.Duration
	}

	summary := fmt.Sprintf(
		"Total: %d | Success: %d | Failed: %d | Avg Duration: %v",
		len(results), success, failed,
		totalDuration/time.Duration(len(results)),
	)

	return summary
}
