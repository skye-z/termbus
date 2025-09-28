package shell

import "strings"

// Enhancement manages scrollback, search, and copy buffer.
type Enhancement struct {
	lines        []string
	scrollOffset int
	copyBuffer   string
}

// Append adds new output to scrollback.
func (e *Enhancement) Append(output string) {
	if output == "" {
		return
	}
	parts := strings.Split(output, "\n")
	e.lines = append(e.lines, parts...)
}

// Scroll moves the scroll offset.
func (e *Enhancement) Scroll(delta int) {
	e.scrollOffset += delta
	if e.scrollOffset < 0 {
		e.scrollOffset = 0
	}
	if e.scrollOffset > len(e.lines) {
		e.scrollOffset = len(e.lines)
	}
}

// ScrollPage scrolls by a full page.
func (e *Enhancement) ScrollPage(height int, direction int) {
	if height <= 0 {
		return
	}
	step := height
	if direction < 0 {
		step = -height
	}
	e.Scroll(step)
}

// Visible returns visible lines within height.
func (e *Enhancement) Visible(height int) string {
	if height <= 0 {
		return ""
	}
	start := len(e.lines) - height - e.scrollOffset
	if start < 0 {
		start = 0
	}
	end := start + height
	if end > len(e.lines) {
		end = len(e.lines)
	}
	return strings.Join(e.lines[start:end], "\n")
}

// Search returns matching lines.
func (e *Enhancement) Search(query string) []string {
	if query == "" {
		return nil
	}
	results := make([]string, 0)
	for _, line := range e.lines {
		if strings.Contains(line, query) {
			results = append(results, line)
		}
	}
	return results
}

// Copy stores selection into buffer.
func (e *Enhancement) Copy(selection string) {
	e.copyBuffer = selection
}

// Paste returns current copy buffer.
func (e *Enhancement) Paste() string {
	return e.copyBuffer
}
