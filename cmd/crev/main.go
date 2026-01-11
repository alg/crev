package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/alg/crev/internal/diff"
	"github.com/alg/crev/internal/review"
	"github.com/alg/crev/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	outputFile = flag.String("o", "", "Output file for review JSON (default: stdout)")
	directory  = flag.String("d", "", "Directory to run git diff in (default: current directory)")
	staged     = flag.Bool("staged", false, "Review staged changes (git diff --cached)")
	jsonRaw    = flag.Bool("json", false, "Output raw JSON without formatting")
	version    = flag.Bool("version", false, "Print version and exit")
)

const appVersion = "0.1.0"

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "crev - Code Review TUI\n\n")
		fmt.Fprintf(os.Stderr, "Usage: crev [options]\n\n")
		fmt.Fprintf(os.Stderr, "Review uncommitted changes with inline comments.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nKeybindings:\n")
		fmt.Fprintf(os.Stderr, "  j/k         Navigate lines\n")
		fmt.Fprintf(os.Stderr, "  J/K         Navigate hunks\n")
		fmt.Fprintf(os.Stderr, "  h/l         Navigate files\n")
		fmt.Fprintf(os.Stderr, "  c           Add comment\n")
		fmt.Fprintf(os.Stderr, "  d           Delete comment\n")
		fmt.Fprintf(os.Stderr, "  e           Edit comment\n")
		fmt.Fprintf(os.Stderr, "  a           Approve and submit\n")
		fmt.Fprintf(os.Stderr, "  s           Submit without approval\n")
		fmt.Fprintf(os.Stderr, "  q           Quit without submitting\n")
		fmt.Fprintf(os.Stderr, "  ?           Show help\n")
	}

	flag.Parse()

	if *version {
		fmt.Printf("crev version %s\n", appVersion)
		os.Exit(0)
	}

	// Change to specified directory if provided
	if *directory != "" {
		if err := os.Chdir(*directory); err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot change to directory %s: %v\n", *directory, err)
			os.Exit(1)
		}
	}

	// Get git status for all files
	gitStatus, err := getGitStatus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting git status: %v\n", err)
		os.Exit(1)
	}

	if len(gitStatus) == 0 {
		fmt.Println("No changes to review.")
		os.Exit(0)
	}

	// Get git diff
	diffOutput, err := getGitDiff()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting git diff: %v\n", err)
		os.Exit(1)
	}

	// Parse diff
	var d *diff.Diff
	if strings.TrimSpace(diffOutput) != "" {
		d, err = diff.ParseString(diffOutput)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing diff: %v\n", err)
			os.Exit(1)
		}
	} else {
		d = &diff.Diff{}
	}

	// Build a map of files already in the diff
	diffFiles := make(map[string]bool)
	for i := range d.Files {
		diffFiles[d.Files[i].Path] = true
		// For deleted files without hunks, populate content from git
		if d.Files[i].IsDeleted && len(d.Files[i].Hunks) == 0 {
			if content, err := getFileFromGit(d.Files[i].Path); err == nil {
				lines := strings.Split(content, "\n")
				if len(lines) > 0 && lines[len(lines)-1] == "" {
					lines = lines[:len(lines)-1]
				}
				if len(lines) > 0 {
					hunk := diff.Hunk{
						OldStart: 1,
						OldCount: len(lines),
						NewStart: 0,
						NewCount: 0,
						Header:   fmt.Sprintf("@@ -1,%d +0,0 @@ (deleted file)", len(lines)),
					}
					for i, line := range lines {
						hunk.Lines = append(hunk.Lines, diff.Line{
							Type:    diff.LineRemoved,
							Content: line,
							OldNum:  i + 1,
						})
					}
					d.Files[i].Hunks = append(d.Files[i].Hunks, hunk)
				}
			}
		}
	}

	// Add untracked files from git status
	for _, status := range gitStatus {
		if status.Untracked && !diffFiles[status.Path] {
			file := diff.File{
				Path:        status.Path,
				IsNew:       true,
				IsUntracked: true,
			}
			// Read file contents and create a hunk with all lines as additions
			if content, err := os.ReadFile(status.Path); err == nil {
				lines := strings.Split(string(content), "\n")
				// Remove trailing empty line if file ends with newline
				if len(lines) > 0 && lines[len(lines)-1] == "" {
					lines = lines[:len(lines)-1]
				}
				if len(lines) > 0 {
					hunk := diff.Hunk{
						OldStart: 0,
						OldCount: 0,
						NewStart: 1,
						NewCount: len(lines),
						Header:   fmt.Sprintf("@@ -0,0 +1,%d @@ (new file)", len(lines)),
					}
					for i, line := range lines {
						hunk.Lines = append(hunk.Lines, diff.Line{
							Type:    diff.LineAdded,
							Content: line,
							NewNum:  i + 1,
						})
					}
					file.Hunks = append(file.Hunks, hunk)
				}
			}
			d.Files = append(d.Files, file)
		}
	}

	if len(d.Files) == 0 {
		fmt.Println("No changes to review.")
		os.Exit(0)
	}

	// Get base commit for reference
	baseCommit := getBaseCommit()

	// Run TUI
	model := tui.NewModel(d, *outputFile)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	// Get the final model
	m, ok := finalModel.(tui.Model)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: unexpected model type\n")
		os.Exit(1)
	}

	// Check if review was submitted
	if !m.WasSubmitted() {
		fmt.Println("Review cancelled.")
		os.Exit(0)
	}

	// Set base commit on review
	rev := m.GetReview()
	rev.BaseCommit = baseCommit

	// Output review
	pretty := !*jsonRaw
	if *outputFile != "" {
		if err := review.WriteToFile(rev, *outputFile, pretty); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing review to file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Review saved to %s\n", *outputFile)
	} else {
		if err := review.WriteJSON(rev, os.Stdout, pretty); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing review: %v\n", err)
			os.Exit(1)
		}
	}

	// Print summary
	if rev.Approved {
		fmt.Fprintln(os.Stderr, "\nReview: APPROVED")
	} else {
		fmt.Fprintln(os.Stderr, "\nReview: NOT APPROVED")
	}
	fmt.Fprintf(os.Stderr, "Comments: %d\n", rev.CommentCount())
	if rev.HasBlockers() {
		fmt.Fprintln(os.Stderr, "Warning: Review contains blockers!")
	}
}

// GitFileStatus represents the status of a file from git status
type GitFileStatus struct {
	Path      string
	Staged    bool // Has staged changes
	Unstaged  bool // Has unstaged changes
	Untracked bool // File is untracked
}

func getGitStatus() ([]GitFileStatus, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git status failed: %s", string(exitErr.Stderr))
		}
		return nil, err
	}

	var files []GitFileStatus
	for _, line := range strings.Split(string(output), "\n") {
		if len(line) < 3 {
			continue
		}
		// git status --porcelain format: XY filename
		// X = staged status, Y = unstaged status
		// ?? = untracked
		x := line[0]
		y := line[1]
		path := strings.TrimSpace(line[3:])

		// Handle renamed files (format: R  old -> new)
		if strings.Contains(path, " -> ") {
			parts := strings.Split(path, " -> ")
			path = parts[len(parts)-1]
		}

		status := GitFileStatus{Path: path}
		if x == '?' && y == '?' {
			status.Untracked = true
		} else {
			// Staged changes: any non-space, non-? in X column
			if x != ' ' && x != '?' {
				status.Staged = true
			}
			// Unstaged changes: any non-space in Y column
			if y != ' ' {
				status.Unstaged = true
			}
		}
		files = append(files, status)
	}

	return files, nil
}

func getGitDiff() (string, error) {
	// If --staged flag is set, only show staged changes
	if *staged {
		return runGitDiff("--cached")
	}

	// Get all changes: both staged and unstaged
	stagedDiff, err := runGitDiff("--cached")
	if err != nil {
		return "", err
	}

	unstagedDiff, err := runGitDiff()
	if err != nil {
		return "", err
	}

	// Combine both diffs
	combined := stagedDiff
	if combined != "" && unstagedDiff != "" {
		combined += "\n"
	}
	combined += unstagedDiff

	return combined, nil
}

func runGitDiff(args ...string) (string, error) {
	// Use --no-ext-diff to disable external diff tools (like difft)
	// and get standard unified diff format
	cmdArgs := append([]string{"diff", "--no-ext-diff", "--no-color"}, args...)
	cmd := exec.Command("git", cmdArgs...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git diff failed: %s", string(exitErr.Stderr))
		}
		return "", err
	}
	return string(output), nil
}

func getFileFromGit(path string) (string, error) {
	cmd := exec.Command("git", "show", "HEAD:"+path)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func getBaseCommit() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
