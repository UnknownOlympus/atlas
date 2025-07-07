package repository

import (
	"context"
	"log/slog"

	"github.com/Houeta/geocoding-service/internal/models"
)

type Repository struct {
	db  Database
	log *slog.Logger
}

type Interface interface {
	FetchTasksForGeocoding(ctx context.Context, limit int) ([]models.Task, error)
	UpdateTaskCoordinates(ctx context.Context, taskID int, coords models.Coordinates) error
	IncrementFailureCount(ctx context.Context, taskID int, errMsg string) error
}

// NewRepository creates a new instance of Repository with the provided Database.
// It returns a pointer to the newly created Repository.
func NewRepository(db Database, log *slog.Logger) *Repository {
	return &Repository{db: db, log: log}
}
