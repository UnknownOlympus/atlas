package geocoding

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Houeta/geocoding-service/internal/models"
	"googlemaps.github.io/maps"
)

type GoogleProvider struct {
	client *maps.Client
	log    *slog.Logger
}

var ErrEmptyResponse = errors.New("get empty response from Google Maps API")

func NewGoogleProvider(apiKey string, log *slog.Logger, workers int) (*GoogleProvider, error) {
	client, err := maps.NewClient(maps.WithAPIKey(apiKey), maps.WithRateLimit((50 / workers)))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Google client: %w", err)
	}

	return &GoogleProvider{client: client, log: log}, nil
}

func (gp *GoogleProvider) Geocode(ctx context.Context, address string) (*models.Coordinates, error) {
	gp.log.DebugContext(ctx, "Geocoding using Google Maps", "address", address)

	req := maps.GeocodingRequest{Address: address}
	geocodeResponse, err := gp.client.Geocode(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("failed to geocode address: %w", err)
	}

	if len(geocodeResponse) == 0 {
		return nil, ErrEmptyResponse
	}
	coords := geocodeResponse[0].Geometry.Location

	return &models.Coordinates{Longitude: coords.Lng, Latidude: coords.Lat}, nil
}
