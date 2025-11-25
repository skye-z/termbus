package pluginrpc

// InitRequest represents plugin initialization request.
type InitRequest struct {
	SessionID string
	Config    map[string]string
}

// InitResponse represents plugin initialization response.
type InitResponse struct {
	Success bool
	Error   string
}

// ExecuteRequest represents plugin execution request.
type ExecuteRequest struct {
	Command string
	Args    []string
	Env     map[string]string
}

// ExecuteResponse represents plugin execution response.
type ExecuteResponse struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Error    string
}

// StopRequest represents plugin stop request.
type StopRequest struct {
	Force bool
}

// StopResponse represents plugin stop response.
type StopResponse struct {
	Success bool
	Error   string
}

// InfoResponse represents plugin info response.
type InfoResponse struct {
	Name        string
	Version     string
	Description string
	Author      string
}

// ManifestResponse represents plugin manifest response.
type ManifestResponse struct {
	Name         string
	Version      string
	Description  string
	Author       string
	Permissions  []string
	Commands     []string
	ConfigSchema map[string]string
}
