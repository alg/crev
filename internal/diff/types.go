package diff

// LineType represents the type of a diff line
type LineType int

const (
	LineContext LineType = iota
	LineAdded
	LineRemoved
)

// Line represents a single line in a diff hunk
type Line struct {
	Type    LineType `json:"type"`
	Content string   `json:"content"`
	OldNum  int      `json:"old_num"` // Line number in old file (0 if not applicable)
	NewNum  int      `json:"new_num"` // Line number in new file (0 if not applicable)
}

// Hunk represents a section of changes in a file
type Hunk struct {
	OldStart int    `json:"old_start"`
	OldCount int    `json:"old_count"`
	NewStart int    `json:"new_start"`
	NewCount int    `json:"new_count"`
	Header   string `json:"header"` // The @@ line
	Lines    []Line `json:"lines"`
}

// Stats returns the number of additions and deletions in the hunk
func (h *Hunk) Stats() (additions, deletions int) {
	for _, line := range h.Lines {
		switch line.Type {
		case LineAdded:
			additions++
		case LineRemoved:
			deletions++
		}
	}
	return
}

// File represents a single file in a diff
type File struct {
	Path        string `json:"path"`
	OldPath     string `json:"old_path,omitempty"` // Different from Path for renames
	Hunks       []Hunk `json:"hunks"`
	IsNew       bool   `json:"is_new,omitempty"`
	IsDeleted   bool   `json:"is_deleted,omitempty"`
	IsBinary    bool   `json:"is_binary,omitempty"`
	IsUntracked bool   `json:"is_untracked,omitempty"` // File is not tracked by git
}

// Stats returns the total additions and deletions for the file
func (f *File) Stats() (additions, deletions int) {
	for _, hunk := range f.Hunks {
		a, d := hunk.Stats()
		additions += a
		deletions += d
	}
	return
}

// Diff represents a complete git diff output
type Diff struct {
	Files []File `json:"files"`
}

// Stats returns the total additions and deletions across all files
func (d *Diff) Stats() (additions, deletions int) {
	for _, file := range d.Files {
		a, del := file.Stats()
		additions += a
		deletions += del
	}
	return
}

// FileCount returns the number of files in the diff
func (d *Diff) FileCount() int {
	return len(d.Files)
}
