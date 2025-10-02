package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/termbus/termbus/pkg/interfaces"
	"github.com/termbus/termbus/pkg/types"
	"github.com/termbus/termbus/tui/styles"
)

type TunnelEditModel struct {
	tunnel      *types.ForwardTunnel
	selected    int
	sessionID   string
	width       int
	height      int
	tunnelMgr   interfaces.TunnelManager
	inputType   string
	inputLocal  string
	inputRemote string
	step        int
}

func NewTunnelEdit(tunnelMgr interfaces.TunnelManager, width, height int) TunnelEditModel {
	return TunnelEditModel{
		tunnelMgr:   tunnelMgr,
		tunnel:      &types.ForwardTunnel{},
		selected:    0,
		inputType:   "local",
		inputLocal:  "localhost:8080",
		inputRemote: "localhost:80",
		step:        0,
		width:       width,
		height:      height,
	}
}

func (m *TunnelEditModel) SetTunnel(t *types.ForwardTunnel) {
	m.tunnel = t
	m.inputType = string(t.Type)
	m.inputLocal = t.LocalAddr
	m.inputRemote = t.RemoteAddr
}

func (m *TunnelEditModel) Validate() error {
	if m.inputType != "local" && m.inputType != "remote" && m.inputType != "dynamic" {
		return fmt.Errorf("invalid tunnel type: %s", m.inputType)
	}

	if m.inputLocal == "" {
		return fmt.Errorf("local address is required")
	}

	if m.inputRemote == "" && m.inputType != "dynamic" {
		return fmt.Errorf("remote address is required")
	}

	return nil
}

func (m *TunnelEditModel) GetTunnel() *types.ForwardTunnel {
	m.tunnel.Type = types.ForwardType(m.inputType)
	m.tunnel.LocalAddr = m.inputLocal
	m.tunnel.RemoteAddr = m.inputRemote
	m.tunnel.Status = types.TunnelStatusStopped
	return m.tunnel
}

func (m *TunnelEditModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *TunnelEditModel) Init() tea.Cmd {
	return nil
}

func (m *TunnelEditModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < 3 {
				m.selected++
			}
		case "enter":
			m.step++
		case "tab":
			m.step = (m.step + 1) % 4
		case "esc":
			return m, func() tea.Msg { return cancelEditMsg{} }
		}
	}

	return m, nil
}

func (m *TunnelEditModel) View() string {
	var content strings.Builder

	header := styles.GetEditorHeaderStyle().
		Width(m.width - 2).
		Render("Create/Edit Tunnel")

	content.WriteString(header + "\n\n")

	tunnelTypes := []string{"local", "remote", "dynamic"}
	for i, tt := range tunnelTypes {
		prefix := "  "
		if m.step == 0 && m.selected == i {
			prefix = styles.Active.Render("> ")
		}
		if m.inputType == tt {
			content.WriteString(prefix + fmt.Sprintf("[%s] %s\n", tt, getTunnelTypeDesc(tt)))
		} else {
			content.WriteString(prefix + fmt.Sprintf("[ ] %s\n", tt))
		}
	}

	content.WriteString("\n")
	localPrefix := "  "
	if m.step == 1 {
		localPrefix = styles.Active.Render("> ")
	}
	content.WriteString(fmt.Sprintf("%sLocal Address: %s\n", localPrefix, m.inputLocal))

	remotePrefix := "  "
	if m.step == 2 {
		remotePrefix = styles.Active.Render("> ")
	}
	if m.inputType != "dynamic" {
		content.WriteString(fmt.Sprintf("%sRemote Address: %s\n", remotePrefix, m.inputRemote))
	} else {
		content.WriteString(fmt.Sprintf("%sRemote Address: (auto SOCKS5)\n", remotePrefix))
	}

	content.WriteString("\n")
	savePrefix := "  "
	if m.step == 3 {
		savePrefix = styles.Active.Render("> ")
	}
	content.WriteString(fmt.Sprintf("%s[Save] Create Tunnel\n", savePrefix))

	footer := styles.GetEditorFooterStyle().
		Width(m.width - 2).
		Render("↑↓: Select | Enter: Confirm | Tab: Next | Esc: Cancel")

	container := styles.GetEditorContainerStyle().Width(m.width).Height(m.height)
	return container.Render(content.String() + footer)
}

func getTunnelTypeDesc(t string) string {
	switch t {
	case "local":
		return "Local port forward (-L)"
	case "remote":
		return "Remote port forward (-R)"
	case "dynamic":
		return "Dynamic SOCKS5 proxy (-D)"
	default:
		return ""
	}
}

type cancelEditMsg struct{}
type saveTunnelMsg struct {
	tunnel *types.ForwardTunnel
}
