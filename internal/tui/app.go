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
	ModeFileList Mode = iota
	ModeDiffView
	ModeCommentInput
	ModeHelp
	ModeSummaryInput
)

// Model is the main application model
type Model struct {
	diff   *diff.Diff
	review *review.Review
	keys   KeyMap

	// UI state
	mode          Mode
	previousMode  Mode
	width         int
	height        int
	ready         bool

	// File list state
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

	sa := textarea.New()
	sa.Placeholder = "Overall review summary (optional)..."
	sa.CharLimit = 500
	sa.SetWidth(60)
	sa.SetHeight(2)

	return Model{
		diff:                d,
		review:              review.NewReview(),
		keys:                DefaultKeyMap(),
		mode:                ModeFileList,
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

		switch m.mode {
		case ModeFileList:
			return m.updateFileList(msg)
		case ModeDiffView:
			return m.updateDiffView(msg)
		case ModeCommentInput:
			return m.updateCommentInput(msg)
		case ModeSummaryInput:
			return m.updateSummaryInput(msg)
		case ModeHelp:
			return m.updateHelp(msg)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update textarea widths
		m.commentTextarea.SetWidth(min(60, m.width-10))
		m.summaryTextarea.SetWidth(min(60, m.width-10))
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updateFileList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Down):
		if m.fileIndex < len(m.diff.Files)-1 {
			m.fileIndex++
		}

	case key.Matches(msg, m.keys.Up):
		if m.fileIndex > 0 {
			m.fileIndex--
		}

	case key.Matches(msg, m.keys.Right), key.Matches(msg, m.keys.Confirm):
		if len(m.diff.Files) > 0 {
			m.mode = ModeDiffView
			m.lineIndex = 0
			m.hunkIndex = 0
			m.scrollOffset = 0
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
		m.mode = ModeFileList
		return m, nil
	}

	totalLines := m.totalLinesInFile(file)

	switch {
	case key.Matches(msg, m.keys.Quit), key.Matches(msg, m.keys.Left):
		m.mode = ModeFileList

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
		if m.fileIndex < len(m.diff.Files)-1 {
			m.fileIndex++
			m.lineIndex = 0
			m.hunkIndex = 0
			m.scrollOffset = 0
		}

	case key.Matches(msg, m.keys.Comment):
		m.startComment()
		return m, textarea.Blink

	case key.Matches(msg, m.keys.DeleteComment):
		m.deleteCommentAtCursor()

	case key.Matches(msg, m.keys.EditComment):
		m.editCommentAtCursor()
		if m.mode == ModeCommentInput {
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

func (m Model) updateCommentInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.mode = ModeDiffView
		m.commentTextarea.Reset()
		m.editingCommentIndex = -1
		return m, nil

	case key.Matches(msg, m.keys.Confirm):
		m.saveComment()
		m.mode = ModeDiffView
		m.commentTextarea.Reset()
		m.editingCommentIndex = -1
		return m, nil

	case key.Matches(msg, m.keys.Severity1):
		m.commentSeverity = review.SeveritySuggestion
	case key.Matches(msg, m.keys.Severity2):
		m.commentSeverity = review.SeverityQuestion
	case key.Matches(msg, m.keys.Severity3):
		m.commentSeverity = review.SeverityConcern
	case key.Matches(msg, m.keys.Severity4):
		m.commentSeverity = review.SeverityBlocker

	case key.Matches(msg, m.keys.NextSeverity):
		m.cycleSeverity(1)
	case key.Matches(msg, m.keys.PrevSeverity):
		m.cycleSeverity(-1)

	default:
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

// View renders the UI
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	switch m.mode {
	case ModeHelp:
		return m.viewHelp()
	case ModeCommentInput:
		return m.viewWithModal(m.viewCommentModal())
	case ModeSummaryInput:
		return m.viewWithModal(m.viewSummaryModal())
	default:
		return m.viewMain()
	}
}

func (m Model) viewMain() string {
	// Title bar
	title := titleStyle.Width(m.width).Render(fmt.Sprintf(" crev - Code Review (%d files, %d comments)",
		len(m.diff.Files), m.review.CommentCount()))

	// Main content
	var content string
	contentHeight := m.height - 4 // title + status bar + margins

	switch m.mode {
	case ModeFileList:
		content = m.viewFileList(contentHeight)
	case ModeDiffView:
		content = m.viewDiffView(contentHeight)
	}

	// Status bar
	statusBar := m.viewStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, title, content, statusBar)
}

func (m Model) viewStatusBar() string {
	var status string

	switch m.mode {
	case ModeFileList:
		status = fmt.Sprintf(" %d/%d files | ", m.fileIndex+1, len(m.diff.Files))
	case ModeDiffView:
		file := m.currentFile()
		if file != nil {
			add, del := file.Stats()
			status = fmt.Sprintf(" %s | +%d -%d | ", file.Path, add, del)
		}
	}

	help := helpKeyStyle.Render("?") + helpStyle.Render(" help  ")
	help += helpKeyStyle.Render("c") + helpStyle.Render(" comment  ")
	help += helpKeyStyle.Render("a") + helpStyle.Render(" approve  ")
	help += helpKeyStyle.Render("s") + helpStyle.Render(" submit  ")
	help += helpKeyStyle.Render("q") + helpStyle.Render(" quit")

	return statusBarStyle.Width(m.width).Render(status + help)
}

func (m Model) viewWithModal(modal string) string {
	base := m.viewMain()
	// Center the modal
	modalWidth := lipgloss.Width(modal)
	modalHeight := lipgloss.Height(modal)

	x := (m.width - modalWidth) / 2
	y := (m.height - modalHeight) / 2

	return placeOverlay(x, y, modal, base)
}

func (m Model) viewCommentModal() string {
	severities := review.AllSeverities()
	var pills []string
	for _, s := range severities {
		pill := SeverityPillStyle(string(s), s == m.commentSeverity).Render(
			fmt.Sprintf("%s %s", s.Shortcut(), s.Label()))
		pills = append(pills, pill)
	}

	title := modalTitleStyle.Render("Add Comment")
	severityRow := lipgloss.JoinHorizontal(lipgloss.Center, pills...)
	textarea := m.commentTextarea.View()
	hint := helpStyle.Render("Enter to save, Esc to cancel, 1-4 to set severity")

	content := lipgloss.JoinVertical(lipgloss.Left, title, severityRow, "", textarea, "", hint)
	return modalStyle.Render(content)
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

func (m Model) viewHelp() string {
	title := titleStyle.Width(m.width).Render(" crev - Help")

	help := `
Navigation:
  j/k, up/down    Navigate lines
  J/K             Navigate hunks
  h/l, left/right Navigate files
  ctrl+d/u        Page down/up
  g/G             Go to top/bottom

Actions:
  c               Add comment on current line
  d               Delete comment on current line
  e               Edit comment on current line

Comment Modal:
  1-4             Set severity (suggestion/question/concern/blocker)
  Tab             Cycle severity
  Enter           Save comment
  Esc             Cancel

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

func (m *Model) totalLinesInFile(file *diff.File) int {
	total := 0
	for _, hunk := range file.Hunks {
		total++ // hunk header
		total += len(hunk.Lines)
	}
	return max(total, 1)
}

func (m *Model) ensureLineVisible() {
	viewHeight := m.height - 6
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
	m.mode = ModeCommentInput
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
			m.mode = ModeCommentInput
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

// Helper function to place an overlay on top of base content
func placeOverlay(x, y int, overlay, base string) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	for i, overlayLine := range overlayLines {
		baseLineIdx := y + i
		if baseLineIdx >= 0 && baseLineIdx < len(baseLines) {
			baseLine := baseLines[baseLineIdx]
			baseRunes := []rune(baseLine)

			// Pad base line if needed
			for len(baseRunes) < x+len([]rune(overlayLine)) {
				baseRunes = append(baseRunes, ' ')
			}

			// Replace section with overlay
			overlayRunes := []rune(overlayLine)
			for j, r := range overlayRunes {
				if x+j < len(baseRunes) {
					baseRunes[x+j] = r
				}
			}
			baseLines[baseLineIdx] = string(baseRunes)
		}
	}

	return strings.Join(baseLines, "\n")
}
