package web

import (
	"encoding/json"
	"net/http"
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
