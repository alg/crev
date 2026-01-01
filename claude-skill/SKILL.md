---
name: review
description: Request code review for uncommitted changes. Use when asking for review of AI-generated code, reviewing git diff, or when user says "review my changes", "let me review", or "I want to check the changes".
---

# Code Review Skill

Review uncommitted changes using the `crev` TUI in a tmux popup.

## Workflow

1. **Summarize Changes**: Briefly list files modified and key changes
2. **Launch Review**: Run `crev-popup` to open crev in a tmux popup
3. **Process Feedback**: Parse the JSON output and address comments

## How to Launch

Run the crev-popup script which opens crev in a tmux popup:

```bash
crev-popup
```

This will:
- Open crev in a 90% tmux popup with full TTY
- Wait for user to review and submit (or cancel)
- Output the review JSON if submitted
- Output nothing if cancelled (user pressed 'q')

## Handling Output

**If output contains `=== CREV REVIEW SUBMITTED ===`:**
Parse the JSON between the markers and address each comment.

**If no output (empty):**
User cancelled the review. Say "Review cancelled, no changes needed."

## JSON Format

```json
{
  "timestamp": "2025-01-01T10:30:00Z",
  "base_commit": "abc123",
  "comments": [
    {
      "file": "src/example.py",
      "line_start": 42,
      "line_end": 42,
      "side": "new",
      "severity": "question",
      "text": "Why did you use this approach?"
    }
  ],
  "summary": "Overall looks good",
  "approved": true
}
```

## Handling Comments by Severity

- **blocker**: Must address before proceeding. Fix the issue or discuss why it can't be fixed.
- **concern**: Should address. Either fix or provide clear justification.
- **question**: Answer the question clearly. May lead to code changes.
- **suggestion**: Consider and implement if appropriate. OK to defer with explanation.

## Response Format

After reading feedback, respond with:

1. Acknowledgment of the review
2. For each comment:
   - Quote the comment text
   - Your response/action
3. Summary of changes made (if any)
4. Ask if user wants to proceed

Example:
```
Thanks for the review! Here's my response to your comments:

**src/auth.py:45** (question): "Why use retry here?"
→ Good catch! The retry handles transient network failures. I've added a comment explaining this.

**src/auth.py:52** (suggestion): "Consider extracting this logic"
→ Agreed, I've moved this to a separate `validate_with_retry()` function.

I've made 2 changes based on your feedback. Would you like to review again, or shall we proceed?
```

## Keybindings Reference (for user assistance)

```
Navigation:
  j/k         Navigate lines
  J/K         Navigate hunks
  h/l         Navigate files

Actions:
  c           Add comment on current line
  d           Delete comment
  e           Edit comment
  1-4         Set severity (in comment modal)

Submit:
  a           Approve and submit
  s           Submit without approval
  q           Quit without submitting
```
