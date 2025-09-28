package shell

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/termbus/termbus/pkg/interfaces"
)

type Model struct {
	Title   string
	Content string
	Width   int
	Height  int
	Session string

	pty      *PTYModel
	renderer *ANSIRenderer
	eventBus interfaces.EventBus
	enhance  *Enhancement
}

// New creates a shell view model.
func New(title string, eventBus interfaces.EventBus) Model {
	return Model{
		Title:    title,
		Content:  "",
		renderer: NewANSIRenderer(),
		eventBus: eventBus,
		enhance:  &Enhancement{},
	}
}

// View renders the shell content.
func (m Model) View() string {
	box := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)
	content := m.Content
	if m.enhance != nil && m.Height > 2 {
		content = m.enhance.Visible(m.Height - 2)
	}
	if content == "" {
		content = "(shell not connected)"
	}
	body := fmt.Sprintf("%s\n%s", m.Title, content)
	return box.Width(m.Width).Height(m.Height).Render(body)
}

// Init initializes the shell view.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles shell input events.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "pgup":
			if m.enhance != nil {
				m.enhance.ScrollPage(m.Height-2, -1)
			}
		case "pgdown":
			if m.enhance != nil {
				m.enhance.ScrollPage(m.Height-2, 1)
			}
		case "ctrl+shift+c":
			if m.enhance != nil {
				m.enhance.Copy(m.Content)
			}
		case "ctrl+shift+v":
			if m.enhance != nil && m.pty != nil {
				_ = m.pty.Write([]byte(m.enhance.Paste()))
			}
		default:
			if m.pty != nil {
				_ = m.pty.Write([]byte(msg.String()))
			}
		}
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	}
	return m, nil
}

// Bind connects the shell to a PTY model.
func (m *Model) Bind(pty *PTYModel, sessionID string) {
	m.pty = pty
	m.Session = sessionID
}

// AppendOutput appends output to the shell view.
func (m *Model) AppendOutput(output string) {
	if m.renderer != nil {
		rendered := m.renderer.Render(output)
		m.Content += rendered
		if m.enhance != nil {
			m.enhance.Append(rendered)
		}
	} else {
		m.Content += output
		if m.enhance != nil {
			m.enhance.Append(output)
		}
	}
	if m.eventBus != nil {
		m.eventBus.Publish("shell.output", m.Session, output)
	}
}

func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
	if m.pty != nil {
		rows := height - 2
		cols := width - 2
		_ = m.pty.Resize(rows, cols)
		if m.eventBus != nil {
			m.eventBus.Publish("shell.resized", m.Session, rows, cols)
		}
	}
}
