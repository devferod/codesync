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
)

var allowedProviders = map[string]bool{
	"github": true,
	"gitlab": true,
	"gitea":  true,
}

// RepoHandler handles repository-related HTTP requests
type RepoHandler struct {
	DB *database.DB
}

// NewRepoHandler creates a new RepoHandler
func NewRepoHandler(db *database.DB) *RepoHandler {
	return &RepoHandler{DB: db}
}

// CreateRepository handles POST /repositories
// @Summary Create a repository
// @Description Create a new repository with source provider and URL
// @Tags repositories
// @Accept json
// @Produce json
// @Param repository body models.CreateRepositoryRequest true "Repository data"
// @Success 201 {object} models.Repository
// @Router /repositories [post]
func (h *RepoHandler) CreateRepository(w http.ResponseWriter, r *http.Request) {
	var req models.CreateRepositoryRequest
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

	repo := models.Repository{
		Name:           req.Name,
		SourceProvider: req.SourceProvider,
		SourceURL:      req.SourceURL,
		CreatedAt:      time.Now(),
	}

	ctx := context.Background()
	err := h.DB.QueryRowContext(ctx,
		`INSERT INTO repositories (name, source_provider, source_url, created_at) 
		 VALUES ($1, $2, $3, $4) 
		 RETURNING id`,
		repo.Name, repo.SourceProvider, repo.SourceURL, repo.CreatedAt).Scan(&repo.ID)
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
// @Success 200 {array} models.Repository
// @Router /repositories [get]
func (h *RepoHandler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Get all repositories
	rows, err := h.DB.QueryContext(ctx,
		`SELECT id, name, source_provider, source_url, created_at FROM repositories ORDER BY created_at DESC`)
	if err != nil {
		http.Error(w, "failed to fetch repositories", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var repos []models.Repository
	for rows.Next() {
		var repo models.Repository
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
			var target models.Target
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
		repos = []models.Repository{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repos)
}
