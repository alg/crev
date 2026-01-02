package tui

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	colorPrimary   = lipgloss.Color("212") // Bright blue
	colorSecondary = lipgloss.Color("241") // Gray
	colorSuccess   = lipgloss.Color("34")  // Green
	colorWarning   = lipgloss.Color("214") // Orange
	colorDanger    = lipgloss.Color("196") // Red
	colorInfo      = lipgloss.Color("39")  // Cyan

	colorAdded    = lipgloss.Color("34")  // Green
	colorRemoved  = lipgloss.Color("196") // Red
	colorContext  = lipgloss.Color("245") // Light gray
	colorHunk     = lipgloss.Color("39")  // Cyan
	colorLineNum  = lipgloss.Color("240") // Dim gray
	colorSelected = lipgloss.Color("212") // Bright blue
	colorComment  = lipgloss.Color("214") // Orange/Yellow
)

// Severity colors
var severityColors = map[string]lipgloss.Color{
	"suggestion": lipgloss.Color("34"),  // Green
	"question":   lipgloss.Color("39"),  // Cyan
	"concern":    lipgloss.Color("214"), // Orange
	"blocker":    lipgloss.Color("196"), // Red
}

// Styles
var (
	// Base styles
	baseStyle = lipgloss.NewStyle()

	// Title bar
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(colorPrimary).
			Padding(0, 1)

	// Status bar
	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Padding(0, 1)

	statusKeyStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	// File list
	fileListStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSecondary).
			Padding(0, 1)

	fileItemStyle = lipgloss.NewStyle().
			PaddingLeft(1)

	fileItemSelectedStyle = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(lipgloss.Color("15")).
				Bold(true).
				PaddingLeft(1)

	fileStatsStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	fileAddedStyle = lipgloss.NewStyle().
			Foreground(colorAdded)

	fileRemovedStyle = lipgloss.NewStyle().
			Foreground(colorRemoved)

	// Diff view
	diffViewStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSecondary)

	lineNumStyle = lipgloss.NewStyle().
			Foreground(colorLineNum).
			Width(4).
			Align(lipgloss.Right)

	lineAddedStyle = lipgloss.NewStyle().
			Foreground(colorAdded)

	lineRemovedStyle = lipgloss.NewStyle().
			Foreground(colorRemoved)

	lineContextStyle = lipgloss.NewStyle().
			Foreground(colorContext)

	hunkHeaderStyle = lipgloss.NewStyle().
			Foreground(colorHunk).
			Bold(true)

	lineSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("237"))

	// Comment marker shown in diff
	commentMarkerStyle = lipgloss.NewStyle().
				Foreground(colorComment).
				Bold(true)

	// Comment input modal
	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)

	modalTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	severityPillStyle = lipgloss.NewStyle().
				Padding(0, 1).
				MarginRight(1)

	severitySelectedStyle = lipgloss.NewStyle().
				Padding(0, 1).
				MarginRight(1).
				Bold(true).
				Reverse(true)

	// Help
	helpStyle = lipgloss.NewStyle().
			Foreground(colorSecondary)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	// Approval status
	approvedStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	notApprovedStyle = lipgloss.NewStyle().
			Foreground(colorWarning)

	blockerWarningStyle = lipgloss.NewStyle().
				Foreground(colorDanger).
				Bold(true)
)

// SeverityStyle returns the style for a given severity level
func SeverityStyle(severity string) lipgloss.Style {
	color, ok := severityColors[severity]
	if !ok {
		color = colorSecondary
	}
	return lipgloss.NewStyle().Foreground(color)
}

// SeverityPillStyle returns a pill-style badge for a severity
func SeverityPillStyle(severity string, selected bool) lipgloss.Style {
	color, ok := severityColors[severity]
	if !ok {
		color = colorSecondary
	}
	if selected {
		return severitySelectedStyle.Background(color)
	}
	return severityPillStyle.Foreground(color).Border(lipgloss.RoundedBorder()).BorderForeground(color)
}
