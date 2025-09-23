package styles

import "github.com/charmbracelet/lipgloss"

var (
	Border = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true)

	Title = lipgloss.NewStyle().Bold(true)

	Muted = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	Active = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
)

var EditorContainerStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder(), true).
	BorderForeground(lipgloss.Color("green"))

var EditorHeaderStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("green")).
	Foreground(lipgloss.Color("black")).
	Padding(0, 1)

var EditorFooterStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("blue")).
	Foreground(lipgloss.Color("white")).
	Padding(0, 1)

func GetEditorContainerStyle() lipgloss.Style {
	return EditorContainerStyle
}

func GetEditorHeaderStyle() lipgloss.Style {
	return EditorHeaderStyle
}

func GetEditorFooterStyle() lipgloss.Style {
	return EditorFooterStyle
}
