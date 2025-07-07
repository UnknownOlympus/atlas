package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Houeta/geocoding-service/internal/geocoding"
	"github.com/Houeta/geocoding-service/internal/metrics"
	"github.com/Houeta/geocoding-service/internal/models"
	"github.com/Houeta/geocoding-service/internal/repository"
)

type GeocodingService struct {
	log          *slog.Logger
	repo         repository.Interface
	provider     geocoding.Provider
	metrics      *metrics.Metrics
	numWorkers   int
	pollInterval time.Duration
}

func NewGeocodingServie(
	log *slog.Logger,
	repo repository.Interface,
	provider geocoding.Provider,
	metrics *metrics.Metrics,
	numWorkers int,
	pollInterval time.Duration,
) *GeocodingService {
	return &GeocodingService{
		log:          log,
		repo:         repo,
		provider:     provider,
		metrics:      metrics,
		numWorkers:   numWorkers,
		pollInterval: pollInterval,
	}
}

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

func (gs *GeocodingService) processTask(ctx context.Context) {
	tasks, err := gs.repo.FetchTasksForGeocoding(ctx, 100)
	if err != nil {
		gs.log.ErrorContext(ctx, "Failed to fetch tasks", "error", err)
		return
	}
	if len(tasks) == 0 {
		gs.log.InfoContext(ctx, "No tasks to process.")
		return
	}

	gs.log.InfoContext(ctx, "Found tasks to process. Starting worker pool.", "jobs", len(tasks), "num_workers", gs.numWorkers)

	jobs := make(chan models.Task, len(tasks))
	var wg sync.WaitGroup

	for i := 1; i <= gs.numWorkers; i++ {
		wg.Add(1)
		go gs.worker(ctx, i, &wg, jobs)
	}

	for _, task := range tasks {
		jobs <- task
	}
	close(jobs)

	wg.Wait()
	gs.log.InfoContext(ctx, "Processing batch finished")
}

func (gs *GeocodingService) worker(ctx context.Context, id int, wg *sync.WaitGroup, jobs <-chan models.Task) {
	defer wg.Done()
	for task := range jobs {
		gs.metrics.ActiveWorkers.Inc()
		gs.log.DebugContext(ctx, "Processing task", "worker", id, "task", task.ID)

		startTime := time.Now()
		coords, err := gs.provider.Geocode(ctx, task.Address)
		duration := time.Since(startTime).Seconds()
		gs.metrics.RequestSeconds.WithLabelValues("google").Observe(duration)

		if err != nil {
			gs.log.ErrorContext(ctx, "Failed to geocode", "worker", id, "task", task.ID)
			gs.metrics.TaskProcessed.WithLabelValues("failure").Inc()
			gs.metrics.APIErrors.Inc()

			if err := gs.repo.IncrementFailureCount(ctx, task.ID, err.Error()); err != nil {
				gs.log.ErrorContext(ctx, "Could not update failure count for task", "worker", id, "task", task.ID, "error", err)
			}
			gs.metrics.ActiveWorkers.Dec()
			continue
		}

		gs.metrics.TaskProcessed.WithLabelValues("success").Inc()

		if err := gs.repo.UpdateTaskCoordinates(ctx, task.ID, *coords); err != nil {
			gs.log.ErrorContext(ctx, "Failed to update coordinates for task", "worker", id, "task", task.ID, "error", err)
		} else {
			gs.log.DebugContext(ctx, "Worker successfully processed the task", "worker", id, "task", task.ID)
		}

		gs.metrics.ActiveWorkers.Dec()
	}
}
