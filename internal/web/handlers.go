package web

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/alg/crev/internal/diff"
	"github.com/alg/crev/internal/review"
)

type handlers struct {
	diff       *diff.Diff
	baseCommit string
	submitCh   chan *review.Review
}

type diffResponse struct {
	Files      []diff.File `json:"files"`
	BaseCommit string      `json:"base_commit"`
}

func (h *handlers) handleDiff(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := diffResponse{
		Files:      h.diff.Files,
		BaseCommit: h.baseCommit,
	}
	json.NewEncoder(w).Encode(resp)
}

type submitRequest struct {
	Comments []review.Comment `json:"comments"`
	Summary  string           `json:"summary"`
	Approved bool             `json:"approved"`
}

func (h *handlers) handleSubmit(w http.ResponseWriter, r *http.Request) {
	var req submitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	rev := &review.Review{
		Timestamp:  time.Now().UTC(),
		BaseCommit: h.baseCommit,
		Comments:   req.Comments,
		Summary:    req.Summary,
		Approved:   req.Approved,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})

	h.submitCh <- rev
}

func (h *handlers) handleFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	// Only serve files that are in the diff
	found := false
	for _, f := range h.diff.Files {
		if f.Path == path {
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "file not in review", http.StatusForbidden)
		return
	}

	// Prevent path traversal
	clean := filepath.Clean(path)
	if filepath.IsAbs(clean) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(clean)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", http.DetectContentType(data))
	w.Write(data)
}
