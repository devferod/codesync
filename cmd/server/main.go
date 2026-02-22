// @title           GitSync API
// @version         1.0
// @description     API for managing git repositories and replication targets
// @host            localhost:8080
// @BasePath        /
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"gitsync/internal/database"
	"gitsync/internal/handlers"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	swaggerdocs "gitsync/docs"
)

func init() {
	// Initialize swagger docs
	swaggerdocs.SwaggerInfo.Host = "localhost:8080"
}

func main() {
	// Connect to database
	db, err := database.New()
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.RunMigrations(); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Initialize handlers
	h := &handlers.Handler{DB: db}

	// Setup router
	r := mux.NewRouter()
	r.HandleFunc("/health", h.HealthCheck).Methods("GET")
	r.HandleFunc("/repositories", h.CreateRepository).Methods("POST")
	r.HandleFunc("/repositories", h.ListRepositories).Methods("GET")
	r.HandleFunc("/repositories/{id}/targets", h.CreateTarget).Methods("POST")

	// Swagger documentation - serve swagger.json from embedded docs
	r.HandleFunc("/swagger/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs/swagger.json")
	})

	// Swagger UI
	r.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/swagger/swagger.json"),
	))

	// Get server configuration
	host := getEnv("SERVER_HOST", "0.0.0.0")
	port := getEnv("SERVER_PORT", "8080")
	addr := fmt.Sprintf("%s:%s", host, port)

	log.Printf("Starting server on %s", addr)
	log.Printf("Swagger UI available at http://%s/swagger/index.html", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
