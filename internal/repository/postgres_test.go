package repository_test

import (
	"log/slog"
	"regexp"
	"testing"

	"github.com/Houeta/geocoding-service/internal/models"
	"github.com/Houeta/geocoding-service/internal/repository"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fetchTasksQuery = `
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

func TestFetchTasksForGeocoding(t *testing.T) {
	t.Parallel()
	logger := slog.Default()
	ctx := t.Context()
	limit := 10

	t.Run("error - query active tasks", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock, logger)

		mock.ExpectQuery(regexp.QuoteMeta(fetchTasksQuery)).
			WithArgs(limit).
			WillReturnError(assert.AnError)

		tasks, err := repo.FetchTasksForGeocoding(ctx, limit)

		require.Nil(t, tasks)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to query active tasks")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - scan active tasks", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock, logger)

		mock.ExpectQuery(regexp.QuoteMeta(fetchTasksQuery)).
			WithArgs(limit).
			WillReturnRows(
				pgxmock.NewRows([]string{"task_id", "address"}).AddRow("invalid_id", "valid address"),
			)

		tasks, err := repo.FetchTasksForGeocoding(ctx, limit)

		require.Nil(t, tasks)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to scan active task")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - rows error", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock, logger)

		mock.ExpectQuery(regexp.QuoteMeta(fetchTasksQuery)).
			WithArgs(limit).
			WillReturnRows(
				pgxmock.NewRows([]string{"task_id", "address"}).AddRow(123, "valid address").
					RowError(1, assert.AnError),
			)

		tasks, err := repo.FetchTasksForGeocoding(ctx, limit)

		require.Nil(t, tasks)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed to read row")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - fetch tasks with address", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock, logger)

		mock.ExpectQuery(regexp.QuoteMeta(fetchTasksQuery)).
			WithArgs(limit).
			WillReturnRows(
				pgxmock.NewRows([]string{"task_id", "address"}).AddRow(123, "valid address"),
			)

		tasks, err := repo.FetchTasksForGeocoding(ctx, limit)
		task := tasks[0]

		assert.Equal(t, 123, task.ID)
		assert.Equal(t, "valid address", task.Address)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUpdateTasCoordinates(t *testing.T) {
	t.Parallel()
	logger := slog.Default()
	ctx := t.Context()
	taskID := 123
	coords := models.Coordinates{
		Longitude: 123.123,
		Latitude:  456.456,
	}
	query := `
		UPDATE tasks
		SET
			latitude = $1,
			longitude = $2,
			geocoding_error = NULL
		WHERE
			task_id = $3;
	`

	t.Run("error - update task coords", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock, logger)

		mock.ExpectExec(regexp.QuoteMeta(query)).WithArgs(coords.Latitude, coords.Longitude, taskID).
			WillReturnError(assert.AnError)

		err = repo.UpdateTaskCoordinates(ctx, taskID, coords)

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to update task coordinates")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - update task coords", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock, logger)

		mock.ExpectExec(regexp.QuoteMeta(query)).WithArgs(coords.Latitude, coords.Longitude, taskID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err = repo.UpdateTaskCoordinates(ctx, taskID, coords)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIncrementFailureCount(t *testing.T) {
	t.Parallel()
	logger := slog.Default()
	ctx := t.Context()
	taskID := 123
	query := `
		UPDATE tasks
		SET
			geocoding_attempts = geocoding_attempts + 1,
			geocoding_error = $1
		WHERE task_id = $2;
	`

	t.Run("error - increment failure count", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock, logger)

		mock.ExpectExec(regexp.QuoteMeta(query)).WithArgs("error", taskID).
			WillReturnError(assert.AnError)

		err = repo.IncrementFailureCount(ctx, taskID, "error")

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to update geocoding error and number of attempts")
		require.ErrorIs(t, err, assert.AnError)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - increment failure count", func(t *testing.T) {
		t.Parallel()
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		repo := repository.NewRepository(mock, logger)

		mock.ExpectExec(regexp.QuoteMeta(query)).WithArgs("error", taskID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err = repo.IncrementFailureCount(ctx, taskID, "error")

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
