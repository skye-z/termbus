package commandbar

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	input     textinput.Model
	width     int
	completer *Completer
	options   []string
	optionIdx int
}

// New creates a command bar model.
func New() Model {
	input := textinput.New()
	input.Placeholder = ":command"
	input.Prompt = ": "
	input.Focus()
	return Model{input: input, completer: NewCompleter()}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "tab" && m.completer != nil {
			m.options = m.completer.Complete(m.input.Value())
			m.optionIdx = 0
			if len(m.options) > 0 {
				m.input.SetValue(m.options[0])
			}
		}
		if msg.String() == "shift+tab" && len(m.options) > 0 {
			m.optionIdx--
			if m.optionIdx < 0 {
				m.optionIdx = len(m.options) - 1
			}
			m.input.SetValue(m.options[m.optionIdx])
		}
	}
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the command bar.
func (m Model) View() string {
	bar := m.input.View()
	if len(m.options) == 0 {
		return lipgloss.NewStyle().Width(m.width).Render(bar)
	}
	options := strings.Join(m.options, "  ")
	options = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(options)
	return lipgloss.JoinVertical(lipgloss.Left, options, lipgloss.NewStyle().Width(m.width).Render(bar))
}

// SetSize updates the width of the command bar.
func (m *Model) SetSize(width int) {
	m.width = width
	m.input.Width = width - 4
}

// SetPlaceholder sets the placeholder text.
func (m *Model) SetPlaceholder(value string) {
	m.input.Placeholder = value
}

// SetCompleter sets the completer instance.
func (m *Model) SetCompleter(completer *Completer) {
	m.completer = completer
}

func (m Model) Value() string {
	return m.input.Value()
}

// Reset clears the input value.
func (m *Model) Reset() {
	m.input.SetValue("")
	m.options = nil
	m.optionIdx = 0
}

// OptionsVisible returns whether completion options are visible.
func (m Model) OptionsVisible() bool {
	return len(m.options) > 0
}
