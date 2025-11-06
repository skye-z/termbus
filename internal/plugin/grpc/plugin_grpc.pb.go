package grpc

import (
	context "context"
)

// PluginServer defines the server interface.
type PluginServer interface {
	Init(ctx context.Context, req *InitRequest) (*InitResponse, error)
	Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)
	Stop(ctx context.Context, req *StopRequest) (*StopResponse, error)
	Info(ctx context.Context, req *InfoRequest) (*InfoResponse, error)
	Manifest(ctx context.Context, req *ManifestRequest) (*ManifestResponse, error)
}
