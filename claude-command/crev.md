---
allowed-tools: Bash(crev-popup)
description: Launch crev TUI for interactive code review
---

Launch the crev code review TUI in a tmux popup. Wait for the user to review changes and submit feedback.

Run: `crev-popup`

After the review completes:

**If output contains `=== CREV REVIEW SUBMITTED ===`:**
- Parse the JSON between the markers
- For each comment, quote it and respond with your action
- Address blockers and concerns first
- Ask if user wants to proceed or review again

**If output is empty:**
- User cancelled. Say "Review cancelled, no changes needed."

**Comment severities:**
- **blocker**: Must fix before proceeding
- **concern**: Should address or justify
- **question**: Answer clearly
- **suggestion**: Consider and implement if appropriate
