package tui

import (
	"fmt"
	"strings"

	"github.com/alg/crev/internal/diff"
	"github.com/alg/crev/internal/review"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Mode represents the current UI mode
type Mode int

const (
	ModeNormal Mode = iota // Normal navigation (sidebar + diff view)
	ModeCommentEdit
	ModeHelp
	ModeSummaryInput
	ModeQuitConfirm
)

// Focus represents which pane has focus
type Focus int

const (
	FocusSidebar Focus = iota
	FocusMain
)

// Model is the main application model
type Model struct {
	diff   *diff.Diff
	review *review.Review
	keys   KeyMap

	// UI state
	mode          Mode
	previousMode  Mode
	focus         Focus
	width         int
	height        int
	ready         bool

	// Sidebar state
	tree         *FileTree
	sidebarWidth int

	// File selection state
	fileIndex int

	// Diff view state
	lineIndex int   // Current line within the flattened diff
	hunkIndex int   // Current hunk index
	scrollOffset int

	// Comment input state
	commentTextarea textarea.Model
	commentSeverity review.Severity
	commentLineStart int
	commentLineEnd   int
	commentSide      string
	editingCommentIndex int // -1 if creating new

	// Summary input state
	summaryTextarea textarea.Model
	submitting      bool
	approved        bool

	// Help state
	showFullHelp bool

	// Output
	outputPath string
	submitted  bool
}

// NewModel creates a new application model
func NewModel(d *diff.Diff, outputPath string) Model {
	ta := textarea.New()
	ta.Placeholder = "Enter your comment..."
	ta.CharLimit = 1000
	ta.SetWidth(60)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.Base = lipgloss.NewStyle()

	sa := textarea.New()
	sa.Placeholder = "Overall review summary (optional)..."
	sa.CharLimit = 500
	sa.SetWidth(60)
	sa.SetHeight(2)
	sa.ShowLineNumbers = false
	sa.Prompt = ""
	sa.FocusedStyle.CursorLine = lipgloss.NewStyle()
	sa.FocusedStyle.Base = lipgloss.NewStyle()
	sa.BlurredStyle.CursorLine = lipgloss.NewStyle()
	sa.BlurredStyle.Base = lipgloss.NewStyle()

	return Model{
		diff:                d,
		review:              review.NewReview(),
		keys:                DefaultKeyMap(),
		mode:                ModeNormal,
		focus:               FocusSidebar,
		tree:                BuildFileTree(d.Files),
		sidebarWidth:        30,
		fileIndex:           0,
		lineIndex:           0,
		commentTextarea:     ta,
		commentSeverity:     review.SeveritySuggestion,
		editingCommentIndex: -1,
		summaryTextarea:     sa,
		outputPath:          outputPath,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle global keys first
		if key.Matches(msg, m.keys.QuitForce) {
			return m, tea.Quit
		}

		// Modal modes take priority
		switch m.mode {
		case ModeCommentEdit:
			return m.updateCommentEdit(msg)
		case ModeSummaryInput:
			return m.updateSummaryInput(msg)
		case ModeHelp:
			return m.updateHelp(msg)
		case ModeQuitConfirm:
			return m.updateQuitConfirm(msg)
		}

		// Focus-based navigation (normal mode)
		switch m.focus {
		case FocusSidebar:
			return m.updateSidebar(msg)
		case FocusMain:
			return m.updateDiffView(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update textarea widths to fill comment pane
		// mainWidth = width - sidebarWidth, pane inner width = mainWidth - 4 (border + padding)
		textareaWidth := m.width - m.sidebarWidth - 4
		if textareaWidth < 20 {
			textareaWidth = 20
		}
		m.commentTextarea.SetWidth(textareaWidth)
		m.summaryTextarea.SetWidth(min(60, m.width-10))
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updateSidebar(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.Cancel):
		m.previousMode = m.mode
		m.mode = ModeQuitConfirm
		return m, nil

	case key.Matches(msg, m.keys.Down):
		m.tree.MoveDown()
		m.selectCurrentTreeNode()

	case key.Matches(msg, m.keys.Up):
		m.tree.MoveUp()
		m.selectCurrentTreeNode()

	case key.Matches(msg, m.keys.Right):
		// Move focus to main area
		m.focus = FocusMain

	case key.Matches(msg, m.keys.ToggleExpand):
		node := m.tree.SelectedNode()
		if node != nil {
			if node.Type == TreeNodeDir {
				// Toggle directory expansion
				m.tree.ToggleExpand()
				// After toggling, select current node if it's a file
				m.selectCurrentTreeNode()
			} else {
				// Select file and move to main
				m.fileIndex = node.FileIndex
				m.focus = FocusMain
				m.lineIndex = 0
				m.hunkIndex = 0
				m.scrollOffset = 0
			}
		}

	case key.Matches(msg, m.keys.Approve):
		m.approved = true
		m.previousMode = m.mode
		m.mode = ModeSummaryInput
		m.summaryTextarea.Focus()
		return m, textarea.Blink

	case key.Matches(msg, m.keys.Submit):
		m.approved = false
		m.previousMode = m.mode
		m.mode = ModeSummaryInput
		m.summaryTextarea.Focus()
		return m, textarea.Blink

	case key.Matches(msg, m.keys.Help):
		m.previousMode = m.mode
		m.mode = ModeHelp
	}

	return m, nil
}

func (m Model) updateDiffView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	file := m.currentFile()
	if file == nil {
		m.focus = FocusSidebar
		return m, nil
	}

	totalLines := m.totalLinesInFile(file)

	switch {
	case key.Matches(msg, m.keys.Cancel), key.Matches(msg, m.keys.Left):
		// Move focus back to sidebar
		m.focus = FocusSidebar

	case key.Matches(msg, m.keys.Quit):
		m.previousMode = m.mode
		m.mode = ModeQuitConfirm
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.lineIndex < totalLines-1 {
			m.lineIndex++
			m.ensureLineVisible()
		}

	case key.Matches(msg, m.keys.Up):
		if m.lineIndex > 0 {
			m.lineIndex--
			m.ensureLineVisible()
		}

	case key.Matches(msg, m.keys.NextHunk):
		m.goToNextHunk()

	case key.Matches(msg, m.keys.PrevHunk):
		m.goToPrevHunk()

	case key.Matches(msg, m.keys.PageDown):
		m.lineIndex = min(m.lineIndex+10, totalLines-1)
		m.ensureLineVisible()

	case key.Matches(msg, m.keys.PageUp):
		m.lineIndex = max(m.lineIndex-10, 0)
		m.ensureLineVisible()

	case key.Matches(msg, m.keys.Home):
		m.lineIndex = 0
		m.scrollOffset = 0

	case key.Matches(msg, m.keys.End):
		m.lineIndex = totalLines - 1
		m.ensureLineVisible()

	case key.Matches(msg, m.keys.Right):
		// Next file
		if m.fileIndex < len(m.diff.Files)-1 {
			m.fileIndex++
			m.lineIndex = 0
			m.hunkIndex = 0
			m.scrollOffset = 0
			// Sync tree selection
			m.tree.SelectFileIndex(m.fileIndex)
		}

	case key.Matches(msg, m.keys.Comment):
		m.startComment()
		return m, textarea.Blink

	case key.Matches(msg, m.keys.DeleteComment):
		m.deleteCommentAtCursor()

	case key.Matches(msg, m.keys.EditComment):
		m.editCommentAtCursor()
		if m.mode == ModeCommentEdit {
			return m, textarea.Blink
		}

	case key.Matches(msg, m.keys.Approve):
		m.approved = true
		m.previousMode = m.mode
		m.mode = ModeSummaryInput
		m.summaryTextarea.Focus()
		return m, textarea.Blink

	case key.Matches(msg, m.keys.Submit):
		m.approved = false
		m.previousMode = m.mode
		m.mode = ModeSummaryInput
		m.summaryTextarea.Focus()
		return m, textarea.Blink

	case key.Matches(msg, m.keys.Help):
		m.previousMode = m.mode
		m.mode = ModeHelp
	}

	return m, nil
}

func (m Model) updateCommentEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		// Esc saves and exits comment editing
		m.saveComment()
		m.mode = ModeNormal
		m.commentTextarea.Reset()
		m.editingCommentIndex = -1
		return m, nil

	case key.Matches(msg, m.keys.NextSeverity):
		m.cycleSeverity(1)
	case key.Matches(msg, m.keys.PrevSeverity):
		m.cycleSeverity(-1)

	default:
		// All other keys (including Enter) go to textarea
		var cmd tea.Cmd
		m.commentTextarea, cmd = m.commentTextarea.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) updateSummaryInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.mode = m.previousMode
		m.summaryTextarea.Reset()
		return m, nil

	case key.Matches(msg, m.keys.Confirm):
		m.review.Summary = strings.TrimSpace(m.summaryTextarea.Value())
		m.review.Approved = m.approved
		m.submitted = true
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		m.summaryTextarea, cmd = m.summaryTextarea.Update(msg)
		return m, cmd
	}
}

func (m Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel), key.Matches(msg, m.keys.Help), key.Matches(msg, m.keys.Quit):
		m.mode = m.previousMode
	}
	return m, nil
}

func (m Model) updateQuitConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m, tea.Quit
	case "n", "N", "esc":
		m.mode = m.previousMode
	}
	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	switch m.mode {
	case ModeHelp:
		return m.viewHelp()
	case ModeSummaryInput:
		return m.viewWithModal(m.viewSummaryModal())
	case ModeQuitConfirm:
		return m.viewWithModal(m.viewQuitConfirmModal())
	case ModeCommentEdit:
		return m.viewMain() // Comment edit is inline in diff view
	default:
		return m.viewMain()
	}
}

func (m Model) viewMain() string {
	// Title bar
	title := titleStyle.Width(m.width).Render(fmt.Sprintf(" crev - Code Review (%d files, %d comments)",
		len(m.diff.Files), m.review.CommentCount()))

	// Status bar
	statusBar := m.viewStatusBar()

	// Calculate heights
	commentPaneHeight := 6 // Height for comment pane
	totalContentHeight := m.height - 3 // title + status bar + margins
	diffContentHeight := totalContentHeight - commentPaneHeight

	// Main content width (sidebar width includes its border)
	mainWidth := m.width - m.sidebarWidth

	// Sidebar stretches full height
	sidebar := m.viewSidebar(totalContentHeight)

	// Main content (diff view + comment pane)
	diffContent := m.viewDiffViewWithWidth(diffContentHeight, mainWidth)
	commentPane := m.viewCommentPaneWithWidth(mainWidth)
	mainContent := lipgloss.JoinVertical(lipgloss.Left, diffContent, commentPane)

	// Horizontal layout: sidebar | main
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainContent)

	return lipgloss.JoinVertical(lipgloss.Left, title, body, statusBar)
}

func (m Model) viewCommentPaneWithWidth(width int) string {
	// Get current line info
	lineInfo := m.getCurrentLineInfo()

	var header string
	var body string

	if lineInfo == nil {
		header = helpStyle.Render("No line selected")
		body = ""
	} else {
		file := m.currentFile()
		fileName := ""
		if file != nil {
			fileName = file.Path
		}

		// Check if there's a comment on this line
		var existingComment *review.Comment
		for i := range m.review.Comments {
			c := &m.review.Comments[i]
			if c.File == fileName && c.LineStart == lineInfo.lineNum && c.Side == lineInfo.side {
				existingComment = c
				break
			}
		}

		if m.mode == ModeCommentEdit {
			// Editing mode - show textarea
			severityLabel := SeverityStyle(string(m.commentSeverity)).Render(
				fmt.Sprintf("[%s]", m.commentSeverity.Label()))
			header = fmt.Sprintf("Line %d %s  (Tab: severity, Esc: save)",
				lineInfo.lineNum, severityLabel)
			body = m.commentTextarea.View()
		} else if existingComment != nil {
			// Show existing comment
			severityLabel := SeverityStyle(string(existingComment.Severity)).Render(
				fmt.Sprintf("[%s]", existingComment.Severity.Label()))
			header = fmt.Sprintf("Line %d %s  (e: edit, d: delete)", lineInfo.lineNum, severityLabel)
			body = existingComment.Text
		} else {
			// No comment on this line
			header = fmt.Sprintf("Line %d  (i: add comment)", lineInfo.lineNum)
			body = helpStyle.Render("No comment on this line")
		}
	}

	headerStyled := lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(header)

	pane := lipgloss.JoinVertical(lipgloss.Left, headerStyled, body)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSecondary).
		Padding(0, 1).
		Width(width - 2).
		Height(4).
		Render(pane)
}

func (m Model) viewStatusBar() string {
	var status string
	var help string

	// Show current file info
	file := m.currentFile()
	if file != nil {
		add, del := file.Stats()
		status = fmt.Sprintf(" %s | +%d -%d | ", file.Path, add, del)
	} else {
		status = fmt.Sprintf(" %d files | ", len(m.diff.Files))
	}

	// Show focus-specific help
	if m.mode == ModeCommentEdit {
		status = " EDITING COMMENT | "
		help = helpKeyStyle.Render("Tab") + helpStyle.Render(" severity  ")
		help += helpKeyStyle.Render("Esc") + helpStyle.Render(" save & exit")
	} else if m.focus == FocusSidebar {
		help = helpKeyStyle.Render("l/Enter") + helpStyle.Render(" select  ")
		help += helpKeyStyle.Render("?") + helpStyle.Render(" help  ")
		help += helpKeyStyle.Render("a") + helpStyle.Render(" approve  ")
		help += helpKeyStyle.Render("q") + helpStyle.Render(" quit")
	} else {
		help = helpKeyStyle.Render("h") + helpStyle.Render(" sidebar  ")
		help += helpKeyStyle.Render("i") + helpStyle.Render(" comment  ")
		help += helpKeyStyle.Render("?") + helpStyle.Render(" help  ")
		help += helpKeyStyle.Render("a") + helpStyle.Render(" approve")
	}

	return statusBarStyle.Width(m.width).Render(status + help)
}

func (m Model) viewWithModal(modal string) string {
	// Use lipgloss.Place to center the modal on a full-screen background
	// This properly handles ANSI escape codes unlike character-by-character overlay
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}

func (m Model) viewSummaryModal() string {
	var title string
	if m.approved {
		title = approvedStyle.Render("Approve & Submit Review")
	} else {
		title = modalTitleStyle.Render("Submit Review")
	}

	stats := fmt.Sprintf("%d comments", m.review.CommentCount())
	if m.review.HasBlockers() {
		stats += blockerWarningStyle.Render(" (has blockers!)")
	}

	textarea := m.summaryTextarea.View()
	hint := helpStyle.Render("Enter to submit, Esc to cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, stats, "", textarea, "", hint)
	return modalStyle.Render(content)
}

func (m Model) viewQuitConfirmModal() string {
	title := modalTitleStyle.Render("Quit without submitting?")
	body := fmt.Sprintf("You have %d comments that will be lost.", m.review.CommentCount())
	hint := helpKeyStyle.Render("y") + helpStyle.Render(" quit  ") +
		helpKeyStyle.Render("n/Esc") + helpStyle.Render(" cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", body, "", hint)
	return modalStyle.Render(content)
}

func (m Model) viewHelp() string {
	title := titleStyle.Width(m.width).Render(" crev - Help")

	help := `
Navigation:
  h/l             Switch between sidebar and diff view
  j/k, up/down    Navigate in current pane
  J/K             Navigate hunks (in diff view)
  ctrl+d/u        Page down/up
  g/G             Go to top/bottom

Sidebar (Tree):
  Enter/Space     Expand/collapse directory or select file

Diff View:
  i               Add comment on current line
  e               Edit comment on current line
  d               Delete comment on current line

Editing Comment:
  Tab             Cycle severity
  Enter           New line
  Esc             Save and exit

Submit:
  a               Approve and submit
  s               Submit without approval
  q               Quit without submitting

Press Esc or ? to close this help.
`
	return lipgloss.JoinVertical(lipgloss.Left, title, help)
}

// Helper methods

func (m *Model) currentFile() *diff.File {
	if m.fileIndex >= 0 && m.fileIndex < len(m.diff.Files) {
		return &m.diff.Files[m.fileIndex]
	}
	return nil
}

// selectCurrentTreeNode updates fileIndex when navigating to a file in the tree
func (m *Model) selectCurrentTreeNode() {
	node := m.tree.SelectedNode()
	if node != nil && node.Type == TreeNodeFile && node.FileIndex >= 0 {
		if m.fileIndex != node.FileIndex {
			m.fileIndex = node.FileIndex
			m.lineIndex = 0
			m.hunkIndex = 0
			m.scrollOffset = 0
		}
	}
}

func (m *Model) totalLinesInFile(file *diff.File) int {
	total := 0
	for _, hunk := range file.Hunks {
		total++ // hunk header
		total += len(hunk.Lines)
	}
	return max(total, 1)
}

func (m *Model) ensureLineVisible() {
	// Calculate visible height: total height - title - status - comment pane - borders
	// Comment pane is always visible now
	commentPaneHeight := 8 // comment pane + borders
	viewHeight := m.height - 5 - commentPaneHeight

	if viewHeight < 1 {
		viewHeight = 1
	}

	if m.lineIndex < m.scrollOffset {
		m.scrollOffset = m.lineIndex
	} else if m.lineIndex >= m.scrollOffset+viewHeight {
		m.scrollOffset = m.lineIndex - viewHeight + 1
	}
}

func (m *Model) goToNextHunk() {
	file := m.currentFile()
	if file == nil {
		return
	}

	lineCount := 0
	for i, hunk := range file.Hunks {
		if i > m.hunkIndex {
			m.lineIndex = lineCount
			m.hunkIndex = i
			m.ensureLineVisible()
			return
		}
		lineCount++ // hunk header
		lineCount += len(hunk.Lines)
	}
}

func (m *Model) goToPrevHunk() {
	file := m.currentFile()
	if file == nil {
		return
	}

	if m.hunkIndex > 0 {
		m.hunkIndex--
		lineCount := 0
		for i := 0; i < m.hunkIndex; i++ {
			lineCount++
			lineCount += len(file.Hunks[i].Lines)
		}
		m.lineIndex = lineCount
		m.ensureLineVisible()
	}
}

func (m *Model) startComment() {
	file := m.currentFile()
	if file == nil {
		return
	}

	// Find the current line info
	lineInfo := m.getCurrentLineInfo()
	if lineInfo == nil {
		return
	}

	m.commentLineStart = lineInfo.lineNum
	m.commentLineEnd = lineInfo.lineNum
	m.commentSide = lineInfo.side
	m.commentSeverity = review.SeveritySuggestion
	m.commentTextarea.Reset()
	m.commentTextarea.Focus()
	m.editingCommentIndex = -1
	m.mode = ModeCommentEdit
}

type lineInfo struct {
	lineNum int
	side    string // "new" or "old"
}

func (m *Model) getCurrentLineInfo() *lineInfo {
	file := m.currentFile()
	if file == nil {
		return nil
	}

	currentLine := 0
	for _, hunk := range file.Hunks {
		if m.lineIndex == currentLine {
			// On hunk header - use first line of hunk
			if len(hunk.Lines) > 0 {
				line := hunk.Lines[0]
				if line.NewNum > 0 {
					return &lineInfo{lineNum: line.NewNum, side: "new"}
				}
				return &lineInfo{lineNum: line.OldNum, side: "old"}
			}
			return nil
		}
		currentLine++

		for _, line := range hunk.Lines {
			if m.lineIndex == currentLine {
				switch line.Type {
				case diff.LineAdded:
					return &lineInfo{lineNum: line.NewNum, side: "new"}
				case diff.LineRemoved:
					return &lineInfo{lineNum: line.OldNum, side: "old"}
				default:
					if line.NewNum > 0 {
						return &lineInfo{lineNum: line.NewNum, side: "new"}
					}
					return &lineInfo{lineNum: line.OldNum, side: "old"}
				}
			}
			currentLine++
		}
	}
	return nil
}

func (m *Model) saveComment() {
	file := m.currentFile()
	if file == nil {
		return
	}

	text := strings.TrimSpace(m.commentTextarea.Value())
	if text == "" {
		return
	}

	comment := review.Comment{
		File:      file.Path,
		LineStart: m.commentLineStart,
		LineEnd:   m.commentLineEnd,
		Side:      m.commentSide,
		Severity:  m.commentSeverity,
		Text:      text,
	}

	if m.editingCommentIndex >= 0 {
		// Replace existing comment
		m.review.Comments[m.editingCommentIndex] = comment
	} else {
		m.review.AddComment(comment)
	}
}

func (m *Model) deleteCommentAtCursor() {
	file := m.currentFile()
	if file == nil {
		return
	}

	lineInfo := m.getCurrentLineInfo()
	if lineInfo == nil {
		return
	}

	// Find and delete comment at this location
	for i, c := range m.review.Comments {
		if c.File == file.Path && c.LineStart == lineInfo.lineNum && c.Side == lineInfo.side {
			m.review.RemoveComment(i)
			return
		}
	}
}

func (m *Model) editCommentAtCursor() {
	file := m.currentFile()
	if file == nil {
		return
	}

	lineInfo := m.getCurrentLineInfo()
	if lineInfo == nil {
		return
	}

	// Find comment at this location
	for i, c := range m.review.Comments {
		if c.File == file.Path && c.LineStart == lineInfo.lineNum && c.Side == lineInfo.side {
			m.commentLineStart = c.LineStart
			m.commentLineEnd = c.LineEnd
			m.commentSide = c.Side
			m.commentSeverity = c.Severity
			m.commentTextarea.SetValue(c.Text)
			m.commentTextarea.Focus()
			m.editingCommentIndex = i
			m.mode = ModeCommentEdit
			return
		}
	}
}

func (m *Model) cycleSeverity(dir int) {
	severities := review.AllSeverities()
	for i, s := range severities {
		if s == m.commentSeverity {
			newIndex := (i + dir + len(severities)) % len(severities)
			m.commentSeverity = severities[newIndex]
			return
		}
	}
}

// GetReview returns the review for output
func (m *Model) GetReview() *review.Review {
	return m.review
}

// WasSubmitted returns true if the review was submitted
func (m Model) WasSubmitted() bool {
	return m.submitted
}

