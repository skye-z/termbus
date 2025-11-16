package grpc

import (
	context "context"
	"fmt"

	"github.com/hashicorp/go-plugin"
	grpc "google.golang.org/grpc"
)

// PluginGRPC implements go-plugin gRPC bridge.
type PluginGRPC struct{}

// GRPCServer is a no-op placeholder.
func (p *PluginGRPC) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	return nil
}

// GRPCClient creates a client for the plugin service.
func (p *PluginGRPC) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &noopClient{}, nil
}

type noopClient struct{}

func (c *noopClient) Init(ctx context.Context, in *InitRequest) (*InitResponse, error) {
	return nil, fmt.Errorf("grpc client not implemented")
}

func (c *noopClient) Execute(ctx context.Context, in *ExecuteRequest) (*ExecuteResponse, error) {
	return nil, fmt.Errorf("grpc client not implemented")
}

func (c *noopClient) Stop(ctx context.Context, in *StopRequest) (*StopResponse, error) {
	return nil, fmt.Errorf("grpc client not implemented")
}

func (c *noopClient) Info(ctx context.Context, in *InfoRequest) (*InfoResponse, error) {
	return nil, fmt.Errorf("grpc client not implemented")
}

func (c *noopClient) Manifest(ctx context.Context, in *ManifestRequest) (*ManifestResponse, error) {
	return nil, fmt.Errorf("grpc client not implemented")
}
