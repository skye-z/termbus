package modal

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmModal represents a confirmation modal.
type ConfirmModal struct {
	Title   string
	Message string
	OnYes   func()
	OnNo    func()
	Active  bool
}

// View renders the confirmation modal.
func (m ConfirmModal) View(width int) string {
	box := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Width(width)
	body := m.Title + "\n" + m.Message
	return box.Render(body)
}

// Update handles yes/no confirmation.
func (m ConfirmModal) Update(msg tea.Msg) (ConfirmModal, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.Active && m.OnYes != nil {
				m.OnYes()
			}
		case "esc":
			if m.Active && m.OnNo != nil {
				m.OnNo()
			}
		}
	}
	return m, cmd
}
