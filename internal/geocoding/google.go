package geocoding

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Houeta/geocoding-service/internal/models"
	"googlemaps.github.io/maps"
)

// GoogleProvider is a struct that holds the client for Google Maps API
// and a logger for logging purposes. It is used to interact with the
// Google Maps geocoding services.
type GoogleProvider struct {
	client *maps.Client // client is the Google Maps API client
	log    *slog.Logger // log is the logger for logging operations
}

const googleReqLimit = 50

// ErrEmptyResponse is returned when the Google Maps API responds with an empty result.
var ErrEmptyResponse = errors.New("get empty response from Google Maps API")

// NewGoogleProvider initializes a new GoogleProvider with the given API key, logger, and number of workers.
// It creates a Google Maps client with rate limiting based on the number of workers.
// Returns a pointer to the GoogleProvider and an error if the client initialization fails.
func NewGoogleProvider(apiKey string, log *slog.Logger, workers int) (*GoogleProvider, error) {
	client, err := maps.NewClient(maps.WithAPIKey(apiKey), maps.WithRateLimit((googleReqLimit / workers)))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Google client: %w", err)
	}

	return &GoogleProvider{client: client, log: log}, nil
}

// Geocode takes a context and an address string as input, and returns the geographical coordinates
// (longitude and latitude) of the provided address using the Google Maps Geocoding API.
// It logs the geocoding request and handles any errors that may occur during the process.
// If the address cannot be geocoded or if the response is empty, it returns an appropriate error.
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
