package repository

import (
	"context"
	"fmt"

	"github.com/Houeta/geocoding-service/internal/models"
)

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
