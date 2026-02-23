package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"gitsync/internal/database"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// @title           GitSync API
// @version         1.0
// @description     API for managing git repositories and replication targets
// @host            localhost:8080
// @BasePath        /

var allowedProviders = map[string]bool{
	"github": true,
	"gitlab": true,
	"gitea":  true,
}

type Repository struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	SourceProvider string    `json:"source_provider"`
	SourceURL      string    `json:"source_url"`
	CreatedAt      time.Time `json:"created_at"`
	Targets        []Target  `json:"targets,omitempty"`
}

type Target struct {
	ID           string    `json:"id"`
	RepositoryID string    `json:"repository_id"`
	Provider     string    `json:"provider"`
	RemoteURL    string    `json:"remote_url"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateRepositoryRequest struct {
	Name           string `json:"name"`
	SourceProvider string `json:"source_provider"`
	SourceURL      string `json:"source_url"`
}

type CreateTargetRequest struct {
	Provider  string `json:"provider"`
	RemoteURL string `json:"remote_url"`
}

type Handler struct {
	DB *database.DB
}

// HealthCheck returns the health status of the service
// @Summary Health check
// @Description Returns the health status of the service
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// CreateRepository handles POST /repositories
// @Summary Create a repository
// @Description Create a new repository with source provider and URL
// @Tags repositories
// @Accept json
// @Produce json
// @Param repository body CreateRepositoryRequest true "Repository data"
// @Success 201 {object} Repository
// @Router /repositories [post]
func (h *Handler) CreateRepository(w http.ResponseWriter, r *http.Request) {
	var req CreateRepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.SourceProvider) == "" {
		http.Error(w, "source_provider is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.SourceURL) == "" {
		http.Error(w, "source_url is required", http.StatusBadRequest)
		return
	}

	// Validate provider
	if !allowedProviders[req.SourceProvider] {
		http.Error(w, "invalid source_provider. allowed: github, gitlab, gitea", http.StatusBadRequest)
		return
	}

	// Validate URL (must start with https:// or ssh)
	if !strings.HasPrefix(req.SourceURL, "https://") && !strings.HasPrefix(req.SourceURL, "ssh://") {
		http.Error(w, "source_url must start with https:// or ssh://", http.StatusBadRequest)
		return
	}

	// Verify if repository URL already exists in the database
	var exists bool
	if err := h.DB.QueryRowContext(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM repositories WHERE source_url = $1)", req.SourceURL).Scan(&exists); err != nil {
		log.Printf("ERROR: failed to check if repository exists: %v", err)
		http.Error(w, "failed to check repository existence", http.StatusInternalServerError)
		return
	}

	if exists {
		http.Error(w, "repository with this source_url already exists", http.StatusConflict)
		return
	}

	repo := Repository{
		ID:             uuid.New().String(),
		Name:           req.Name,
		SourceProvider: req.SourceProvider,
		SourceURL:      req.SourceURL,
		CreatedAt:      time.Now(),
	}

	ctx := context.Background()
	_, err := h.DB.ExecContext(ctx,
		`INSERT INTO repositories (id, name, source_provider, source_url, created_at) 
		 VALUES ($1, $2, $3, $4, $5)`,
		repo.ID, repo.Name, repo.SourceProvider, repo.SourceURL, repo.CreatedAt)
	if err != nil {
		log.Printf("ERROR: failed to insert repository: %v", err)
		http.Error(w, "failed to create repository: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(repo)
}

// ListRepositories handles GET /repositories
// @Summary List repositories
// @Description Get all repositories with their replication targets
// @Tags repositories
// @Accept json
// @Produce json
// @Success 200 {array} Repository
// @Router /repositories [get]
func (h *Handler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Get all repositories
	rows, err := h.DB.QueryContext(ctx,
		`SELECT id, name, source_provider, source_url, created_at FROM repositories ORDER BY created_at DESC`)
	if err != nil {
		http.Error(w, "failed to fetch repositories", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var repos []Repository
	for rows.Next() {
		var repo Repository
		if err := rows.Scan(&repo.ID, &repo.Name, &repo.SourceProvider, &repo.SourceURL, &repo.CreatedAt); err != nil {
			http.Error(w, "failed to scan repository", http.StatusInternalServerError)
			return
		}

		// Get targets for this repository
		targetRows, err := h.DB.QueryContext(ctx,
			`SELECT id, repository_id, provider, remote_url, created_at 
			 FROM replication_targets WHERE repository_id = $1`, repo.ID)
		if err != nil {
			http.Error(w, "failed to fetch targets", http.StatusInternalServerError)
			return
		}

		for targetRows.Next() {
			var target Target
			if err := targetRows.Scan(&target.ID, &target.RepositoryID, &target.Provider, &target.RemoteURL, &target.CreatedAt); err != nil {
				targetRows.Close()
				http.Error(w, "failed to scan target", http.StatusInternalServerError)
				return
			}
			repo.Targets = append(repo.Targets, target)
		}
		targetRows.Close()

		repos = append(repos, repo)
	}

	if repos == nil {
		repos = []Repository{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repos)
}

// CreateTarget handles POST /repositories/{id}/targets
// @Summary Create a replication target
// @Description Add a replication target to an existing repository
// @Tags targets
// @Accept json
// @Produce json
// @Param id path string true "Repository ID"
// @Param target body CreateTargetRequest true "Target data"
// @Success 201 {object} Target
// @Router /repositories/{id}/targets [post]
func (h *Handler) CreateTarget(w http.ResponseWriter, r *http.Request) {
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

	var req CreateTargetRequest
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

	target := Target{
		ID:           uuid.New().String(),
		RepositoryID: repoID,
		Provider:     req.Provider,
		RemoteURL:    req.RemoteURL,
		CreatedAt:    time.Now(),
	}

	ctx := context.Background()
	_, err = h.DB.ExecContext(ctx,
		`INSERT INTO replication_targets (id, repository_id, provider, remote_url, created_at) 
		 VALUES ($1, $2, $3, $4, $5)`,
		target.ID, target.RepositoryID, target.Provider, target.RemoteURL, target.CreatedAt)
	if err != nil {
		http.Error(w, "failed to create target", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(target)
}
