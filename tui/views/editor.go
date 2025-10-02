package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/termbus/termbus/pkg/interfaces"
	"github.com/termbus/termbus/tui/styles"
)

type EditorModel struct {
	sessionID string
	filePath  string
	lines     []string
	cursor    int
	scroll    int
	modified  bool
	loading   bool
	err       error
	footer    string
	width     int
	height    int
	sftpMgr   interfaces.SFTPManager
}

func NewEditor(sftpMgr interfaces.SFTPManager, width, height int) EditorModel {
	return EditorModel{
		sftpMgr:  sftpMgr,
		lines:    []string{},
		width:    width,
		height:   height,
		footer:   "↑↓: Navigate | Enter: Edit | Ctrl+S: Save | Ctrl+Q: Quit",
		modified: false,
		cursor:   0,
		scroll:   0,
	}
}

func (m *EditorModel) Load(sessionID, path string) error {
	m.sessionID = sessionID
	m.filePath = path
	m.loading = true

	content, err := m.sftpMgr.ReadFile(sessionID, path)
	if err != nil {
		m.err = fmt.Errorf("failed to load file: %w", err)
		m.loading = false
		return m.err
	}

	m.lines = strings.Split(content, "\n")
	m.loading = false
	m.modified = false
	m.footer = fmt.Sprintf("File: %s | %d lines | Modified: no", path, len(m.lines))

	return nil
}

func (m *EditorModel) Save() error {
	if m.sessionID == "" || m.filePath == "" {
		return fmt.Errorf("no file loaded")
	}

	content := strings.Join(m.lines, "\n")
	err := m.sftpMgr.WriteFile(m.sessionID, m.filePath, content)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	m.modified = false
	m.footer = fmt.Sprintf("File: %s | %d lines | Saved!", m.filePath, len(m.lines))

	return nil
}

func (m *EditorModel) Validate() error {
	if m.filePath == "" {
		return fmt.Errorf("no file selected")
	}
	return nil
}

func (m *EditorModel) IsModified() bool {
	return m.modified
}

func (m *EditorModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *EditorModel) Init() tea.Cmd {
	return nil
}

func (m *EditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.lines)-1 {
				m.cursor++
			}
		case "ctrl+s":
			if err := m.Save(); err != nil {
				m.err = err
			}
		case "ctrl+q":
			return m, tea.Quit
		case "enter":
			m.modified = true
			m.footer = fmt.Sprintf("File: %s | %d lines | Modified: yes", m.filePath, len(m.lines))
		}
	}

	return m, nil
}

func (m *EditorModel) View() string {
	header := styles.GetEditorHeaderStyle().
		Width(m.width - 2).
		Render(fmt.Sprintf("Editing: %s", m.filePath))

	footer := styles.GetEditorFooterStyle().
		Width(m.width - 2).
		Render(m.footer)

	var content strings.Builder
	startLine := m.scroll
	endLine := m.scroll + m.height - 6
	if endLine > len(m.lines) {
		endLine = len(m.lines)
	}

	for i := startLine; i < endLine; i++ {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		content.WriteString(fmt.Sprintf("%s%4d  %s\n", prefix, i+1, m.lines[i]))
	}

	container := styles.GetEditorContainerStyle().Width(m.width).Height(m.height)

	return container.Render(header + "\n" + content.String() + footer)
}

type confirmQuitMsg struct{}
type confirmDiscardMsg struct{}
