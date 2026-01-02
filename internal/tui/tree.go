package tui

import (
	"sort"
	"strings"

	"github.com/alg/crev/internal/diff"
)

// TreeNodeType distinguishes directories from files
type TreeNodeType int

const (
	TreeNodeDir TreeNodeType = iota
	TreeNodeFile
)

// TreeNode represents a node in the file tree
type TreeNode struct {
	Name      string       // Just the name (e.g., "app.go")
	Path      string       // Full path (e.g., "internal/tui/app.go")
	Type      TreeNodeType
	Children  []*TreeNode
	FileIndex int  // Index into diff.Files (-1 for dirs)
	Expanded  bool // Whether directory is expanded
	Depth     int  // Nesting depth for indentation
}

// FileTree holds the root of the tree and navigation state
type FileTree struct {
	Root          *TreeNode   // Virtual root node
	FlatList      []*TreeNode // Flattened visible nodes for navigation
	SelectedIndex int         // Index into FlatList
}

// BuildFileTree creates a tree from flat file paths
func BuildFileTree(files []diff.File) *FileTree {
	root := &TreeNode{
		Name:      "",
		Type:      TreeNodeDir,
		Children:  []*TreeNode{},
		Expanded:  true,
		Depth:     -1, // Root is invisible
		FileIndex: -1,
	}

	for i, file := range files {
		insertPath(root, file.Path, i)
	}

	// Sort children at each level
	sortChildren(root)

	tree := &FileTree{Root: root}
	tree.Rebuild()
	return tree
}

// insertPath inserts a file path into the tree, creating directories as needed
func insertPath(root *TreeNode, path string, fileIndex int) {
	parts := strings.Split(path, "/")
	current := root

	for i, part := range parts {
		isLast := i == len(parts)-1

		// Look for existing child
		var found *TreeNode
		for _, child := range current.Children {
			if child.Name == part {
				found = child
				break
			}
		}

		if found == nil {
			// Create new node
			nodeType := TreeNodeDir
			fileIdx := -1
			if isLast {
				nodeType = TreeNodeFile
				fileIdx = fileIndex
			}

			found = &TreeNode{
				Name:      part,
				Path:      strings.Join(parts[:i+1], "/"),
				Type:      nodeType,
				Children:  []*TreeNode{},
				FileIndex: fileIdx,
				Expanded:  true, // Default expanded
				Depth:     i,
			}
			current.Children = append(current.Children, found)
		}
		current = found
	}
}

// sortChildren recursively sorts children: directories first, then files, alphabetically
func sortChildren(node *TreeNode) {
	if len(node.Children) == 0 {
		return
	}

	sort.Slice(node.Children, func(i, j int) bool {
		// Directories before files
		if node.Children[i].Type != node.Children[j].Type {
			return node.Children[i].Type == TreeNodeDir
		}
		// Alphabetically within type
		return node.Children[i].Name < node.Children[j].Name
	})

	for _, child := range node.Children {
		sortChildren(child)
	}
}

// Rebuild regenerates the flat list from visible nodes
func (t *FileTree) Rebuild() {
	t.FlatList = nil
	t.buildFlatList(t.Root)

	// Clamp selected index
	if t.SelectedIndex >= len(t.FlatList) {
		t.SelectedIndex = len(t.FlatList) - 1
	}
	if t.SelectedIndex < 0 && len(t.FlatList) > 0 {
		t.SelectedIndex = 0
	}
}

func (t *FileTree) buildFlatList(node *TreeNode) {
	// Skip root itself but process children
	if node.Depth >= 0 {
		t.FlatList = append(t.FlatList, node)
	}

	if node.Type == TreeNodeDir && node.Expanded {
		for _, child := range node.Children {
			t.buildFlatList(child)
		}
	}
}

// SelectedNode returns the currently selected node, or nil if none
func (t *FileTree) SelectedNode() *TreeNode {
	if t.SelectedIndex >= 0 && t.SelectedIndex < len(t.FlatList) {
		return t.FlatList[t.SelectedIndex]
	}
	return nil
}

// SelectFileIndex finds and selects the node with the given file index
func (t *FileTree) SelectFileIndex(fileIndex int) {
	for i, node := range t.FlatList {
		if node.FileIndex == fileIndex {
			t.SelectedIndex = i
			return
		}
	}
}

// ToggleExpand toggles the expansion state of the selected node (if it's a directory)
func (t *FileTree) ToggleExpand() {
	node := t.SelectedNode()
	if node != nil && node.Type == TreeNodeDir {
		node.Expanded = !node.Expanded
		t.Rebuild()
	}
}

// MoveUp moves selection up
func (t *FileTree) MoveUp() {
	if t.SelectedIndex > 0 {
		t.SelectedIndex--
	}
}

// MoveDown moves selection down
func (t *FileTree) MoveDown() {
	if t.SelectedIndex < len(t.FlatList)-1 {
		t.SelectedIndex++
	}
}
