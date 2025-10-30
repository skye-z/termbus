package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandRegistry(t *testing.T) {
	registry := NewCommandRegistry()

	cmd := &Command{
		Name:        "test",
		Aliases:     []string{"t"},
		Description: "Test command",
		Category:    "test",
		Handler:     func(ctx *ExecutionContext) error { return nil },
		Args:        []ArgDefinition{},
		Flags:       []FlagDefinition{},
		Examples:    []string{":test"},
	}

	err := registry.Register(cmd)
	assert.NoError(t, err)

	retrieved, err := registry.Get("test")
	assert.NoError(t, err)
	assert.Equal(t, "test", retrieved.Name)

	retrieved, err = registry.Get("t")
	assert.NoError(t, err)
	assert.Equal(t, "test", retrieved.Name)

	list := registry.List("test")
	assert.Len(t, list, 1)

	search := registry.Search("test")
	assert.Len(t, search, 1)
}

func TestCommandParser(t *testing.T) {
	registry := NewCommandRegistry()
	parser := NewCommandParser(registry)

	cmd, err := parser.Parse(":test --flag value arg1 arg2")
	assert.NoError(t, err)
	assert.Equal(t, ":test", cmd.Command)
	assert.Equal(t, "value", cmd.Flags["flag"])
	assert.Equal(t, []string{"arg1", "arg2"}, cmd.Args)
}

func TestAliasManager(t *testing.T) {
	aliasMgr := NewAliasManager()

	err := aliasMgr.Add("ls", "ls -la")
	assert.NoError(t, err)

	cmd, err := aliasMgr.Get("ls")
	assert.NoError(t, err)
	assert.Equal(t, "ls -la", cmd)
}

func TestHistoryManager(t *testing.T) {
	cfg := &HistoryConfig{MaxSize: 100}
	histMgr := NewHistoryManager(cfg)

	err := histMgr.Add("test command", "session-1")
	assert.NoError(t, err)

	entries := histMgr.Get("session-1", 10)
	assert.Len(t, entries, 1)
	assert.Equal(t, "test command", entries[0].Command)

	search := histMgr.Search("test", "session-1")
	assert.Len(t, search, 1)
}

func TestGroupManager(t *testing.T) {
	groupMgr := NewGroupManager()

	group, err := groupMgr.Create("test-group", []string{"host1", "host2"})
	assert.NoError(t, err)
	assert.Equal(t, "test-group", group.Name)
	assert.Len(t, group.Hosts, 2)

	retrieved, err := groupMgr.Get("test-group")
	assert.NoError(t, err)
	assert.Equal(t, "test-group", retrieved.Name)

	list := groupMgr.List()
	assert.Len(t, list, 1)
}
