package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/termbus/termbus/pkg/interfaces"
	"github.com/termbus/termbus/pkg/types"
	"github.com/termbus/termbus/tui/styles"
)

type TunnelListModel struct {
	tunnels   []*types.ForwardTunnel
	selected  int
	sessionID string
	width     int
	height    int
	tunnelMgr interfaces.TunnelManager
}

func NewTunnelList(tunnelMgr interfaces.TunnelManager, width, height int) TunnelListModel {
	return TunnelListModel{
		tunnelMgr: tunnelMgr,
		tunnels:   []*types.ForwardTunnel{},
		selected:  0,
		width:     width,
		height:    height,
	}
}

func (m *TunnelListModel) Refresh(sessionID string) error {
	m.sessionID = sessionID

	tunnels, err := m.tunnelMgr.ListTunnels(sessionID)
	if err != nil {
		return fmt.Errorf("failed to list tunnels: %w", err)
	}

	m.tunnels = tunnels
	if m.selected >= len(m.tunnels) && len(m.tunnels) > 0 {
		m.selected = len(m.tunnels) - 1
	}

	return nil
}

func (m *TunnelListModel) GetSelected() *types.ForwardTunnel {
	if m.selected >= 0 && m.selected < len(m.tunnels) {
		return m.tunnels[m.selected]
	}
	return nil
}

func (m *TunnelListModel) StartSelected() error {
	tunnel := m.GetSelected()
	if tunnel == nil {
		return fmt.Errorf("no tunnel selected")
	}

	return m.tunnelMgr.StartTunnel(tunnel.ID)
}

func (m *TunnelListModel) StopSelected() error {
	tunnel := m.GetSelected()
	if tunnel == nil {
		return fmt.Errorf("no tunnel selected")
	}

	return m.tunnelMgr.StopTunnel(tunnel.ID)
}

func (m *TunnelListModel) DeleteSelected() error {
	tunnel := m.GetSelected()
	if tunnel == nil {
		return fmt.Errorf("no tunnel selected")
	}

	err := m.tunnelMgr.DeleteTunnel(tunnel.ID)
	if err != nil {
		return err
	}

	m.tunnels = append(m.tunnels[:m.selected], m.tunnels[m.selected+1:]...)
	if m.selected >= len(m.tunnels) && len(m.tunnels) > 0 {
		m.selected = len(m.tunnels) - 1
	}

	return nil
}

func (m *TunnelListModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *TunnelListModel) Init() tea.Cmd {
	return nil
}

func (m *TunnelListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.tunnels)-1 {
				m.selected++
			}
		case "enter":
			tunnel := m.GetSelected()
			if tunnel != nil {
				if tunnel.Status == types.TunnelStatusRunning {
					m.tunnelMgr.StopTunnel(tunnel.ID)
				} else {
					m.tunnelMgr.StartTunnel(tunnel.ID)
				}
			}
		case "d":
			m.DeleteSelected()
		case "n":
			return m, func() tea.Msg { return newTunnelMsg{} }
		}
	}

	return m, nil
}

func (m *TunnelListModel) View() string {
	var content strings.Builder

	header := styles.GetEditorHeaderStyle().
		Width(m.width - 2).
		Render("Tunnels")

	content.WriteString(header + "\n")

	if len(m.tunnels) == 0 {
		content.WriteString(styles.Muted.Render("  No tunnels. Press 'n' to create one.\n"))
	} else {
		content.WriteString(fmt.Sprintf("  %-30s %-10s %-15s %s\n",
			"Type", "Status", "Local", "Remote"))
		content.WriteString(styles.Muted.Render("  " + strings.Repeat("-", 70) + "\n"))

		for i, tunnel := range m.tunnels {
			prefix := "  "
			if i == m.selected {
				prefix = styles.Active.Render("> ")
			}

			statusStr := string(tunnel.Status)
			if tunnel.Status == types.TunnelStatusRunning {
				statusStr = styles.Active.Render("● Running")
			} else {
				statusStr = styles.Muted.Render("○ Stopped")
			}

			content.WriteString(fmt.Sprintf("%s%-30s %-10s %-15s %s\n",
				prefix,
				tunnel.Type,
				statusStr,
				tunnel.LocalAddr,
				tunnel.RemoteAddr,
			))
		}
	}

	footer := styles.GetEditorFooterStyle().
		Width(m.width - 2).
		Render("↑↓: Navigate | Enter: Toggle | d: Delete | n: New | q: Quit")

	container := styles.GetEditorContainerStyle().Width(m.width).Height(m.height)
	return container.Render(content.String() + footer)
}

type newTunnelMsg struct{}
type editTunnelMsg struct {
	tunnel *types.ForwardTunnel
}
