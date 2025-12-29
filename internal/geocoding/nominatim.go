package geocoding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/UnknownOlympus/atlas/internal/models"
)

// NominatimProvider implements the Provider interface using OpenStreetMap's Nominatim API.
// This is a free geocoding service with usage limits (1 request/second for fair use).
type NominatimProvider struct {
	client  HTTPClient   // HTTP client for making requests
	baseURL string       // Base URL for the Nominatim API
	log     *slog.Logger // Logger for logging operations
	// userAgent is required by Nominatim usage policy
	userAgent string
}

// HTTPClient defines the interface for making HTTP requests.
// This allows for easy mocking in tests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// nominatimResponse represents the JSON response from Nominatim API.
type nominatimResponse struct {
	Lat string `json:"lat"` // Latitude as string
	Lon string `json:"lon"` // Longitude as string
}

// Common errors for Nominatim provider.
var (
	ErrNominatimEmptyResponse = errors.New("nominatim API returned empty response")
	ErrNominatimInvalidCoords = errors.New("nominatim API returned invalid coordinates")
)

// NewNominatimProvider creates a new Nominatim geocoding provider.
// Uses the public Nominatim API endpoint by default.
func NewNominatimProvider(log *slog.Logger) *NominatimProvider {
	const timeout = 10
	return &NominatimProvider{
		client: &http.Client{
			Timeout: timeout * time.Second,
		},
		baseURL: "https://nominatim.openstreetmap.org/search",
		log:     log,
		// User-Agent MUST include valid contact info per Nominatim usage policy:
		// https://operations.osmfoundation.org/policies/nominatim/
		userAgent: "Atlas-Geocoding-Service/1.0 (https://github.com/UnknownOlympus/atlas)",
	}
}

// NewNominatimProviderWithClient creates a Nominatim provider with a custom HTTP client.
// Useful for testing with mocked HTTP clients.
func NewNominatimProviderWithClient(client HTTPClient, log *slog.Logger) *NominatimProvider {
	return &NominatimProvider{
		client:    client,
		baseURL:   "https://nominatim.openstreetmap.org/search",
		log:       log,
		userAgent: "Atlas-Geocoding-Service/1.0 (https://github.com/UnknownOlympus/atlas)",
	}
}

// Geocode converts an address to geographic coordinates using the Nominatim API.
// It respects Nominatim's usage policy by including a User-Agent header.
//
// Uses a progressive fallback strategy for rural addresses:
// 1. Try full address with house number
// 2. Try address without house number (e.g., "с. Грабовець, вул. Польова")
// 3. Try village/town name only (e.g., "с. Грабовець")
// 4. Try district level
//
// Note: Nominatim has a rate limit of 1 request/second for fair use.
// For production use with high volume, consider self-hosting Nominatim or using a commercial provider.
func (np *NominatimProvider) Geocode(ctx context.Context, address string) (*models.Coordinates, error) {
	np.log.DebugContext(ctx, "Geocoding using Nominatim", "address", address)

	// Generate address fallback variations
	addressVariations := np.generateAddressFallbacks(address)

	// Try each address variation until we get results
	for idx, addrVariation := range addressVariations {
		coords, err := np.geocodeSingleAddress(ctx, addrVariation)
		if err == nil {
			// Success! Log which fallback level worked
			if idx == 0 {
				np.log.DebugContext(ctx, "Geocoded with full address", "address", addrVariation)
			} else {
				np.log.InfoContext(ctx, "Geocoded using fallback address",
					"original", address,
					"fallback", addrVariation,
					"fallback_level", idx)
			}
			return coords, nil
		}

		// If it's not an empty response error, return immediately (API error, invalid coords, etc.)
		if !errors.Is(err, ErrNominatimEmptyResponse) {
			return nil, err
		}

		// Empty response - try next fallback
		np.log.DebugContext(ctx, "Address variation returned no results, trying fallback",
			"variation", addrVariation,
			"fallback_level", idx)
	}

	// All fallbacks exhausted
	np.log.WarnContext(
		ctx,
		"All address fallbacks exhausted",
		"address",
		address,
		"variations_tried",
		len(addressVariations),
	)
	return nil, ErrNominatimEmptyResponse
}

// generateAddressFallbacks creates a list of progressively simpler address variations.
func (np *NominatimProvider) generateAddressFallbacks(address string) []string {
	if address == "" {
		return []string{""}
	}

	// Use a map to track unique variations and preserve order
	seen := make(map[string]bool)
	variations := []string{}

	// Helper to add variation if not seen
	addVariation := func(v string) {
		if v != "" && !seen[v] {
			seen[v] = true
			variations = append(variations, v)
		}
	}

	// Start with full address
	addVariation(address)

	// Split by comma to get address components
	parts := strings.Split(address, ",")

	// Trim whitespace from all parts
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	// If we have multiple parts, create fallbacks by removing from the end
	if len(parts) > 1 {
		// Remove last component (usually house number)
		addVariation(strings.Join(parts[:len(parts)-1], ", "))

		// If we have 3+ parts, try removing two components
		const lenComponents = 2
		if len(parts) > lenComponents {
			addVariation(strings.Join(parts[:len(parts)-2], ", "))
		}

		// Try just the first component (village/town/city)
		addVariation(parts[0])
	}

	return variations
}

// geocodeSingleAddress performs a single geocoding request without fallback logic.
func (np *NominatimProvider) geocodeSingleAddress(ctx context.Context, address string) (*models.Coordinates, error) {
	// Build request URL with query parameters
	reqURL, err := url.Parse(np.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	query := reqURL.Query()
	query.Set("q", address)
	query.Set("format", "json")
	query.Set("limit", "1")               // Only need the top result
	query.Set("addressdetails", "1")      // Include detailed address breakdown for better matching
	query.Set("accept-language", "uk,en") // Prefer Ukrainian, fallback to English
	reqURL.RawQuery = query.Encode()

	np.log.DebugContext(ctx, "Nominatim request URL", "url", reqURL.String())

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers per Nominatim usage policy
	req.Header.Set("User-Agent", np.userAgent)
	req.Header.Set("Accept-Language", "uk,en") // Prefer Ukrainian results

	// Execute request
	resp, err := np.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute geocoding request: %w", err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		np.log.ErrorContext(ctx, "Nominatim API error", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("nominatim API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log raw response for debugging
	np.log.DebugContext(ctx, "Nominatim raw response", "body", string(body))

	// Parse response
	var results []nominatimResponse
	if err = json.Unmarshal(body, &results); err != nil {
		np.log.ErrorContext(ctx, "Failed to parse Nominatim response", "error", err, "body", string(body))
		return nil, fmt.Errorf("failed to decode nominatim response: %w", err)
	}

	// Check if we got any results
	if len(results) == 0 {
		return nil, ErrNominatimEmptyResponse
	}

	np.log.DebugContext(ctx, "Nominatim found result", "lat", results[0].Lat, "lon", results[0].Lon)

	// Parse coordinates
	var lat, lon float64
	if _, err = fmt.Sscanf(results[0].Lat, "%f", &lat); err != nil {
		return nil, fmt.Errorf("%w: invalid latitude: %s", ErrNominatimInvalidCoords, results[0].Lat)
	}
	if _, err = fmt.Sscanf(results[0].Lon, "%f", &lon); err != nil {
		return nil, fmt.Errorf("%w: invalid longitude: %s", ErrNominatimInvalidCoords, results[0].Lon)
	}

	return &models.Coordinates{
		Latitude:  lat,
		Longitude: lon,
	}, nil
}
