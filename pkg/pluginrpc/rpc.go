package pluginrpc

import (
	"context"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

// ServiceImpl defines plugin behavior.
type ServiceImpl interface {
	Init(ctx context.Context, sessionID string, config map[string]string) error
	Execute(ctx context.Context, command string, args []string, env map[string]string) (*ExecuteResponse, error)
	Stop(ctx context.Context, force bool) error
	Info(ctx context.Context) (*InfoResponse, error)
	Manifest(ctx context.Context) (*ManifestResponse, error)
}

// Service implements net/rpc server methods.
type Service struct {
	Impl ServiceImpl
}

// Init handles init RPC.
func (s *Service) Init(req *InitRequest, resp *InitResponse) error {
	err := s.Impl.Init(context.Background(), req.SessionID, req.Config)
	resp.Success = err == nil
	if err != nil {
		resp.Error = err.Error()
	}
	return nil
}

// Execute handles execute RPC.
func (s *Service) Execute(req *ExecuteRequest, resp *ExecuteResponse) error {
	result, err := s.Impl.Execute(context.Background(), req.Command, req.Args, req.Env)
	if err != nil {
		resp.Error = err.Error()
		return nil
	}
	*resp = *result
	return nil
}

// Stop handles stop RPC.
func (s *Service) Stop(req *StopRequest, resp *StopResponse) error {
	err := s.Impl.Stop(context.Background(), req.Force)
	resp.Success = err == nil
	if err != nil {
		resp.Error = err.Error()
	}
	return nil
}

// Info handles info RPC.
func (s *Service) Info(_ *struct{}, resp *InfoResponse) error {
	info, err := s.Impl.Info(context.Background())
	if err != nil {
		return err
	}
	*resp = *info
	return nil
}

// Manifest handles manifest RPC.
func (s *Service) Manifest(_ *struct{}, resp *ManifestResponse) error {
	manifest, err := s.Impl.Manifest(context.Background())
	if err != nil {
		return err
	}
	*resp = *manifest
	return nil
}

// Client implements plugin RPC calls.
type Client struct {
	rpc *rpc.Client
}

// NewClient creates a new RPC client wrapper.
func NewClient(rpcClient *rpc.Client) *Client {
	return &Client{rpc: rpcClient}
}

// Init calls plugin init.
func (c *Client) Init(req *InitRequest) (*InitResponse, error) {
	var resp InitResponse
	err := c.rpc.Call("Plugin.Init", req, &resp)
	return &resp, err
}

// Execute calls plugin execute.
func (c *Client) Execute(req *ExecuteRequest) (*ExecuteResponse, error) {
	var resp ExecuteResponse
	err := c.rpc.Call("Plugin.Execute", req, &resp)
	return &resp, err
}

// Stop calls plugin stop.
func (c *Client) Stop(req *StopRequest) (*StopResponse, error) {
	var resp StopResponse
	err := c.rpc.Call("Plugin.Stop", req, &resp)
	return &resp, err
}

// Info calls plugin info.
func (c *Client) Info() (*InfoResponse, error) {
	var resp InfoResponse
	err := c.rpc.Call("Plugin.Info", struct{}{}, &resp)
	return &resp, err
}

// Manifest calls plugin manifest.
func (c *Client) Manifest() (*ManifestResponse, error) {
	var resp ManifestResponse
	err := c.rpc.Call("Plugin.Manifest", struct{}{}, &resp)
	return &resp, err
}

// Bridge implements go-plugin net/rpc bridge.
type Bridge struct {
	Impl ServiceImpl
}

// Server returns rpc service.
func (p *Bridge) Server(*plugin.MuxBroker) (interface{}, error) {
	return &Service{Impl: p.Impl}, nil
}

// Client returns rpc client wrapper.
func (p *Bridge) Client(_ *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return NewClient(c), nil
}

// Serve runs the plugin RPC server.
func Serve(impl ServiceImpl) {
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "TERMBUS_PLUGIN",
			MagicCookieValue: "1",
		},
		Plugins: map[string]plugin.Plugin{
			"plugin": &Bridge{Impl: impl},
		},
	})
}
