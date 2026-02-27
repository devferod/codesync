package models

import "time"

// Repository represents a git repository to be replicated
type Repository struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	SourceProvider string    `json:"source_provider"`
	SourceURL      string    `json:"source_url"`
	CreatedAt      time.Time `json:"created_at"`
	Targets        []Target  `json:"targets,omitempty"`
}

// Target represents a replication target for a repository
type Target struct {
	ID           string    `json:"id"`
	RepositoryID string    `json:"repository_id"`
	Provider     string    `json:"provider"`
	RemoteURL    string    `json:"remote_url"`
	CreatedAt    time.Time `json:"created_at"`
}

// CreateRepositoryRequest is the request body for creating a repository
type CreateRepositoryRequest struct {
	Name           string `json:"name"`
	SourceProvider string `json:"source_provider"`
	SourceURL      string `json:"source_url"`
}

// CreateTargetRequest is the request body for creating a target
type CreateTargetRequest struct {
	Provider  string `json:"provider"`
	RemoteURL string `json:"remote_url"`
}
