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
	"time"

	"github.com/UnknownOlympus/atlas/internal/models"
	"golang.org/x/time/rate"
)

// VisicomBaseURL -- Visicom API base URL.
const VisicomBaseURL = "https://api.visicom.ua/data-api/5.0/uk/geocode.json"

// VisicomProvider implements geocoding using Visicom API.
type VisicomProvider struct {
	client  HTTPClient    // HTTP client for making requests
	baseURL string        // Base URL for the Visicom API
	apiKey  string        // API key with geocoding access
	log     *slog.Logger  // Logger for logging operations
	limiter *rate.Limiter // Rate limiter
}

// Common errors for Visicom provider.
var (
	ErrVisicomEmptyResponse = errors.New("visicom API returned empty response")
	ErrVisicomEmptyAddress  = errors.New("visicom provider got empty address")
	ErrVisicomInvalidCoords = errors.New("visicom API returned invalid coordinates")
	ErrVisicomUnathorized   = errors.New("visicom API unathorized (invalid API key)")
)

// Visicom API response (simplified for geocoding use-case).
type visicomResponse struct {
	Geometry struct {
		Coordinates []float64 `json:"coordinates"` // [lon, lat]
	} `json:"geo_centroid"`
}

// NewVisicomProvider creates a new Visicom geocoding provider.
func NewVisicomProvider(apiKey string, rateLimit int, log *slog.Logger) *VisicomProvider {
	const timeout = 10

	return &VisicomProvider{
		client: &http.Client{
			Timeout: timeout * time.Second,
		},
		baseURL: VisicomBaseURL,
		apiKey:  apiKey,
		log:     log,
		limiter: rate.NewLimiter(rate.Limit(rateLimit), rateLimit),
	}
}

// NewVisicomProviderWithClient allows injecting custom HTTP client.
func NewVisicomProviderWithClient(
	client HTTPClient,
	apiKey string,
	limiter *rate.Limiter,
	log *slog.Logger,
) *VisicomProvider {
	return &VisicomProvider{
		client:  client,
		baseURL: VisicomBaseURL,
		apiKey:  apiKey,
		log:     log,
		limiter: limiter,
	}
}

// Geocode converts address into geographic coordinates using Visicom API.
func (vp *VisicomProvider) Geocode(
	ctx context.Context,
	address string,
) (*models.Coordinates, error) {
	const coordsListLength = 2

	// Rate limit
	if err := vp.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit exceeded: %w", err)
	}

	vp.log.DebugContext(ctx, "Geocoding using Visicom", "address", address)

	if address == "" {
		return nil, ErrVisicomEmptyAddress
	}

	reqURL, err := url.Parse(vp.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	query := reqURL.Query()
	query.Set("text", address)
	query.Set("limit", "1")
	query.Set("key", vp.apiKey)
	reqURL.RawQuery = query.Encode()

	vp.log.DebugContext(ctx, "Visicom request URL", "url", reqURL.String())

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		reqURL.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Headers
	req.Header.Set("Accept", "application/json")

	resp, err := vp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute geocoding request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrVisicomUnathorized
	default:
		body, _ := io.ReadAll(resp.Body)
		vp.log.ErrorContext(ctx, "Visicom API error", "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("visicom API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	vp.log.DebugContext(ctx, "Visicom raw response", "body", string(body))

	var result visicomResponse
	if err = json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode visicom response: %w", err)
	}

	coords := result.Geometry.Coordinates
	if len(coords) == 0 {
		return nil, ErrVisicomEmptyResponse
	}

	if len(coords) != coordsListLength {
		return nil, ErrVisicomInvalidCoords
	}

	lon := coords[0]
	lat := coords[1]

	vp.log.InfoContext(ctx, "Visicom found result", "address", address, "lat", lat, "lon", lon)

	return &models.Coordinates{
		Latitude:  lat,
		Longitude: lon,
	}, nil
}
