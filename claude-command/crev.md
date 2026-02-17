---
allowed-tools: Bash(crev-web)
description: Launch crev web UI for interactive code review
---

Launch the crev web-based code review UI. Wait for the user to review changes and submit feedback.

Run: `crev-web`

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
