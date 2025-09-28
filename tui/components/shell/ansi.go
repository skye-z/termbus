package shell

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ANSIState stores the current rendering attributes.
type ANSIState struct {
	Bold      bool
	Underline bool
	Reverse   bool
	FG        lipgloss.Color
	BG        lipgloss.Color
}

// ANSIRenderer applies minimal ANSI styling to text output.
type ANSIRenderer struct {
	state ANSIState
}

// NewANSIRenderer creates a new renderer.
func NewANSIRenderer() *ANSIRenderer {
	return &ANSIRenderer{}
}

// Render applies ANSI control sequences to the incoming text.
func (r *ANSIRenderer) Render(input string) string {
	buf := make([]rune, 0, len(input))
	for i := 0; i < len(input); i++ {
		ch := input[i]
		if ch == '\r' {
			buf = truncateLine(buf)
			continue
		}
		if ch != '\x1b' {
			buf = append(buf, rune(ch))
			continue
		}
		if i+1 < len(input) && input[i+1] == '[' {
			end := i + 2
			for end < len(input) && !isFinal(input[end]) {
				end++
			}
			if end < len(input) {
				seq := input[i+2 : end]
				switch input[end] {
				case 'm':
					r.applySGR(seq)
				case 'J':
					buf = buf[:0]
				case 'H':
					buf = buf[:0]
				case 'K':
					buf = truncateLine(buf)
				}
				i = end
				continue
			}
		}
		buf = append(buf, rune(ch))
	}
	return r.applyStyle(string(buf))
}

func (r *ANSIRenderer) applySGR(seq string) {
	if seq == "" {
		r.reset()
		return
	}
	parts := strings.Split(seq, ";")
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		val := toInt(part)
		switch part {
		case "0":
			r.reset()
		case "1":
			r.state.Bold = true
		case "4":
			r.state.Underline = true
		case "7":
			r.state.Reverse = true
		case "22":
			r.state.Bold = false
		case "24":
			r.state.Underline = false
		case "27":
			r.state.Reverse = false
		case "30", "31", "32", "33", "34", "35", "36", "37":
			r.state.FG = lipgloss.Color(fmt.Sprintf("%d", val-30))
		case "39":
			r.state.FG = ""
		case "40", "41", "42", "43", "44", "45", "46", "47":
			r.state.BG = lipgloss.Color(fmt.Sprintf("%d", val-40))
		case "49":
			r.state.BG = ""
		case "90", "91", "92", "93", "94", "95", "96", "97":
			r.state.FG = lipgloss.Color(fmt.Sprintf("%d", (val-90)+8))
		case "100", "101", "102", "103", "104", "105", "106", "107":
			r.state.BG = lipgloss.Color(fmt.Sprintf("%d", (val-100)+8))
		case "38":
			if i+1 < len(parts) && parts[i+1] == "5" && i+2 < len(parts) {
				r.state.FG = lipgloss.Color(parts[i+2])
				i += 2
			} else if i+1 < len(parts) && parts[i+1] == "2" && i+4 < len(parts) {
				r.state.FG = lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", toByte(parts[i+2]), toByte(parts[i+3]), toByte(parts[i+4])))
				i += 4
			}
		case "48":
			if i+1 < len(parts) && parts[i+1] == "5" && i+2 < len(parts) {
				r.state.BG = lipgloss.Color(parts[i+2])
				i += 2
			} else if i+1 < len(parts) && parts[i+1] == "2" && i+4 < len(parts) {
				r.state.BG = lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", toByte(parts[i+2]), toByte(parts[i+3]), toByte(parts[i+4])))
				i += 4
			}
		}
	}
}

func (r *ANSIRenderer) applyStyle(text string) string {
	style := lipgloss.NewStyle()
	if r.state.Bold {
		style = style.Bold(true)
	}
	if r.state.Underline {
		style = style.Underline(true)
	}
	if r.state.Reverse {
		style = style.Reverse(true)
	}
	if r.state.FG != "" {
		style = style.Foreground(r.state.FG)
	}
	if r.state.BG != "" {
		style = style.Background(r.state.BG)
	}
	return style.Render(text)
}

func (r *ANSIRenderer) reset() {
	r.state = ANSIState{}
}

func isFinal(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func truncateLine(buf []rune) []rune {
	for i := len(buf) - 1; i >= 0; i-- {
		if buf[i] == '\n' {
			return buf[:i+1]
		}
	}
	return buf[:0]
}

func toByte(value string) byte {
	val := toInt(value)
	if val < 0 {
		val = 0
	}
	if val > 255 {
		val = 255
	}
	return byte(val)
}

func toInt(value string) int {
	var v int
	_, _ = fmt.Sscanf(value, "%d", &v)
	return v
}
