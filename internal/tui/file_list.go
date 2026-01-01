package tui

import (
	"fmt"
	"strings"

	"github.com/alg/crev/internal/diff"
	"github.com/charmbracelet/lipgloss"
)

// viewFileList renders the file list view
func (m Model) viewFileList(height int) string {
	if len(m.diff.Files) == 0 {
		return lipgloss.NewStyle().
			Width(m.width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(colorSecondary).
			Render("No changes to review")
	}

	var lines []string

	// Calculate visible range
	visibleLines := height - 2 // Account for border
	startIdx := 0
	if m.fileIndex >= visibleLines {
		startIdx = m.fileIndex - visibleLines + 1
	}
	endIdx := min(startIdx+visibleLines, len(m.diff.Files))

	for i := startIdx; i < endIdx; i++ {
		file := m.diff.Files[i]
		line := m.renderFileItem(file, i == m.fileIndex)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")

	// Add scroll indicator if needed
	if len(m.diff.Files) > visibleLines {
		scrollInfo := fmt.Sprintf(" %d-%d of %d ", startIdx+1, endIdx, len(m.diff.Files))
		content += "\n" + helpStyle.Render(scrollInfo)
	}

	return fileListStyle.
		Width(m.width - 4).
		Height(height).
		Render(content)
}

func (m Model) renderFileItem(file diff.File, selected bool) string {
	// Icon based on file status
	var icon string
	switch {
	case file.IsNew:
		icon = fileAddedStyle.Render("A")
	case file.IsDeleted:
		icon = fileRemovedStyle.Render("D")
	case file.IsBinary:
		icon = fileStatsStyle.Render("B")
	default:
		icon = fileStatsStyle.Render("M")
	}

	// File path
	path := file.Path

	// Stats
	add, del := file.Stats()
	stats := ""
	if add > 0 {
		stats += fileAddedStyle.Render(fmt.Sprintf("+%d", add))
	}
	if del > 0 {
		if stats != "" {
			stats += " "
		}
		stats += fileRemovedStyle.Render(fmt.Sprintf("-%d", del))
	}

	// Comment count for this file
	commentCount := len(m.review.CommentsForFile(file.Path))
	commentBadge := ""
	if commentCount > 0 {
		commentBadge = commentMarkerStyle.Render(fmt.Sprintf(" [%d]", commentCount))
	}

	// Construct the line
	line := fmt.Sprintf("%s %s %s%s", icon, path, stats, commentBadge)

	// Apply selection style
	style := fileItemStyle
	if selected {
		style = fileItemSelectedStyle
	}

	return style.Width(m.width - 8).Render(line)
}
