package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/UnknownOlympus/atlas/internal/metrics"
	"github.com/UnknownOlympus/atlas/internal/models"
	"github.com/UnknownOlympus/atlas/test/mocks"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestProcessTask(t *testing.T) {
	mockRepo := mocks.NewInterface(t)
	mockProvider := mocks.NewProvider(t)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	req := prometheus.NewRegistry()
	metrics := metrics.NewMetrics(req)
	ctx := t.Context()
	service := NewGeocodingServie(logger, mockRepo, mockProvider, metrics, 2, 1*time.Second)

	t.Run("successfull processing", func(t *testing.T) {
		sampleTasks := []models.Task{{ID: 1, Address: "Kyiv"}}
		sampleCoords := &models.Coordinates{Latitude: 50.45, Longitude: 30.52}

		mockRepo.On("FetchTasksForGeocoding", ctx, 100).Return(sampleTasks, nil).Once()
		mockProvider.On("Geocode", ctx, "Kyiv").Return(sampleCoords, nil).Once()
		mockRepo.On("UpdateTaskCoordinates", ctx, 1, *sampleCoords).Return(nil).Once()

		service.processTask(ctx)

		mockRepo.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
	})

	t.Run("fetch tasks return error", func(t *testing.T) {
		mockRepo.On("FetchTasksForGeocoding", ctx, 100).Return(nil, assert.AnError).Once()

		service.processTask(ctx)

		mockRepo.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
	})

	t.Run("fetch tasks return empty list", func(t *testing.T) {
		mockRepo.On("FetchTasksForGeocoding", ctx, 100).Return([]models.Task{}, nil).Once()

		service.processTask(ctx)

		mockRepo.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
	})

	t.Run("geocoding provider returns error", func(t *testing.T) {
		sampleTasks := []models.Task{{ID: 2, Address: "Invalid Address"}}
		geocodeErr := errors.New("geocoding failed")

		mockRepo.On("FetchTasksForGeocoding", ctx, 100).Return(sampleTasks, nil).Once()
		mockProvider.On("Geocode", ctx, "Invalid Address").Return(nil, geocodeErr).Once()
		mockRepo.On("IncrementFailureCount", ctx, 2, geocodeErr.Error()).Return(nil).Once()

		service.processTask(ctx)

		mockRepo.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
	})

	t.Run("error to increment failure count", func(t *testing.T) {
		sampleTasks := []models.Task{{ID: 2, Address: "Invalid Address"}}
		geocodeErr := errors.New("geocoding failed")

		mockRepo.On("FetchTasksForGeocoding", ctx, 100).Return(sampleTasks, nil).Once()
		mockProvider.On("Geocode", ctx, "Invalid Address").Return(nil, geocodeErr).Once()
		mockRepo.On("IncrementFailureCount", ctx, 2, geocodeErr.Error()).Return(assert.AnError).Once()

		service.processTask(ctx)

		mockRepo.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
	})

	t.Run("error to update task coordinates", func(t *testing.T) {
		sampleTasks := []models.Task{{ID: 1, Address: "Kyiv"}}
		sampleCoords := &models.Coordinates{Latitude: 50.45, Longitude: 30.52}

		mockRepo.On("FetchTasksForGeocoding", ctx, 100).Return(sampleTasks, nil).Once()
		mockProvider.On("Geocode", ctx, "Kyiv").Return(sampleCoords, nil).Once()
		mockRepo.On("UpdateTaskCoordinates", ctx, 1, *sampleCoords).Return(assert.AnError).Once()

		service.processTask(ctx)

		mockRepo.AssertExpectations(t)
		mockProvider.AssertExpectations(t)
	})

	t.Run("start context cancelled", func(t *testing.T) {
		tctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
		defer cancel()

		service.Run(tctx)
	})
}
