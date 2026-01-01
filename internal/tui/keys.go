package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all key bindings for the application
type KeyMap struct {
	// Navigation
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	PrevHunk  key.Binding
	NextHunk  key.Binding
	PageUp    key.Binding
	PageDown  key.Binding
	Home      key.Binding
	End       key.Binding

	// Actions
	Comment    key.Binding
	DeleteComment key.Binding
	EditComment   key.Binding
	Approve    key.Binding
	Submit     key.Binding
	Quit       key.Binding
	QuitForce  key.Binding

	// Modal
	Confirm key.Binding
	Cancel  key.Binding

	// Severity selection (in comment modal)
	Severity1 key.Binding
	Severity2 key.Binding
	Severity3 key.Binding
	Severity4 key.Binding
	NextSeverity key.Binding
	PrevSeverity key.Binding

	// Help
	Help key.Binding
}

// DefaultKeyMap returns the default key bindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/up", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/down", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("h/left", "prev file"),
		),
		Right: key.NewBinding(
			key.WithKeys("l", "right", "enter"),
			key.WithHelp("l/right", "next file"),
		),
		PrevHunk: key.NewBinding(
			key.WithKeys("K", "shift+up"),
			key.WithHelp("K", "prev hunk"),
		),
		NextHunk: key.NewBinding(
			key.WithKeys("J", "shift+down"),
			key.WithHelp("J", "next hunk"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+u", "pgup"),
			key.WithHelp("ctrl+u", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+d", "pgdown"),
			key.WithHelp("ctrl+d", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "go to top"),
		),
		End: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "go to bottom"),
		),
		Comment: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "comment"),
		),
		DeleteComment: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete comment"),
		),
		EditComment: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit comment"),
		),
		Approve: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "approve & submit"),
		),
		Submit: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "submit"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		QuitForce: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "force quit"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter", "ctrl+s"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Severity1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "suggestion"),
		),
		Severity2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "question"),
		),
		Severity3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "concern"),
		),
		Severity4: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "blocker"),
		),
		NextSeverity: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next severity"),
		),
		PrevSeverity: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev severity"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// ShortHelp returns key bindings to show in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Comment, k.Approve, k.Submit, k.Help, k.Quit}
}

// FullHelp returns key bindings to show in the full help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right, k.PrevHunk, k.NextHunk},
		{k.PageUp, k.PageDown, k.Home, k.End},
		{k.Comment, k.DeleteComment, k.EditComment},
		{k.Approve, k.Submit, k.Quit},
	}
}
