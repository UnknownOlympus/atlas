package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Houeta/geocoding-service/internal/config"
	"github.com/Houeta/geocoding-service/internal/geocoding"
	"github.com/Houeta/geocoding-service/internal/metrics"
	"github.com/Houeta/geocoding-service/internal/repository"
	"github.com/Houeta/geocoding-service/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"googlemaps.github.io/maps"
)

// Constants for different environment types.
const (
	envLocal = "local"
	envDev   = "development"
	envProd  = "production"
)

// main is the entry point of the application.
func main() {
	// Create a context that will be canceled when an interrupt signal is received.
	// This allows for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// Load application configuration.
	cfg := config.MustLoad()

	// Set up the logger based on the environment.
	logger := setupLogger(cfg.Env)

	// Create a separate registry for metrics with exemplar
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	appMetrics := metrics.NewMetrics(reg)

	// Initialize the database connection.
	dtb, err := repository.NewDatabase(
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name,
	)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	// Create a new repository instance using the database connection.
	repo := repository.NewRepository(dtb, logger)

	// Create a new geocode provider
	googleRateLimit := 50
	client, err := maps.NewClient(maps.WithAPIKey(cfg.APIKey), maps.WithRateLimit((googleRateLimit / cfg.Workers)))
	if err != nil {
		log.Fatalf("Failed to add geocoder provider: %v", err)
	}
	defer stop()

	geoProvider := geocoding.NewGoogleProvider(client, logger)

	// Init a new geocode service using the geo provider.
	geoService := service.NewGeocodingServie(logger, repo, geoProvider, appMetrics, cfg.Workers, cfg.Interval)

	// Log that the application has started.
	logger.InfoContext(ctx, "Application started. Press Ctrl+C to stop.")

	// Start the monitoring server in a goroutine to allow main to listen for signals.
	go startMonitoringServer(ctx, logger, reg, dtb, cfg.Port)

	go geoService.Run(ctx)

	// Wait for the context to be canceled (e.g., by Ctrl+C).
	<-ctx.Done()

	// Log that a shutdown signal has been received.
	logger.InfoContext(ctx, "Shutdown signal received. Stopping application...")

	// Log graceful shutdown completion.
	logger.InfoContext(ctx, "Application stopped gracefully.")
}

// startMonitoringServer starts an HTTP server that provides health check and metrics endpoints.
// It listens on the specified port and logs the server's status and any errors encountered.
//
// Parameters:
// - ctx: A context.Context for managing cancellation and timeouts.
// - log: A logger for logging server events and errors.
// - reg: A registry with Prometheus collectors.
// - dtb: A pgxpool connector for database methods (ping)
// - port: The port number on which the server will listen.
func startMonitoringServer(
	ctx context.Context,
	log *slog.Logger,
	reg *prometheus.Registry,
	dtb *pgxpool.Pool,
	port int,
) {
	http.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		log.DebugContext(ctx, "Performing health checks...")
		status, body := http.StatusOK, "OK"
		if err := dtb.Ping(ctx); err != nil {
			status, body = http.StatusServiceUnavailable, "DB ping failed"
		}
		writer.WriteHeader(status)
		_, err := writer.Write([]byte(body))
		if err != nil {
			log.ErrorContext(ctx, "failed to write reply", "error", err)
		}

		log.DebugContext(ctx, "Health checks completed", "status", http.StatusOK)
	})
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	log.InfoContext(ctx, "Starting monitoring server", "port", port)
	readTimeout := 5
	writeTimeout := 10
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      http.DefaultServeMux,
		ReadTimeout:  time.Duration(readTimeout) * time.Second,
		WriteTimeout: time.Duration(writeTimeout) * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.ErrorContext(ctx, "Monitoring server failed", "error", err)
	}
}

// setupLogger initializes and returns a logger based on the environment provided.
func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelDebug,
				AddSource: true,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					return a
				},
			}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelInfo,
				AddSource: false,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					return a
				},
			}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelWarn,
				AddSource: false,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					if a.Key == slog.TimeKey {
						return slog.Attr{}
					}
					return a
				},
			}),
		)
	default:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelError,
				AddSource: false,
				ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
					if a.Key == slog.TimeKey {
						return slog.Attr{}
					}
					return a
				},
			}),
		)

		log.Error(
			"The env parameter was not specified	 or was invalid. Logging will be minimal, by default.",
			slog.String("available_envs", "local, development, production"))
	}

	return log
}
