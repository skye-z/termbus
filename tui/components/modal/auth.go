package modal

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AuthModal represents an authentication modal with secure input.
type AuthModal struct {
	Title    string
	Prompt   string
	Input    textinput.Model
	OnSubmit func(value string)
}

// NewAuthModal creates a new auth modal.
func NewAuthModal(title, prompt string) AuthModal {
	input := textinput.New()
	input.Placeholder = prompt
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '*'
	input.Focus()
	return AuthModal{Title: title, Prompt: prompt, Input: input}
}

// View renders the auth modal.
func (m AuthModal) View(width int) string {
	box := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Width(width)
	body := m.Title + "\n" + m.Input.View()
	return box.Render(body)
}

// Update handles key events and submission.
func (m AuthModal) Update(msg tea.Msg) (AuthModal, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.OnSubmit != nil {
				m.OnSubmit(m.Input.Value())
			}
		}
	}
	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}
