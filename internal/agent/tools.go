package agent

import (
	"fmt"
	"strings"
	"time"

	"github.com/termbus/termbus/pkg/interfaces"
	"github.com/termbus/termbus/pkg/types"
)

type SSHTool struct {
	sessionMgr interfaces.SessionManager
}

func NewSSHTool(sessionMgr interfaces.SessionManager) *SSHTool {
	return &SSHTool{sessionMgr: sessionMgr}
}

func (t *SSHTool) Name() string {
	return "ssh_exec"
}

func (t *SSHTool) Description() string {
	return "Execute command on remote server via SSH"
}

func (t *SSHTool) Parameters() []ToolParameter {
	return []ToolParameter{
		{Name: "session_id", Type: "string", Description: "SSH session ID", Required: true},
		{Name: "command", Type: "string", Description: "Command to execute", Required: true},
		{Name: "timeout", Type: "integer", Description: "Timeout in seconds", Required: false, Default: 30},
	}
}

func (t *SSHTool) Execute(agent *Agent, params map[string]interface{}) (*ToolResult, error) {
	sessionID, ok := params["session_id"].(string)
	if !ok {
		return &ToolResult{Success: false, Error: "missing session_id"}, nil
	}

	command, ok := params["command"].(string)
	if !ok {
		return &ToolResult{Success: false, Error: "missing command"}, nil
	}

	timeout := 30
	if to, ok := params["timeout"].(float64); ok {
		timeout = int(to)
	}

	client, err := t.sessionMgr.GetSSHClient(sessionID)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	session, err := client.NewSession()
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}
	defer session.Close()

	done := make(chan error, 1)
	go func() {
		_, err = session.CombinedOutput(command)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return &ToolResult{Success: false, Error: err.Error()}, nil
		}
	case <-time.After(time.Duration(timeout) * time.Second):
		return &ToolResult{Success: false, Error: "timeout"}, nil
	}

	output, _ := session.CombinedOutput(command)
	return &ToolResult{Success: true, Output: string(output), Data: map[string]interface{}{"exit_code": 0}}, nil
}

type SFTPReadTool struct {
	sftpMgr SFTPTool
}

func NewSFTPReadTool(sftpMgr SFTPTool) *SFTPReadTool {
	return &SFTPReadTool{sftpMgr: sftpMgr}
}

func (t *SFTPReadTool) Name() string {
	return "sftp_read"
}

func (t *SFTPReadTool) Description() string {
	return "Read file content from remote server via SFTP"
}

func (t *SFTPReadTool) Parameters() []ToolParameter {
	return []ToolParameter{
		{Name: "session_id", Type: "string", Description: "SSH session ID", Required: true},
		{Name: "path", Type: "string", Description: "Remote file path", Required: true},
	}
}

func (t *SFTPReadTool) Execute(agent *Agent, params map[string]interface{}) (*ToolResult, error) {
	sessionID, _ := params["session_id"].(string)
	path, _ := params["path"].(string)

	content, err := t.sftpMgr.ReadFile(sessionID, path)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{Success: true, Output: content}, nil
}

type SFTPWriteTool struct {
	sftpMgr SFTPTool
}

func NewSFTPWriteTool(sftpMgr SFTPTool) *SFTPWriteTool {
	return &SFTPWriteTool{sftpMgr: sftpMgr}
}

func (t *SFTPWriteTool) Name() string {
	return "sftp_write"
}

func (t *SFTPWriteTool) Description() string {
	return "Write content to remote file via SFTP"
}

func (t *SFTPWriteTool) Parameters() []ToolParameter {
	return []ToolParameter{
		{Name: "session_id", Type: "string", Description: "SSH session ID", Required: true},
		{Name: "path", Type: "string", Description: "Remote file path", Required: true},
		{Name: "content", Type: "string", Description: "File content to write", Required: true},
	}
}

func (t *SFTPWriteTool) Execute(agent *Agent, params map[string]interface{}) (*ToolResult, error) {
	sessionID, _ := params["session_id"].(string)
	path, _ := params["path"].(string)
	content, _ := params["content"].(string)

	err := t.sftpMgr.WriteFile(sessionID, path, content)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	return &ToolResult{Success: true, Output: "File written successfully"}, nil
}

type FileListTool struct {
	sftpMgr SFTPTool
}

func NewFileListTool(sftpMgr SFTPTool) *FileListTool {
	return &FileListTool{sftpMgr: sftpMgr}
}

func (t *FileListTool) Name() string {
	return "file_list"
}

func (t *FileListTool) Description() string {
	return "List files in remote directory"
}

func (t *FileListTool) Parameters() []ToolParameter {
	return []ToolParameter{
		{Name: "session_id", Type: "string", Description: "SSH session ID", Required: true},
		{Name: "path", Type: "string", Description: "Remote directory path", Required: false, Default: "/"},
	}
}

func (t *FileListTool) Execute(agent *Agent, params map[string]interface{}) (*ToolResult, error) {
	sessionID, _ := params["session_id"].(string)
	path := "/"
	if p, ok := params["path"].(string); ok {
		path = p
	}

	files, err := t.sftpMgr.List(sessionID, path)
	if err != nil {
		return &ToolResult{Success: false, Error: err.Error()}, nil
	}

	var fileList []string
	for _, f := range files {
		fileList = append(fileList, fmt.Sprintf("%s\t%d\t%s", f.Name, f.Size, f.ModTime.Format(time.RFC3339)))
	}

	return &ToolResult{Success: true, Output: strings.Join(fileList, "\n")}, nil
}

type SFTPTool interface {
	ReadFile(sessionID string, path string) (string, error)
	WriteFile(sessionID string, path string, content string) error
	List(sessionID string, path string) ([]types.FileInfo, error)
}

func GetTools(llmClient LLMClient, sessionMgr interfaces.SessionManager, sftpMgr SFTPTool) []Tool {
	return []Tool{
		NewSSHTool(sessionMgr),
		NewSFTPReadTool(sftpMgr),
		NewSFTPWriteTool(sftpMgr),
		NewFileListTool(sftpMgr),
	}
}
