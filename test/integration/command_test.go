package command

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchExecutor(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	registry := NewCommandRegistry()
	sessMgr := session.NewSessionManager(nil, nil, nil)
	eventBus := eventbus.NewManager()

	executor := NewBatchExecutor(registry, sessMgr, eventBus, 30)

	batch := &BatchCommand{
		Command:    "echo",
		Args:       []string{"test"},
		SessionIDs: []string{"session1", "session2"},
		Timeout:    10,
		Parallel:   2,
	}

	results, err := executor.Execute(batch)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestTaskScheduler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	registry := NewCommandRegistry()
	sessMgr := session.NewSessionManager(nil, nil, nil)
	eventBus := eventbus.NewManager()

	executor := NewBatchExecutor(registry, sessMgr, eventBus, 30)
	scheduler := NewTaskScheduler(executor, eventBus)

	task := &ScheduledTask{
		ID:         "test-task",
		Name:       "Test Task",
		Command:    "echo test",
		Schedule:   "* * * * *",
		SessionIDs: []string{},
		GroupName:  "test",
		Enabled:    true,
	}

	err := scheduler.Add(task)
	assert.NoError(t, err)

	list := scheduler.List()
	assert.Len(t, list, 1)

	err = scheduler.Remove(task.ID)
	assert.NoError(t, err)

	list = scheduler.List()
	assert.Len(t, list, 0)
}

func TestCommandExecutionFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	registry := NewCommandRegistry()
	sessMgr := session.NewSessionManager(nil, nil, nil)
	eventBus := eventbus.NewManager()

	executor := NewExecutor(registry, sessMgr, eventBus)

	cmd := &Command{
		Name:        "test",
		Description: "Test command",
		Category:    "test",
		Handler:     func(ctx *ExecutionContext) error { return nil },
		Args:        []ArgDefinition{},
		Flags:       []FlagDefinition{},
	}

	err := executor.RegisterCommand(cmd)
	require.NoError(t, err)

	err = executor.Execute("test", []string{}, map[string]string{})
	assert.NoError(t, err)

	results := executor.ExecuteOnSessions([]string{"session1", "session2"}, "test", []string{})
	assert.Len(t, results, 2)
}
