package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// viewSidebar renders the directory tree sidebar
func (m Model) viewSidebar(height int) string {
	if m.tree == nil || len(m.tree.FlatList) == 0 {
		return m.renderSidebarBox("No files", height)
	}

	var lines []string

	// Calculate visible range with scrolling
	visibleLines := height - 2 // Account for border
	if visibleLines < 1 {
		visibleLines = 1
	}

	startIdx := 0
	if m.tree.SelectedIndex >= visibleLines {
		startIdx = m.tree.SelectedIndex - visibleLines + 1
	}
	endIdx := startIdx + visibleLines
	if endIdx > len(m.tree.FlatList) {
		endIdx = len(m.tree.FlatList)
	}

	for i := startIdx; i < endIdx; i++ {
		node := m.tree.FlatList[i]
		selected := i == m.tree.SelectedIndex && m.focus == FocusSidebar
		line := m.renderTreeNode(node, selected)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	return m.renderSidebarBox(content, height)
}

// renderSidebarBox wraps content in a styled box
func (m Model) renderSidebarBox(content string, height int) string {
	borderColor := colorSecondary
	if m.focus == FocusSidebar {
		borderColor = colorPrimary
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(m.sidebarWidth - 2).
		Height(height).
		Render(content)
}

// renderTreeNode renders a single tree node
func (m Model) renderTreeNode(node *TreeNode, selected bool) string {
	innerWidth := m.sidebarWidth - 4 // Account for border (2) and padding (2)
	if innerWidth < 10 {
		innerWidth = 10
	}

	// Indentation based on depth
	indent := strings.Repeat("  ", node.Depth)

	// Icon (plain text, no styling yet)
	var icon string
	if node.Type == TreeNodeDir {
		if node.Expanded {
			icon = "\uF07C" // nf-fa-folder_open
		} else {
			icon = "\uF07B" // nf-fa-folder
		}
	} else {
		// Plain status letter
		if node.FileIndex >= 0 && node.FileIndex < len(m.diff.Files) {
			file := m.diff.Files[node.FileIndex]
			switch {
			case file.IsNew:
				icon = "A"
			case file.IsDeleted:
				icon = "D"
			case file.IsBinary:
				icon = "B"
			default:
				icon = "M"
			}
		} else {
			icon = " "
		}
	}

	// Comment count for files (plain text)
	commentBadge := ""
	if node.Type == TreeNodeFile && node.FileIndex >= 0 && node.FileIndex < len(m.diff.Files) {
		file := m.diff.Files[node.FileIndex]
		count := len(m.review.CommentsForFile(file.Path))
		if count > 0 {
			commentBadge = fmt.Sprintf(" [%d]", count)
		}
	}

	// Calculate max name length
	prefixLen := len(indent) + 2 // indent + icon + space
	suffixLen := len(commentBadge)
	maxNameLen := innerWidth - prefixLen - suffixLen
	if maxNameLen < 4 {
		maxNameLen = 4
	}

	// Truncate name if needed
	name := node.Name
	if len(name) > maxNameLen {
		name = name[:maxNameLen-2] + ".."
	}

	// Build plain text line and pad to full width
	line := fmt.Sprintf("%s%s %s%s", indent, icon, name, commentBadge)
	if len(line) < innerWidth {
		line += strings.Repeat(" ", innerWidth-len(line))
	}

	// Apply style to entire line
	if selected {
		return lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Width(innerWidth).
			Render(line)
	} else if node.Type == TreeNodeDir {
		return lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			Width(innerWidth).
			Render(line)
	}

	return lipgloss.NewStyle().Width(innerWidth).Render(line)
}

// getFileStatusIcon returns the status icon for a file
func (m Model) getFileStatusIcon(fileIndex int) string {
	if fileIndex < 0 || fileIndex >= len(m.diff.Files) {
		return " "
	}
	file := m.diff.Files[fileIndex]

	switch {
	case file.IsNew:
		return fileAddedStyle.Render("A")
	case file.IsDeleted:
		return fileRemovedStyle.Render("D")
	case file.IsBinary:
		return fileStatsStyle.Render("B")
	default:
		return fileStatsStyle.Render("M")
	}
}
