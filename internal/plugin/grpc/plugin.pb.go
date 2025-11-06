package grpc

import (
	context "context"
)

// InitRequest represents plugin initialization request.
type InitRequest struct {
	SessionId string
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
	ExitCode int32
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

// InfoRequest represents plugin info request.
type InfoRequest struct{}

// InfoResponse represents plugin info response.
type InfoResponse struct {
	Name        string
	Version     string
	Description string
	Author      string
}

// ManifestRequest represents plugin manifest request.
type ManifestRequest struct{}

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

// PluginClient defines the plugin gRPC client interface.
type PluginClient interface {
	Init(ctx context.Context, in *InitRequest) (*InitResponse, error)
	Execute(ctx context.Context, in *ExecuteRequest) (*ExecuteResponse, error)
	Stop(ctx context.Context, in *StopRequest) (*StopResponse, error)
	Info(ctx context.Context, in *InfoRequest) (*InfoResponse, error)
	Manifest(ctx context.Context, in *ManifestRequest) (*ManifestResponse, error)
}
