package tabs

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type Tab struct {
	ID      string
	Title   string
	Active  bool
	HasWarn bool
}

type Model struct {
	tabs []Tab
}

func New(tabs []Tab) Model {
	return Model{tabs: tabs}
}

func (m Model) View(width int) string {
	activeStyle := lipgloss.NewStyle().Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203"))

	parts := make([]string, 0, len(m.tabs))
	for _, tab := range m.tabs {
		label := tab.Title
		if tab.HasWarn {
			label = warnStyle.Render(label)
		}
		if tab.Active {
			label = activeStyle.Render(label)
		} else {
			label = inactiveStyle.Render(label)
		}
		parts = append(parts, label)
	}

	content := strings.Join(parts, "  ")
	return lipgloss.NewStyle().Width(width).Render(content)
}

func (m Model) SetTabs(tabs []Tab) {
	m.tabs = tabs
}
