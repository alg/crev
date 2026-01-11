package tui

import (
	"fmt"
	"strings"

	"github.com/alg/crev/internal/diff"
	"github.com/charmbracelet/lipgloss"
)

// viewDiffViewWithWidth renders the diff view for the current file with specified width
func (m Model) viewDiffViewWithWidth(height int, width int) string {
	file := m.currentFile()

	// Determine border color based on focus
	borderColor := colorSecondary
	if m.focus == FocusMain {
		borderColor = colorPrimary
	}

	if file == nil {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Width(width - 2).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(colorSecondary).
			Render("No file selected")
	}

	if file.IsBinary {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Width(width - 2).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(colorSecondary).
			Render("Binary file - cannot display diff")
	}

	if len(file.Hunks) == 0 {
		message := "No changes in this file"
		if file.IsNew || file.IsUntracked {
			message = "File is empty"
		}
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Width(width - 2).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(colorSecondary).
			Render(message)
	}

	var lines []string
	lineNum := 0
	viewHeight := height - 2 // Account for border

	// Build all lines first
	for hunkIdx, hunk := range file.Hunks {
		// Hunk header
		header := m.renderHunkHeaderWithWidth(hunk, lineNum == m.lineIndex, width)
		if lineNum >= m.scrollOffset && lineNum < m.scrollOffset+viewHeight {
			lines = append(lines, header)
		}
		lineNum++

		// Hunk lines
		for _, line := range hunk.Lines {
			rendered := m.renderDiffLineWithWidth(file, hunkIdx, line, lineNum == m.lineIndex, width)
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

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(width - 2).
		Height(height).
		Render(content)
}

func (m Model) renderHunkHeaderWithWidth(hunk diff.Hunk, selected bool, width int) string {
	// Truncate header if too long
	header := hunk.Header
	maxLen := width - 10
	if maxLen < 20 {
		maxLen = 20
	}
	if len(header) > maxLen {
		header = header[:maxLen-3] + "..."
	}

	lineWidth := width - 2
	style := lipgloss.NewStyle().
		Foreground(colorHunk).
		Bold(true).
		Width(lineWidth)

	if selected {
		style = style.Background(lipgloss.Color("236"))
	}

	return style.Render(header)
}

func (m Model) renderDiffLineWithWidth(file *diff.File, hunkIdx int, line diff.Line, selected bool, width int) string {
	// Line numbers (plain text)
	oldNum := "    "
	newNum := "    "
	if line.OldNum > 0 {
		oldNum = fmt.Sprintf("%4d", line.OldNum)
	}
	if line.NewNum > 0 {
		newNum = fmt.Sprintf("%4d", line.NewNum)
	}

	// Line prefix
	var prefix string
	switch line.Type {
	case diff.LineAdded:
		prefix = "+"
	case diff.LineRemoved:
		prefix = "-"
	default:
		prefix = " "
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

	// Comment indicator
	commentIndicator := "  "
	if hasComment {
		commentIndicator = ">>"
	}

	// Replace tabs with spaces for consistent width
	content := strings.ReplaceAll(line.Content, "\t", "    ")

	// Truncate content if too long
	maxContentLen := width - 20
	if maxContentLen < 10 {
		maxContentLen = 10
	}
	if len(content) > maxContentLen {
		content = content[:maxContentLen-3] + "..."
	}

	// Build plain text line
	plainLine := fmt.Sprintf("%s %s %s %s%s", commentIndicator, oldNum, newNum, prefix, content)

	// Determine text color based on line type
	var fg lipgloss.Color
	switch line.Type {
	case diff.LineAdded:
		fg = colorAdded
	case diff.LineRemoved:
		fg = colorRemoved
	default:
		fg = colorContext
	}

	// Build style with fixed width - this ensures full-line background
	lineWidth := width - 2
	style := lipgloss.NewStyle().
		Foreground(fg).
		Width(lineWidth)

	if hasComment {
		style = style.Bold(true)
	}
	if selected {
		style = style.Background(lipgloss.Color("236"))
	}

	return style.Render(plainLine)
}
