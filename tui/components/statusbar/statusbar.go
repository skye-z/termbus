package statusbar

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	Left  string
	Right string
	Width int
}

func New() Model {
	return Model{}
}

func (m Model) View() string {
	left := m.Left
	if left == "" {
		left = "Termbus"
	}
	right := m.Right
	if right == "" {
		right = time.Now().Format("15:04:05")
	}

	space := m.Width - lipgloss.Width(left) - lipgloss.Width(right)
	if space < 1 {
		space = 1
	}
	gap := lipgloss.NewStyle().Width(space).Render("")
	return left + gap + right
}

func (m *Model) SetSize(width int) {
	m.Width = width
}
