package geocoding

import (
	"errors"
	"fmt"
	"log/slog"

	"googlemaps.github.io/maps"
)

// ProviderType represents the type of geocoding provider.
type ProviderType string

const (
	// ProviderTypeGoogle represents Google Maps geocoding provider.
	ProviderTypeGoogle ProviderType = "google"
	// ProviderTypeNominatim represents OpenStreetMap Nominatim geocoding provider.
	ProviderTypeNominatim ProviderType = "nominatim"
)

// ProviderConfig holds configuration for creating a geocoding provider.
type ProviderConfig struct {
	Type      ProviderType // Type of provider to create
	APIKey    string       // API key (used by Google provider)
	RateLimit int          // Rate limit for requests per second (used by Google provider)
	Logger    *slog.Logger // Logger for the provider
}

// NewProvider creates a geocoding provider based on the provided configuration.
// It applies the Factory pattern to decouple provider instantiation from business logic.
//
// Supported provider types:
// - "google": Google Maps Geocoding API (requires API key)
// - "nominatim": OpenStreetMap Nominatim API (free, no API key required)
//
// Returns an error if the provider type is unsupported or if provider creation fails.
func NewProvider(config ProviderConfig) (Provider, error) {
	switch config.Type {
	case ProviderTypeGoogle:
		return newGoogleProvider(config)
	case ProviderTypeNominatim:
		return newNominatimProvider(config)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}
}

// newGoogleProvider creates a Google Maps geocoding provider.
func newGoogleProvider(config ProviderConfig) (Provider, error) {
	if config.APIKey == "" {
		return nil, errors.New("API key is required for Google provider")
	}

	// Create Google Maps client with API key and rate limiting
	clientOpts := []maps.ClientOption{
		maps.WithAPIKey(config.APIKey),
	}

	// Apply rate limiting if specified
	if config.RateLimit > 0 {
		clientOpts = append(clientOpts, maps.WithRateLimit(config.RateLimit))
	}

	client, err := maps.NewClient(clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Maps client: %w", err)
	}

	return NewGoogleProvider(client, config.Logger), nil
}

// newNominatimProvider creates a Nominatim geocoding provider.
func newNominatimProvider(config ProviderConfig) (Provider, error) {
	// Nominatim is free and doesn't require an API key
	return NewNominatimProvider(config.Logger), nil
}
