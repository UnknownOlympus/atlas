package repository

import (
	"context"
	"fmt"

	"github.com/Houeta/geocoding-service/internal/models"
)

// FetchTasksForGeocoding retrieves a list of tasks that require geocoding.
// It returns tasks that have a NULL latitude, are not closed, have fewer than 5 geocoding attempts,
// and have a non-empty address. The results are ordered by creation date and limited to the specified count.
//
// Parameters:
// - ctx: The context for the operation, allowing for cancellation and timeout.
// - limit: The maximum number of tasks to retrieve.
//
// Returns:
// - A slice of models.Task containing the tasks that match the criteria.
// - An error if the query fails or if there is an issue scanning the results.
func (r *Repository) FetchTasksForGeocoding(ctx context.Context, limit int) ([]models.Task, error) {
	var tasks []models.Task
	query := `
		SELECT task_id, address
		FROM public.tasks
		WHERE
			latitude IS NULL
			AND is_closed = false
			AND geocoding_attempts < 5
			AND address IS NOT NULL AND address <> ''
		ORDER BY created_at ASC
		LIMIT $1;
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query active tasks with address: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var task models.Task
		if errScan := rows.Scan(&task.ID, &task.Address); errScan != nil {
			return nil, fmt.Errorf("failed to scan active task with address: %w", errScan)
		}
		r.log.DebugContext(ctx, "A new active task without coordinates has been received.",
			"ID", task.ID, "Address", task.Address)
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read row: %w", err)
	}

	return tasks, nil
}

// UpdateTaskCoordinates updates the latitude and longitude of a task identified by taskID.
// It sets the geocoding_error field to NULL. It returns an error if the update fails.
func (r *Repository) UpdateTaskCoordinates(ctx context.Context, taskID int, coords models.Coordinates) error {
	query := `
		UPDATE tasks
		SET
			latitude = $1,
			longitude = $2,
			geocoding_error = NULL
		WHERE
			task_id = $3;
	`

	_, err := r.db.Exec(ctx, query, coords.Latidude, coords.Longitude, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task ccordinates: %w", err)
	}

	return nil
}

// IncrementFailureCount increments the geocoding attempt count for a specific task
// identified by taskID and updates the associated error message. It takes a context
// for managing request-scoped values, cancellation, and deadlines. If the update
// operation fails, it returns an error with additional context.
func (r *Repository) IncrementFailureCount(ctx context.Context, taskID int, errMsg string) error {
	query := `
		UPDATE tasks
		SET
			geocoding_attempts = geocoding_attempts + 1,
			geocoding_error = $1
		WHERE task_id = $2;
	`

	_, err := r.db.Exec(ctx, query, errMsg, taskID)
	if err != nil {
		return fmt.Errorf("failed to update geocoding error and number of attempts: %w", err)
	}

	return nil
}
