package handlers

import (
	"encoding/json"
	"net/http"

	"gitsync/internal/database"
)

// Handler is a facade that delegates to specialized handlers
type Handler struct {
	*RepoHandler
	*TargetHandler
}

// NewHandler creates a new Handler with all sub-handlers
func NewHandler(db *database.DB) *Handler {
	return &Handler{
		RepoHandler:   NewRepoHandler(db),
		TargetHandler: NewTargetHandler(db),
	}
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

// CreateRepository delegates to RepoHandler
func (h *Handler) CreateRepository(w http.ResponseWriter, r *http.Request) {
	h.RepoHandler.CreateRepository(w, r)
}

// ListRepositories delegates to RepoHandler
func (h *Handler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	h.RepoHandler.ListRepositories(w, r)
}

// CreateTarget delegates to TargetHandler
func (h *Handler) CreateTarget(w http.ResponseWriter, r *http.Request) {
	h.TargetHandler.CreateTarget(w, r)
}
