package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/termbus/termbus/internal/command"
)

func TestE2E_CommandRegistry(t *testing.T) {
	registry := command.NewCommandRegistry()

	testCmd := &command.Command{
		Name:        "test-cmd",
		Description: "Test command",
		Category:    "test",
		Handler: func(ctx *command.ExecutionContext) error {
			ctx.Output("test output")
			return nil
		},
	}

	err := registry.Register(testCmd)
	require.NoError(t, err)

	cmd, err := registry.Get("test-cmd")
	require.NoError(t, err)
	assert.Equal(t, "test-cmd", cmd.Name)

	list := registry.List("test")
	assert.Equal(t, 1, len(list))

	search := registry.Search("test")
	assert.GreaterOrEqual(t, len(search), 1)

	err = registry.Unregister("test-cmd")
	require.NoError(t, err)

	_, err = registry.Get("test-cmd")
	assert.Error(t, err)
}

func TestE2E_CommandParser(t *testing.T) {
	registry := command.NewCommandRegistry()
	parser := command.NewCommandParser(registry)

	testCmd := &command.Command{
		Name:        "echo",
		Description: "Echo command",
		Category:    "test",
		Handler: func(ctx *command.ExecutionContext) error {
			return nil
		},
	}
	registry.Register(testCmd)

	tests := []struct {
		name      string
		input     string
		wantCmd   string
		wantArgs  []string
		wantFlags map[string]string
		wantErr   bool
	}{
		{
			name:    "simple command",
			input:   "echo",
			wantCmd: "echo",
			wantErr: false,
		},
		{
			name:     "command with args",
			input:    "echo hello world",
			wantCmd:  "echo",
			wantArgs: []string{"hello", "world"},
			wantErr:  false,
		},
		{
			name:      "command with flags",
			input:     "echo --verbose",
			wantCmd:   "echo",
			wantFlags: map[string]string{"verbose": "true"},
			wantErr:   false,
		},
		{
			name:      "command with flag value",
			input:     "echo --name=test",
			wantCmd:   "echo",
			wantFlags: map[string]string{"name": "test"},
			wantErr:   false,
		},
		{
			name:    "empty input",
			input:   "",
			wantCmd: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantCmd != "" {
				assert.Equal(t, tt.wantCmd, parsed.Command)
			}
			if tt.wantArgs != nil {
				assert.Equal(t, tt.wantArgs, parsed.Args)
			}
			if tt.wantFlags != nil {
				assert.Equal(t, tt.wantFlags, parsed.Flags)
			}
		})
	}
}

func TestE2E_CommandAlias(t *testing.T) {
	aliasMgr := command.NewAliasManager()

	err := aliasMgr.Add("ll", "ls -la")
	require.NoError(t, err)

	expanded := aliasMgr.Expand("ll")
	assert.Equal(t, "ls -la", expanded)

	expanded = aliasMgr.Expand("echo hello")
	assert.Equal(t, "echo hello", expanded)

	list := aliasMgr.List()
	assert.Contains(t, list, "ll")

	_ = list

	err = aliasMgr.Remove("ll")
	require.NoError(t, err)

	_, err = aliasMgr.Get("ll")
	assert.Error(t, err)
}

func TestE2E_CommandHistory(t *testing.T) {
	historyMgr := command.NewHistoryManager(&command.HistoryConfig{
		MaxSize: 100,
	})

	err := historyMgr.Add("echo hello", "session-1")
	require.NoError(t, err)

	err = historyMgr.Add("ls -la", "session-1")
	require.NoError(t, err)

	entries := historyMgr.Get("session-1", 10)
	assert.Equal(t, 2, len(entries))

	entries = historyMgr.Get("session-1", 1)
	assert.Equal(t, 1, len(entries))

	searchResults := historyMgr.Search("echo", "session-1")
	assert.GreaterOrEqual(t, len(searchResults), 1)

	err = historyMgr.Clear("session-1")
	require.NoError(t, err)

	entries = historyMgr.Get("session-1", 10)
	assert.Equal(t, 0, len(entries))
}

func TestE2E_CommandGroup(t *testing.T) {
	groupMgr := command.NewGroupManager()

	hosts := []string{"host1", "host2", "host3"}
	group, err := groupMgr.Create("test-group", hosts)
	require.NoError(t, err)
	assert.Equal(t, "test-group", group.Name)
	assert.Equal(t, 3, len(group.Hosts))

	err = groupMgr.AddHost("test-group", "host4")
	require.NoError(t, err)

	group, err = groupMgr.Get("test-group")
	require.NoError(t, err)
	assert.Equal(t, 4, len(group.Hosts))

	err = groupMgr.RemoveHost("test-group", "host2")
	require.NoError(t, err)

	group, err = groupMgr.Get("test-group")
	require.NoError(t, err)
	assert.Equal(t, 3, len(group.Hosts))

	groups := groupMgr.List()
	assert.GreaterOrEqual(t, len(groups), 1)

	err = groupMgr.Delete("test-group")
	require.NoError(t, err)

	_, err = groupMgr.Get("test-group")
	assert.Error(t, err)
}

func TestE2E_BuildCommand(t *testing.T) {
	tests := []struct {
		name   string
		cmd    string
		args   []string
		flags  map[string]string
		expect string
	}{
		{
			name:   "simple command",
			cmd:    "echo",
			args:   nil,
			flags:  nil,
			expect: "echo",
		},
		{
			name:   "command with args",
			cmd:    "echo",
			args:   []string{"hello", "world"},
			flags:  nil,
			expect: "echo hello world",
		},
		{
			name:   "command with flags",
			cmd:    "echo",
			args:   nil,
			flags:  map[string]string{"verbose": ""},
			expect: "echo --verbose=",
		},
		{
			name:   "command with flag value",
			cmd:    "echo",
			args:   nil,
			flags:  map[string]string{"name": "test"},
			expect: "echo --name=test",
		},
		{
			name:   "full command",
			cmd:    "echo",
			args:   []string{"hello"},
			flags:  map[string]string{"verbose": "", "name": "world"},
			expect: "echo --name=world --verbose= hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := command.BuildCommand(tt.cmd, tt.args, tt.flags)
			if tt.name == "full command" {
				assert.Contains(t, result, "echo")
				assert.Contains(t, result, "hello")
				assert.Contains(t, result, "--name=world")
				assert.Contains(t, result, "--verbose=")
			} else {
				assert.Equal(t, tt.expect, result)
			}
		})
	}
}
