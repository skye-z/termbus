package modal

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Button represents a modal button.
type Button struct {
	Label  string
	Action func()
}

// Modal represents a modal dialog.
type Modal struct {
	Title     string
	Content   string
	Buttons   []Button
	ActiveIdx int
	OnConfirm func()
	OnCancel  func()
}

// View renders the modal.
func (m Modal) View(width int) string {
	box := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Width(width)
	buttons := make([]string, 0, len(m.Buttons))
	for i, btn := range m.Buttons {
		label := btn.Label
		if i == m.ActiveIdx {
			label = lipgloss.NewStyle().Bold(true).Render("[" + label + "]")
		}
		buttons = append(buttons, label)
	}
	footer := strings.Join(buttons, "  ")
	body := m.Title + "\n" + m.Content
	if footer != "" {
		body += "\n" + footer
	}
	return box.Render(body)
}

// Confirm triggers the confirm action.
func (m Modal) Confirm() {
	if m.OnConfirm != nil {
		m.OnConfirm()
	}
	if m.ActiveIdx >= 0 && m.ActiveIdx < len(m.Buttons) {
		if m.Buttons[m.ActiveIdx].Action != nil {
			m.Buttons[m.ActiveIdx].Action()
		}
	}
}

// Cancel triggers the cancel action.
func (m Modal) Cancel() {
	if m.OnCancel != nil {
		m.OnCancel()
	}
}

// Next selects the next button.
func (m *Modal) Next() {
	if len(m.Buttons) == 0 {
		return
	}
	m.ActiveIdx = (m.ActiveIdx + 1) % len(m.Buttons)
}

// Prev selects the previous button.
func (m *Modal) Prev() {
	if len(m.Buttons) == 0 {
		return
	}
	m.ActiveIdx--
	if m.ActiveIdx < 0 {
		m.ActiveIdx = len(m.Buttons) - 1
	}
}
