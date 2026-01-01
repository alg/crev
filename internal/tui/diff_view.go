package tui

import (
	"fmt"
	"strings"

	"github.com/alg/crev/internal/diff"
	"github.com/charmbracelet/lipgloss"
)

// viewDiffView renders the diff view for the current file
func (m Model) viewDiffView(height int) string {
	file := m.currentFile()
	if file == nil {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(colorSecondary).
			Render("No file selected")
	}

	if file.IsBinary {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(colorSecondary).
			Render("Binary file - cannot display diff")
	}

	if len(file.Hunks) == 0 {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(colorSecondary).
			Render("No changes in this file")
	}

	var lines []string
	lineNum := 0
	viewHeight := height - 2 // Account for border

	// Build all lines first
	for hunkIdx, hunk := range file.Hunks {
		// Hunk header
		header := m.renderHunkHeader(hunk, lineNum == m.lineIndex)
		if lineNum >= m.scrollOffset && lineNum < m.scrollOffset+viewHeight {
			lines = append(lines, header)
		}
		lineNum++

		// Hunk lines
		for _, line := range hunk.Lines {
			rendered := m.renderDiffLine(file, hunkIdx, line, lineNum == m.lineIndex)
			if lineNum >= m.scrollOffset && lineNum < m.scrollOffset+viewHeight {
				lines = append(lines, rendered)
			}
			lineNum++
		}
	}

	content := strings.Join(lines, "\n")

	// Show scroll position
	totalLines := m.totalLinesInFile(file)
	if totalLines > viewHeight {
		scrollPct := float64(m.scrollOffset) / float64(totalLines-viewHeight) * 100
		if scrollPct > 100 {
			scrollPct = 100
		}
		scrollInfo := fmt.Sprintf(" Line %d/%d (%.0f%%) ", m.lineIndex+1, totalLines, scrollPct)
		content += "\n" + helpStyle.Render(scrollInfo)
	}

	return diffViewStyle.
		Width(m.width - 4).
		Height(height).
		Render(content)
}

func (m Model) renderHunkHeader(hunk diff.Hunk, selected bool) string {
	style := hunkHeaderStyle
	if selected {
		style = style.Background(lipgloss.Color("237"))
	}
	// Truncate header if too long
	header := hunk.Header
	maxLen := m.width - 10
	if len(header) > maxLen {
		header = header[:maxLen-3] + "..."
	}
	return style.Width(m.width - 8).Render(header)
}

func (m Model) renderDiffLine(file *diff.File, hunkIdx int, line diff.Line, selected bool) string {
	// Line numbers
	oldNum := "    "
	newNum := "    "
	if line.OldNum > 0 {
		oldNum = fmt.Sprintf("%4d", line.OldNum)
	}
	if line.NewNum > 0 {
		newNum = fmt.Sprintf("%4d", line.NewNum)
	}

	lineNums := lineNumStyle.Render(oldNum) + " " + lineNumStyle.Render(newNum)

	// Line prefix and content style
	var prefix string
	var contentStyle lipgloss.Style

	switch line.Type {
	case diff.LineAdded:
		prefix = "+"
		contentStyle = lineAddedStyle
	case diff.LineRemoved:
		prefix = "-"
		contentStyle = lineRemovedStyle
	default:
		prefix = " "
		contentStyle = lineContextStyle
	}

	// Check for comment on this line
	hasComment := false
	lineNum := line.NewNum
	side := "new"
	if line.Type == diff.LineRemoved {
		lineNum = line.OldNum
		side = "old"
	}

	for _, c := range m.review.Comments {
		if c.File == file.Path && c.LineStart == lineNum && c.Side == side {
			hasComment = true
			break
		}
	}

	// Comment indicator at the start
	commentIndicator := "  "
	if hasComment {
		commentIndicator = commentMarkerStyle.Render(">>")
	}

	// Truncate content if too long
	content := line.Content
	maxContentLen := m.width - 24
	if len(content) > maxContentLen {
		content = content[:maxContentLen-3] + "..."
	}

	// Build the line
	rendered := commentIndicator + " " + lineNums + " " + contentStyle.Render(prefix+content)

	// Apply selection highlight
	if selected {
		rendered = lineSelectedStyle.Width(m.width - 8).Render(rendered)
	}

	return rendered
}
