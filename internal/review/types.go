package review

import "time"

// Severity represents the importance level of a review comment
type Severity string

const (
	SeveritySuggestion Severity = "suggestion"
	SeverityQuestion   Severity = "question"
	SeverityConcern    Severity = "concern"
	SeverityBlocker    Severity = "blocker"
)

// AllSeverities returns all available severity levels in order
func AllSeverities() []Severity {
	return []Severity{
		SeveritySuggestion,
		SeverityQuestion,
		SeverityConcern,
		SeverityBlocker,
	}
}

// SeverityLabel returns a human-readable label for the severity
func (s Severity) Label() string {
	switch s {
	case SeveritySuggestion:
		return "Suggestion"
	case SeverityQuestion:
		return "Question"
	case SeverityConcern:
		return "Concern"
	case SeverityBlocker:
		return "Blocker"
	default:
		return string(s)
	}
}

// SeverityShortcut returns the keyboard shortcut for the severity
func (s Severity) Shortcut() string {
	switch s {
	case SeveritySuggestion:
		return "1"
	case SeverityQuestion:
		return "2"
	case SeverityConcern:
		return "3"
	case SeverityBlocker:
		return "4"
	default:
		return ""
	}
}

// Comment represents a single review comment
type Comment struct {
	File      string   `json:"file"`
	LineStart int      `json:"line_start"`
	LineEnd   int      `json:"line_end"`
	Side      string   `json:"side"` // "new" or "old"
	Severity  Severity `json:"severity"`
	Text      string   `json:"text"`
}

// Review represents a complete code review
type Review struct {
	Timestamp  time.Time `json:"timestamp"`
	BaseCommit string    `json:"base_commit,omitempty"`
	Comments   []Comment `json:"comments"`
	Summary    string    `json:"summary,omitempty"`
	Approved   bool      `json:"approved"`
}

// NewReview creates a new review with the current timestamp
func NewReview() *Review {
	return &Review{
		Timestamp: time.Now().UTC(),
		Comments:  []Comment{},
	}
}

// AddComment adds a comment to the review
func (r *Review) AddComment(c Comment) {
	r.Comments = append(r.Comments, c)
}

// RemoveComment removes a comment at the given index
func (r *Review) RemoveComment(index int) {
	if index >= 0 && index < len(r.Comments) {
		r.Comments = append(r.Comments[:index], r.Comments[index+1:]...)
	}
}

// CommentsForFile returns all comments for a specific file
func (r *Review) CommentsForFile(file string) []Comment {
	var result []Comment
	for _, c := range r.Comments {
		if c.File == file {
			result = append(result, c)
		}
	}
	return result
}

// CommentCount returns the total number of comments
func (r *Review) CommentCount() int {
	return len(r.Comments)
}

// HasBlockers returns true if any comment is a blocker
func (r *Review) HasBlockers() bool {
	for _, c := range r.Comments {
		if c.Severity == SeverityBlocker {
			return true
		}
	}
	return false
}
