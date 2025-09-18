package tunnel

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/termbus/termbus/internal/logger"
	"github.com/termbus/termbus/pkg/interfaces"
	"github.com/termbus/termbus/pkg/types"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

var (
	ErrTunnelNotFound = fmt.Errorf("tunnel not found")
	ErrTunnelRunning  = fmt.Errorf("tunnel already running")
)

type TunnelManager struct {
	sessionManager interfaces.SessionManager
	tunnels        map[string]*ManagedTunnel
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

type ManagedTunnel struct {
	*types.ForwardTunnel
	listener net.Listener
	client   *ssh.Client
	ctx      context.Context
	cancel   context.CancelFunc
}

func NewTunnelManager(sessionManager interfaces.SessionManager) *TunnelManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &TunnelManager{
		sessionManager: sessionManager,
		tunnels:        make(map[string]*ManagedTunnel),
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (m *TunnelManager) CreateTunnel(sessionID string, tunnel *types.ForwardTunnel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tunnel.ID = fmt.Sprintf("tunnel_%d", time.Now().UnixNano())
	tunnel.SessionID = sessionID
	tunnel.Status = types.TunnelStatusStopped
	tunnel.CreatedAt = time.Now()

	m.tunnels[tunnel.ID] = &ManagedTunnel{
		ForwardTunnel: tunnel,
	}

	logger.GetLogger().Info("tunnel created",
		zap.String("tunnel_id", tunnel.ID),
		zap.String("session_id", sessionID),
	)

	return nil
}

func (m *TunnelManager) StartTunnel(tunnelID string) error {
	m.mu.Lock()
	managed, exists := m.tunnels[tunnelID]
	if !exists {
		m.mu.Unlock()
		return ErrTunnelNotFound
	}

	if managed.listener != nil {
		m.mu.Unlock()
		return ErrTunnelRunning
	}
	m.mu.Unlock()

	sshClient, err := m.sessionManager.GetSSHClient(managed.SessionID)
	if err != nil {
		return fmt.Errorf("failed to get ssh client: %w", err)
	}

	switch managed.Type {
	case types.ForwardTypeLocal:
		return m.startLocalForward(managed, sshClient)
	case types.ForwardTypeRemote:
		return m.startRemoteForward(managed, sshClient)
	case types.ForwardTypeDynamic:
		return m.startDynamicForward(managed, sshClient)
	default:
		return fmt.Errorf("unsupported tunnel type: %s", managed.Type)
	}
}

func (m *TunnelManager) startLocalForward(t *ManagedTunnel, client *ssh.Client) error {
	listener, err := net.Listen("tcp", t.LocalAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	t.listener = listener
	t.client = client
	t.ctx, t.cancel = context.WithCancel(m.ctx)
	t.Status = types.TunnelStatusRunning

	go func() {
		for {
			select {
			case <-t.ctx.Done():
				listener.Close()
				return
			default:
			}

			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-t.ctx.Done():
					return
				default:
					continue
				}
			}

			go m.handleLocalForward(t, conn)
		}
	}()

	logger.GetLogger().Info("local tunnel started",
		zap.String("tunnel_id", t.ID),
		zap.String("local_addr", t.LocalAddr),
		zap.String("remote_addr", t.RemoteAddr),
	)

	return nil
}

func (m *TunnelManager) handleLocalForward(t *ManagedTunnel, localConn net.Conn) {
	defer localConn.Close()

	remoteConn, err := t.client.Dial("tcp", t.RemoteAddr)
	if err != nil {
		logger.GetLogger().Error("failed to dial remote",
			zap.String("error", err.Error()),
		)
		return
	}
	defer remoteConn.Close()

	done := make(chan struct{}, 2)

	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := localConn.Read(buf)
			if n > 0 {
				remoteConn.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := remoteConn.Read(buf)
			if n > 0 {
				localConn.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	<-done
}

func (m *TunnelManager) startRemoteForward(t *ManagedTunnel, client *ssh.Client) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	t.client = client
	t.ctx, t.cancel = context.WithCancel(m.ctx)
	t.Status = types.TunnelStatusRunning

	go func() {
		<-t.ctx.Done()
		session.Close()
	}()

	logger.GetLogger().Info("remote tunnel started",
		zap.String("tunnel_id", t.ID),
		zap.String("remote_addr", t.RemoteAddr),
	)

	return nil
}

func (m *TunnelManager) startDynamicForward(t *ManagedTunnel, client *ssh.Client) error {
	listener, err := net.Listen("tcp", t.LocalAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	t.listener = listener
	t.client = client
	t.ctx, t.cancel = context.WithCancel(m.ctx)
	t.Status = types.TunnelStatusRunning

	go func() {
		for {
			select {
			case <-t.ctx.Done():
				listener.Close()
				return
			default:
			}

			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-t.ctx.Done():
					return
				default:
					continue
				}
			}

			go m.handleSOCKS(t, conn)
		}
	}()

	logger.GetLogger().Info("dynamic tunnel started",
		zap.String("tunnel_id", t.ID),
		zap.String("local_addr", t.LocalAddr),
	)

	return nil
}

func (m *TunnelManager) handleSOCKS(t *ManagedTunnel, conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 32*1024)
	n, err := conn.Read(buf)
	if err != nil || n < 2 {
		return
	}

	if buf[0] != 0x05 {
		return
	}

	conn.Write([]byte{0x05, 0x00})

	n, err = conn.Read(buf)
	if err != nil || n < 7 {
		return
	}

	var targetAddr string
	switch buf[1] {
	case 0x01:
		targetAddr = fmt.Sprintf("%s:%d", net.IP(buf[2:6]).String(), int(buf[6])<<8+int(buf[7]))
	case 0x03:
		domainLen := int(buf[3])
		targetAddr = fmt.Sprintf("%s:%d", string(buf[4:4+domainLen]), int(buf[4+domainLen])<<8+int(buf[4+domainLen+1]))
	}

	remoteConn, err := t.client.Dial("tcp", targetAddr)
	if err != nil {
		conn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer remoteConn.Close()

	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	done := make(chan struct{}, 2)

	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				remoteConn.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := remoteConn.Read(buf)
			if n > 0 {
				conn.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	<-done
}

func (m *TunnelManager) StopTunnel(tunnelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	managed, exists := m.tunnels[tunnelID]
	if !exists {
		return ErrTunnelNotFound
	}

	if managed.cancel != nil {
		managed.cancel()
	}

	if managed.listener != nil {
		managed.listener.Close()
		managed.listener = nil
	}

	managed.Status = types.TunnelStatusStopped

	logger.GetLogger().Info("tunnel stopped",
		zap.String("tunnel_id", tunnelID),
	)

	return nil
}

func (m *TunnelManager) DeleteTunnel(tunnelID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	managed, exists := m.tunnels[tunnelID]
	if !exists {
		return ErrTunnelNotFound
	}

	if managed.cancel != nil {
		managed.cancel()
	}

	if managed.listener != nil {
		managed.listener.Close()
	}

	delete(m.tunnels, tunnelID)

	logger.GetLogger().Info("tunnel deleted",
		zap.String("tunnel_id", tunnelID),
	)

	return nil
}

func (m *TunnelManager) ListTunnels(sessionID string) []*types.ForwardTunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*types.ForwardTunnel
	for _, managed := range m.tunnels {
		if sessionID == "" || managed.SessionID == sessionID {
			result = append(result, managed.ForwardTunnel)
		}
	}

	return result
}

func (m *TunnelManager) GetTunnel(tunnelID string) (*types.ForwardTunnel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	managed, exists := m.tunnels[tunnelID]
	if !exists {
		return nil, ErrTunnelNotFound
	}

	return managed.ForwardTunnel, nil
}

func (m *TunnelManager) Close() {
	m.cancel()
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, managed := range m.tunnels {
		if managed.cancel != nil {
			managed.cancel()
		}
		if managed.listener != nil {
			managed.listener.Close()
		}
	}
}
