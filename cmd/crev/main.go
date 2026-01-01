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

	// Get git diff
	diffOutput, err := getGitDiff()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting git diff: %v\n", err)
		os.Exit(1)
	}

	if strings.TrimSpace(diffOutput) == "" {
		fmt.Println("No changes to review.")
		os.Exit(0)
	}

	// Parse diff
	d, err := diff.ParseString(diffOutput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing diff: %v\n", err)
		os.Exit(1)
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

func getGitDiff() (string, error) {
	// If --staged flag is set, show staged changes
	if *staged {
		return getGitDiffStaged()
	}

	// First try unstaged changes
	output, err := runGitDiff()
	if err != nil {
		return "", err
	}

	// If no unstaged changes, try staged changes
	if strings.TrimSpace(output) == "" {
		return getGitDiffStaged()
	}

	return output, nil
}

func getGitDiffStaged() (string, error) {
	return runGitDiff("--cached")
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

func getBaseCommit() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
