package shell

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/creack/pty"
	"github.com/termbus/termbus/pkg/interfaces"
	"golang.org/x/crypto/ssh"
)

// PTYModel manages a PTY-backed SSH shell session.
type PTYModel struct {
	sessionID string
	sshClient *ssh.Client
	ptyFile   *os.File
	sshSess   *ssh.Session
	outputMu  sync.RWMutex
	output    string
	rows      int
	cols      int
	running   bool
}

// Connect binds a PTY to an SSH session for a session ID.
func (m *PTYModel) Connect(sessionID string, sessions interfaces.SessionManager) error {
	if sessions == nil {
		return fmt.Errorf("session manager is nil")
	}
	client, err := sessions.GetSSHClient(sessionID)
	if err != nil {
		return err
	}

	sshSess, err := client.NewSession()
	if err != nil {
		return err
	}

	master, slave, err := pty.Open()
	if err != nil {
		_ = sshSess.Close()
		return err
	}

	rows := m.rows
	cols := m.cols
	if rows <= 0 {
		rows = 24
	}
	if cols <= 0 {
		cols = 80
	}

	if err := sshSess.RequestPty("xterm-256color", rows, cols, ssh.TerminalModes{}); err != nil {
		_ = slave.Close()
		_ = master.Close()
		_ = sshSess.Close()
		return err
	}

	sshSess.Stdin = slave
	sshSess.Stdout = slave
	sshSess.Stderr = slave

	if err := sshSess.Shell(); err != nil {
		_ = slave.Close()
		_ = master.Close()
		_ = sshSess.Close()
		return err
	}
	_ = slave.Close()

	m.sessionID = sessionID
	m.sshClient = client
	m.ptyFile = master
	m.sshSess = sshSess
	m.running = true

	return nil
}

// Resize resizes the underlying PTY.
func (m *PTYModel) Resize(rows, cols int) error {
	if m.ptyFile == nil {
		return fmt.Errorf("pty not connected")
	}
	if rows <= 0 || cols <= 0 {
		return fmt.Errorf("invalid size")
	}
	m.rows = rows
	m.cols = cols
	return pty.Setsize(m.ptyFile, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
}

// Write writes input bytes to the PTY.
func (m *PTYModel) Write(data []byte) error {
	if m.ptyFile == nil {
		return fmt.Errorf("pty not connected")
	}
	_, err := m.ptyFile.Write(data)
	return err
}

// Read reads a chunk of output from the PTY.
func (m *PTYModel) Read() (string, error) {
	if m.ptyFile == nil {
		return "", fmt.Errorf("pty not connected")
	}
	buf := make([]byte, 4096)
	read, err := m.ptyFile.Read(buf)
	if err != nil {
		if err == io.EOF {
			return "", nil
		}
		return "", err
	}
	chunk := string(buf[:read])

	m.outputMu.Lock()
	m.output += chunk
	m.outputMu.Unlock()

	return chunk, nil
}

// Stream continuously reads PTY output and forwards chunks.
func (m *PTYModel) Stream(handler func(string)) error {
	if handler == nil {
		return fmt.Errorf("handler is nil")
	}
	for m.running {
		chunk, err := m.Read()
		if err != nil {
			return err
		}
		if chunk != "" {
			handler(chunk)
		}
	}
	return nil
}

// Output returns the buffered output.
func (m *PTYModel) Output() string {
	m.outputMu.RLock()
	defer m.outputMu.RUnlock()
	return m.output
}

// Close closes the PTY and SSH session.
func (m *PTYModel) Close() error {
	var err error
	m.running = false
	if m.ptyFile != nil {
		_ = m.ptyFile.Close()
		m.ptyFile = nil
	}
	if m.sshSess != nil {
		err = m.sshSess.Close()
		m.sshSess = nil
	}
	return err
}
