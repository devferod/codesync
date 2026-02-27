package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"gitsync/internal/database"
	"gitsync/internal/models"

	"github.com/gorilla/mux"
)

// TargetHandler handles target-related HTTP requests
type TargetHandler struct {
	DB *database.DB
}

// NewTargetHandler creates a new TargetHandler
func NewTargetHandler(db *database.DB) *TargetHandler {
	return &TargetHandler{DB: db}
}

// CreateTarget handles POST /repositories/{id}/targets
// @Summary Create a replication target
// @Description Add a replication target to an existing repository
// @Tags targets
// @Accept json
// @Produce json
// @Param id path string true "Repository ID"
// @Param target body models.CreateTargetRequest true "Target data"
// @Success 201 {object} models.Target
// @Router /repositories/{id}/targets [post]
func (h *TargetHandler) CreateTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	repoID := vars["id"]

	// Verify repository exists
	var exists bool
	err := h.DB.QueryRowContext(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM repositories WHERE id = $1)", repoID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "repository not found", http.StatusNotFound)
		return
	}

	var req models.CreateTargetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate provider
	if strings.TrimSpace(req.Provider) == "" {
		http.Error(w, "provider is required", http.StatusBadRequest)
		return
	}
	if !allowedProviders[req.Provider] {
		http.Error(w, "invalid provider. allowed: github, gitlab, gitea", http.StatusBadRequest)
		return
	}

	// Validate URL
	if strings.TrimSpace(req.RemoteURL) == "" {
		http.Error(w, "remote_url is required", http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(req.RemoteURL, "https://") && !strings.HasPrefix(req.RemoteURL, "ssh://") {
		http.Error(w, "remote_url must start with https:// or ssh://", http.StatusBadRequest)
		return
	}

	// Verify if target URL already exists for this repository
	var target_exists bool
	if err := h.DB.QueryRowContext(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM replication_targets WHERE repository_id = $1 AND remote_url = $2)",
		repoID, req.RemoteURL).Scan(&target_exists); err != nil {
		log.Printf("ERROR: failed to check if target exists: %v", err)
		http.Error(w, "failed to check target existence", http.StatusInternalServerError)
		return
	}

	if target_exists {
		http.Error(w, "target with this remote_url already exists for this repository", http.StatusConflict)
		return
	}

	target := models.Target{
		RepositoryID: repoID,
		Provider:     req.Provider,
		RemoteURL:    req.RemoteURL,
		CreatedAt:    time.Now(),
	}

	ctx := context.Background()
	err = h.DB.QueryRowContext(ctx,
		`INSERT INTO replication_targets (repository_id, provider, remote_url, created_at) 
		 VALUES ($1, $2, $3, $4) 
		 RETURNING id`,
		target.RepositoryID, target.Provider, target.RemoteURL, target.CreatedAt).Scan(&target.ID)
	if err != nil {
		http.Error(w, "failed to create target", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(target)
}
