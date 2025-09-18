package tunnel

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/termbus/termbus/internal/logger"
	"github.com/termbus/termbus/pkg/types"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type SOCKSConfig struct {
	Username  string
	Password  string
	AllowIPs  []string
	DenyIPs   []string
	RemoteDNS bool
}

func (m *TunnelManager) StartDynamicForwardWithConfig(sessionID string, config *SOCKSConfig) error {
	tunnel := &types.ForwardTunnel{
		Type:      types.ForwardTypeDynamic,
		LocalAddr: "127.0.0.1:1080",
		Status:    types.TunnelStatusStopped,
	}

	if config != nil && config.Username != "" {
		tunnel.Type = "dynamic-auth"
	}

	err := m.CreateTunnel(sessionID, tunnel)
	if err != nil {
		return err
	}

	sshClient, err := m.sessionManager.GetSSHClient(sessionID)
	if err != nil {
		return fmt.Errorf("failed to get ssh client: %w", err)
	}

	return m.startSOCKS5WithConfig(tunnel.ID, sshClient, config)
}

func (m *TunnelManager) startSOCKS5WithConfig(tunnelID string, client *ssh.Client, config *SOCKSConfig) error {
	m.mu.Lock()
	managed, exists := m.tunnels[tunnelID]
	if !exists {
		m.mu.Unlock()
		return ErrTunnelNotFound
	}
	m.mu.Unlock()

	listener, err := net.Listen("tcp", managed.LocalAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	managed.listener = listener
	managed.client = client
	managed.ctx, managed.cancel = context.WithCancel(m.ctx)
	managed.Status = types.TunnelStatusRunning

	go func() {
		for {
			select {
			case <-managed.ctx.Done():
				listener.Close()
				return
			default:
			}

			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-managed.ctx.Done():
					return
				default:
					continue
				}
			}

			go m.handleSOCKS5(conn, config)
		}
	}()

	logger.GetLogger().Info("SOCKS5 proxy started",
		zap.String("tunnel_id", tunnelID),
		zap.String("local_addr", managed.LocalAddr),
	)

	return nil
}

func (m *TunnelManager) handleSOCKS5(conn net.Conn, config *SOCKSConfig) {
	defer conn.Close()

	buf := make([]byte, 32*1024)
	n, err := conn.Read(buf)
	if err != nil || n < 2 {
		return
	}

	if buf[0] != 0x05 {
		conn.Write([]byte{0x05, 0xFF})
		return
	}

	methods := buf[2:n]
	hasAuth := config != nil && config.Username != ""

	var chosenAuth byte = 0x00
	for _, method := range methods {
		if hasAuth && method == 0x02 {
			chosenAuth = 0x02
			break
		}
		if method == 0x00 {
			chosenAuth = 0x00
		}
	}

	if chosenAuth == 0xFF {
		conn.Write([]byte{0x05, 0xFF})
		return
	}

	conn.Write([]byte{0x05, chosenAuth})

	if chosenAuth == 0x02 {
		if !m.handleSOCKSAuth(conn, config) {
			return
		}
	}

	n, err = conn.Read(buf)
	if err != nil || n < 7 {
		return
	}

	if buf[0] != 0x05 || buf[1] != 0x01 {
		return
	}

	remoteAddr, err := m.parseSOCKS5Target(buf[3:n])
	if err != nil {
		conn.Write([]byte{0x05, 0x01, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	if config != nil && !m.checkSOCKSAccess(remoteAddr.IP.String(), config) {
		conn.Write([]byte{0x05, 0x02, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}

	m.mu.RLock()
	managed := m.tunnels[m.findTunnelByConn(conn)]
	m.mu.RUnlock()

	var remoteConn net.Conn
	var dialErr error

	if config != nil && config.RemoteDNS && managed != nil {
		remoteConn, dialErr = managed.client.Dial("tcp", remoteAddr.String())
	} else {
		remoteConn, dialErr = net.Dial("tcp", remoteAddr.String())
	}

	if dialErr != nil {
		conn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer remoteConn.Close()

	bindAddr := remoteConn.LocalAddr().(*net.TCPAddr)
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01,
		byte(bindAddr.IP[0]), byte(bindAddr.IP[1]), byte(bindAddr.IP[2]), byte(bindAddr.IP[3]),
		byte(bindAddr.Port >> 8), byte(bindAddr.Port & 0xFF)})

	m.transferSOCKSData(conn, remoteConn)
}

func (m *TunnelManager) handleSOCKSAuth(conn net.Conn, config *SOCKSConfig) bool {
	buf := make([]byte, 32*1024)
	n, err := conn.Read(buf)
	if err != nil || n < 2 {
		return false
	}

	version := buf[0]
	if version != 0x01 {
		return false
	}

	usernameLen := int(buf[1])
	if n < 2+usernameLen+1 {
		return false
	}

	username := string(buf[2 : 2+usernameLen])
	passwordLen := int(buf[2+usernameLen])
	if n < 2+usernameLen+1+passwordLen {
		return false
	}

	password := string(buf[2+usernameLen+1 : 2+usernameLen+1+passwordLen])

	if username == config.Username && password == config.Password {
		conn.Write([]byte{0x01, 0x00})
		return true
	}

	conn.Write([]byte{0x01, 0x01})
	return false
}

func (m *TunnelManager) parseSOCKS5Target(data []byte) (*net.TCPAddr, error) {
	if len(data) < 7 {
		return nil, fmt.Errorf("invalid request")
	}

	var addr string
	var port int

	switch data[0] {
	case 0x01:
		ip := net.IP(data[1:5])
		port = int(data[5])<<8 + int(data[6])
		addr = fmt.Sprintf("%s:%d", ip.String(), port)
	case 0x03:
		domainLen := int(data[1])
		if len(data) < 2+domainLen+2 {
			return nil, fmt.Errorf("invalid domain length")
		}
		domain := string(data[2 : 2+domainLen])
		port = int(data[2+domainLen])<<8 + int(data[2+domainLen+1])
		addr = fmt.Sprintf("%s:%d", domain, port)
	default:
		return nil, fmt.Errorf("unsupported address type")
	}

	return net.ResolveTCPAddr("tcp", addr)
}

func (m *TunnelManager) checkSOCKSAccess(ip string, config *SOCKSConfig) bool {
	if len(config.DenyIPs) > 0 {
		for _, cidr := range config.DenyIPs {
			if matchCIDR(ip, cidr) {
				return false
			}
		}
	}

	if len(config.AllowIPs) > 0 {
		allowed := false
		for _, cidr := range config.AllowIPs {
			if matchCIDR(ip, cidr) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	return true
}

func (m *TunnelManager) findTunnelByConn(conn net.Conn) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id, managed := range m.tunnels {
		if managed.listener != nil {
			return id
		}
	}
	return ""
}

func (m *TunnelManager) transferSOCKSData(local, remote net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := local.Read(buf)
			if n > 0 {
				remote.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		wg.Done()
	}()

	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := remote.Read(buf)
			if n > 0 {
				local.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		wg.Done()
	}()

	wg.Wait()
}

func matchCIDR(ip, cidr string) bool {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return ip == cidr
	}
	return ipNet.Contains(net.ParseIP(ip))
}
