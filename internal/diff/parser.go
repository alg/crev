package diff

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var (
	// Matches: diff --git a/path b/path
	diffHeaderRe = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	// Matches: @@ -oldStart,oldCount +newStart,newCount @@ optional context
	hunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$`)
	// Matches: --- a/path or --- /dev/null
	oldFileRe = regexp.MustCompile(`^--- (?:a/(.+)|/dev/null)$`)
	// Matches: +++ b/path or +++ /dev/null
	newFileRe = regexp.MustCompile(`^\+\+\+ (?:b/(.+)|/dev/null)$`)
)

// Parse parses git diff output and returns a Diff structure
func Parse(r io.Reader) (*Diff, error) {
	scanner := bufio.NewScanner(r)
	diff := &Diff{}
	var currentFile *File
	var currentHunk *Hunk
	var oldLineNum, newLineNum int

	for scanner.Scan() {
		line := scanner.Text()

		// Check for diff header (new file in diff)
		if matches := diffHeaderRe.FindStringSubmatch(line); matches != nil {
			// Save previous file if exists
			if currentFile != nil {
				if currentHunk != nil {
					currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
				}
				diff.Files = append(diff.Files, *currentFile)
			}

			currentFile = &File{
				OldPath: matches[1],
				Path:    matches[2],
			}
			currentHunk = nil
			continue
		}

		// Skip if no current file
		if currentFile == nil {
			continue
		}

		// Check for old file marker
		if matches := oldFileRe.FindStringSubmatch(line); matches != nil {
			if matches[1] == "" {
				currentFile.IsNew = true
			}
			continue
		}

		// Check for new file marker
		if matches := newFileRe.FindStringSubmatch(line); matches != nil {
			if matches[1] == "" {
				currentFile.IsDeleted = true
			}
			continue
		}

		// Check for binary file
		if strings.HasPrefix(line, "Binary files") {
			currentFile.IsBinary = true
			continue
		}

		// Check for hunk header
		if matches := hunkHeaderRe.FindStringSubmatch(line); matches != nil {
			// Save previous hunk
			if currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}

			oldStart, _ := strconv.Atoi(matches[1])
			oldCount := 1
			if matches[2] != "" {
				oldCount, _ = strconv.Atoi(matches[2])
			}
			newStart, _ := strconv.Atoi(matches[3])
			newCount := 1
			if matches[4] != "" {
				newCount, _ = strconv.Atoi(matches[4])
			}

			currentHunk = &Hunk{
				OldStart: oldStart,
				OldCount: oldCount,
				NewStart: newStart,
				NewCount: newCount,
				Header:   line,
			}
			oldLineNum = oldStart
			newLineNum = newStart
			continue
		}

		// Parse diff lines
		if currentHunk != nil && len(line) > 0 {
			diffLine := Line{}

			switch line[0] {
			case '+':
				diffLine.Type = LineAdded
				diffLine.Content = line[1:]
				diffLine.NewNum = newLineNum
				newLineNum++
			case '-':
				diffLine.Type = LineRemoved
				diffLine.Content = line[1:]
				diffLine.OldNum = oldLineNum
				oldLineNum++
			case ' ':
				diffLine.Type = LineContext
				diffLine.Content = line[1:]
				diffLine.OldNum = oldLineNum
				diffLine.NewNum = newLineNum
				oldLineNum++
				newLineNum++
			case '\\':
				// "\ No newline at end of file" - skip
				continue
			default:
				// Unknown line type, treat as context
				diffLine.Type = LineContext
				diffLine.Content = line
				diffLine.OldNum = oldLineNum
				diffLine.NewNum = newLineNum
			}

			currentHunk.Lines = append(currentHunk.Lines, diffLine)
		}
	}

	// Don't forget the last file and hunk
	if currentFile != nil {
		if currentHunk != nil {
			currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
		}
		diff.Files = append(diff.Files, *currentFile)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return diff, nil
}

// ParseString is a convenience function to parse a diff from a string
func ParseString(s string) (*Diff, error) {
	return Parse(strings.NewReader(s))
}
