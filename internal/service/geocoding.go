package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/UnknownOlympus/atlas/internal/geocoding"
	"github.com/UnknownOlympus/atlas/internal/metrics"
	"github.com/UnknownOlympus/atlas/internal/models"
	"github.com/UnknownOlympus/atlas/internal/repository"
)

// GeocodingService provides methods for geocoding operations,
// including logging, repository access, provider integration,
// metrics tracking, and worker management.
type GeocodingService struct {
	log          *slog.Logger         // Logger for logging service activities
	repo         repository.Interface // Interface for data repository access
	provider     geocoding.Provider   // Geocoding provider for external geocoding services
	providerName string               // Name of the provider for metrics labeling
	metrics      *metrics.Metrics     // Metrics for tracking service performance
	numWorkers   int                  // Number of concurrent workers for processing
	pollInterval time.Duration        // Interval for polling geocoding updates
	addresPrefix string               // Address prefix for more accurate geocoding (indicating country, city, etc.)

}

// NewGeocodingServie creates a new instance of GeocodingService.
// It takes a logger, a repository interface, a geocoding provider,
// provider name for metrics, metrics for monitoring, the number of workers
// to use, and a polling interval for geocoding requests. It returns a pointer
// to the newly created GeocodingService.
func NewGeocodingServie(
	log *slog.Logger,
	repo repository.Interface,
	provider geocoding.Provider,
	providerName string,
	metrics *metrics.Metrics,
	numWorkers int,
	pollInterval time.Duration,
	addressPrefix string,
) *GeocodingService {
	return &GeocodingService{
		log:          log,
		repo:         repo,
		provider:     provider,
		providerName: providerName,
		metrics:      metrics,
		numWorkers:   numWorkers,
		pollInterval: pollInterval,
		addresPrefix: addressPrefix,
	}
}

// Run starts the geocoding service, which periodically polls for new tasks to geocode.
// It listens for a cancellation signal from the context to gracefully stop the service.
func (gs *GeocodingService) Run(ctx context.Context) {
	ticker := time.NewTicker(gs.pollInterval)
	defer ticker.Stop()

	gs.log.InfoContext(ctx, "Geocoding service started...")

	for {
		select {
		case <-ctx.Done():
			gs.log.InfoContext(ctx, "Goecoding service stopped.")
			return
		case <-ticker.C:
			gs.log.InfoContext(ctx, "Polling for new tasks to geocode...")
			gs.processTask(ctx)
		}
	}
}

// processTask fetches tasks for geocoding from the repository, starts a worker pool to process the tasks,
// and waits for all workers to finish. It logs errors if task fetching fails and logs the status of task processing.
func (gs *GeocodingService) processTask(ctx context.Context) {
	taskLimit := 100
	tasks, err := gs.repo.FetchTasksForGeocoding(ctx, taskLimit)
	if err != nil {
		gs.log.ErrorContext(ctx, "Failed to fetch tasks", "error", err)
		return
	}
	if len(tasks) == 0 {
		gs.log.InfoContext(ctx, "No tasks to process.")
		return
	}

	gs.log.InfoContext(
		ctx,
		"Found tasks to process. Starting worker pool.",
		"jobs",
		len(tasks),
		"num_workers",
		gs.numWorkers,
	)

	jobs := make(chan models.Task, len(tasks))
	var wgr sync.WaitGroup

	for i := 1; i <= gs.numWorkers; i++ {
		wgr.Add(1)
		go gs.worker(ctx, i, &wgr, jobs)
	}

	for _, task := range tasks {
		jobs <- task
	}
	close(jobs)

	wgr.Wait()
	gs.log.InfoContext(ctx, "Processing batch finished")
}

// worker processes tasks from the jobs channel. It increments the active worker count,
// logs the processing of each task, and measures the time taken for geocoding.
// In case of an error, it updates the failure count and logs the error.
// On successful geocoding, it updates the task with the obtained coordinates.
// The function takes a context, an index for the worker, a wait group to signal completion,
// and a channel of tasks to process.
func (gs *GeocodingService) worker(ctx context.Context, idx int, wg *sync.WaitGroup, jobs <-chan models.Task) {
	defer wg.Done()
	for task := range jobs {
		var err error

		gs.metrics.ActiveWorkers.Inc()
		gs.log.DebugContext(ctx, "Processing task", "worker", idx, "task", task.ID)

		task.Address = gs.addresPrefix + task.Address
		startTime := time.Now()
		coords, err := gs.provider.Geocode(ctx, task.Address)
		duration := time.Since(startTime).Seconds()
		gs.metrics.RequestSeconds.WithLabelValues(gs.providerName).Observe(duration)

		if err != nil {
			gs.log.ErrorContext(ctx, "Failed to geocode", "worker", idx, "task", task.ID, "error", err)
			gs.metrics.TaskProcessed.WithLabelValues("failure").Inc()
			gs.metrics.APIErrors.Inc()

			if err = gs.repo.IncrementFailureCount(ctx, task.ID, err.Error()); err != nil {
				gs.log.ErrorContext(
					ctx,
					"Could not update failure count for task",
					"worker", idx,
					"task", task.ID,
					"error", err,
				)
			}
			gs.metrics.ActiveWorkers.Dec()
			continue
		}

		gs.metrics.TaskProcessed.WithLabelValues("success").Inc()

		if err = gs.repo.UpdateTaskCoordinates(ctx, task.ID, *coords); err != nil {
			gs.log.ErrorContext(
				ctx,
				"Failed to update coordinates for task",
				"worker", idx,
				"task", task.ID,
				"error", err,
			)
		} else {
			gs.log.DebugContext(ctx, "Worker successfully processed the task", "worker", idx, "task", task.ID)
		}

		gs.metrics.ActiveWorkers.Dec()
	}
}
