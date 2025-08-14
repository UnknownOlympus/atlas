package repository

import (
	"context"
	"log/slog"

	"github.com/UnknownOlympus/atlas/internal/models"
)

// Repository represents a data repository that interacts with the database
// and provides logging capabilities. It holds a reference to the database
// and a logger instance for logging operations.
type Repository struct {
	db  Database
	log *slog.Logger
}

// Interface defines the methods for interacting with geocoding tasks in the repository.
// It provides functionality to fetch tasks, update task coordinates, and increment failure counts.
type Interface interface {
	// FetchTasksForGeocoding retrieves a list of tasks for geocoding with a specified limit.
	FetchTasksForGeocoding(ctx context.Context, limit int) ([]models.Task, error)

	// UpdateTaskCoordinates updates the coordinates of a specific task identified by taskID.
	UpdateTaskCoordinates(ctx context.Context, taskID int, coords models.Coordinates) error

	// IncrementFailureCount increments the failure count for a specific task identified by taskID
	// and logs the provided error message.
	IncrementFailureCount(ctx context.Context, taskID int, errMsg string) error
}

// NewRepository creates a new instance of Repository with the provided Database.
// It returns a pointer to the newly created Repository.
func NewRepository(db Database, log *slog.Logger) *Repository {
	return &Repository{db: db, log: log}
}
