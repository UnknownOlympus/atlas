package geocoding_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/UnknownOlympus/atlas/internal/geocoding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHTTPClient is a mock implementation of HTTPClient for testing.
type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func TestNominatimProvider_Geocode(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	t.Run("successful geocoding", func(t *testing.T) {
		mockClient := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				// Verify request parameters
				assert.Equal(t, "GET", req.Method)
				assert.Contains(t, req.URL.String(), "nominatim.openstreetmap.org")
				assert.Equal(t, "1600 Amphitheatre Parkway, Mountain View, CA", req.URL.Query().Get("q"))
				assert.Equal(t, "json", req.URL.Query().Get("format"))
				assert.Equal(t, "1", req.URL.Query().Get("limit"))
				assert.Equal(
					t,
					"Atlas-Geocoding-Service/1.0 (https://github.com/UnknownOlympus/atlas)",
					req.Header.Get("User-Agent"),
				)

				// Return mock response
				responseBody := `[{"lat":"37.4224764","lon":"-122.0842499"}]`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
				}, nil
			},
		}

		provider := geocoding.NewNominatimProviderWithClient(mockClient, logger)
		coords, err := provider.Geocode(ctx, "1600 Amphitheatre Parkway, Mountain View, CA")

		require.NoError(t, err)
		require.NotNil(t, coords)
		assert.InEpsilon(t, 37.4224764, coords.Latitude, 0.0001)
		assert.InEpsilon(t, -122.0842499, coords.Longitude, 0.0001)
	})

	t.Run("empty response from API", func(t *testing.T) {
		mockClient := &mockHTTPClient{
			doFunc: func(_ *http.Request) (*http.Response, error) {
				responseBody := `[]`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
				}, nil
			},
		}

		provider := geocoding.NewNominatimProviderWithClient(mockClient, logger)
		coords, err := provider.Geocode(ctx, "invalid address")

		require.Error(t, err)
		require.Nil(t, coords)
		assert.ErrorIs(t, err, geocoding.ErrNominatimEmptyResponse)
	})

	t.Run("HTTP error status", func(t *testing.T) {
		mockClient := &mockHTTPClient{
			doFunc: func(_ *http.Request) (*http.Response, error) {
				responseBody := `{"error":"Rate limit exceeded"}`
				return &http.Response{
					StatusCode: http.StatusTooManyRequests,
					Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
				}, nil
			},
		}

		provider := geocoding.NewNominatimProviderWithClient(mockClient, logger)
		coords, err := provider.Geocode(ctx, "some address")

		require.Error(t, err)
		require.Nil(t, coords)
		assert.Contains(t, err.Error(), "nominatim API returned status 429")
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		mockClient := &mockHTTPClient{
			doFunc: func(_ *http.Request) (*http.Response, error) {
				responseBody := `invalid json`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
				}, nil
			},
		}

		provider := geocoding.NewNominatimProviderWithClient(mockClient, logger)
		coords, err := provider.Geocode(ctx, "some address")

		require.Error(t, err)
		require.Nil(t, coords)
		assert.Contains(t, err.Error(), "failed to decode nominatim response")
	})

	t.Run("invalid latitude in response", func(t *testing.T) {
		mockClient := &mockHTTPClient{
			doFunc: func(_ *http.Request) (*http.Response, error) {
				responseBody := `[{"lat":"invalid","lon":"-122.0842499"}]`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
				}, nil
			},
		}

		provider := geocoding.NewNominatimProviderWithClient(mockClient, logger)
		coords, err := provider.Geocode(ctx, "some address")

		require.Error(t, err)
		require.Nil(t, coords)
		require.ErrorIs(t, err, geocoding.ErrNominatimInvalidCoords)
		assert.Contains(t, err.Error(), "invalid latitude")
	})

	t.Run("invalid longitude in response", func(t *testing.T) {
		mockClient := &mockHTTPClient{
			doFunc: func(_ *http.Request) (*http.Response, error) {
				responseBody := `[{"lat":"37.4224764","lon":"invalid"}]`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
				}, nil
			},
		}

		provider := geocoding.NewNominatimProviderWithClient(mockClient, logger)
		coords, err := provider.Geocode(ctx, "some address")

		require.Error(t, err)
		require.Nil(t, coords)
		require.ErrorIs(t, err, geocoding.ErrNominatimInvalidCoords)
		assert.Contains(t, err.Error(), "invalid longitude")
	})

	t.Run("HTTP client returns error", func(t *testing.T) {
		mockClient := &mockHTTPClient{
			doFunc: func(_ *http.Request) (*http.Response, error) {
				return nil, assert.AnError
			},
		}

		provider := geocoding.NewNominatimProviderWithClient(mockClient, logger)
		coords, err := provider.Geocode(ctx, "some address")

		require.Error(t, err)
		require.Nil(t, coords)
		assert.Contains(t, err.Error(), "failed to execute geocoding request")
	})

	t.Run("context cancellation", func(t *testing.T) {
		newCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		mockClient := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				return nil, req.Context().Err()
			},
		}

		provider := geocoding.NewNominatimProviderWithClient(mockClient, logger)
		coords, err := provider.Geocode(newCtx, "some address")

		require.Error(t, err)
		require.Nil(t, coords)
	})
}

func TestNominatimProvider_AddressFallback(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	t.Run("fallback to village name when full address fails", func(t *testing.T) {
		requestCount := 0
		mockClient := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				requestCount++
				query := req.URL.Query().Get("q")

				// First request (full address) returns empty
				if query == "с. Грабовець, вул. Польова, 3" {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(`[]`)),
					}, nil
				}

				// Second request (without house number) returns empty
				if query == "с. Грабовець, вул. Польова" {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(`[]`)),
					}, nil
				}

				// Third request (village only) succeeds
				if query == "с. Грабовець" {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(`[{"lat":"49.1234","lon":"24.5678"}]`)),
					}, nil
				}

				t.Fatalf("Unexpected query: %s", query)
				return nil, assert.AnError
			},
		}

		provider := geocoding.NewNominatimProviderWithClient(mockClient, logger)
		coords, err := provider.Geocode(ctx, "с. Грабовець, вул. Польова, 3")

		require.NoError(t, err)
		require.NotNil(t, coords)
		assert.InEpsilon(t, 49.1234, coords.Latitude, 0.0001)
		assert.InEpsilon(t, 24.5678, coords.Longitude, 0.0001)
		assert.Equal(t, 3, requestCount, "should try 3 fallback levels")
	})

	t.Run("success on first try with full address", func(t *testing.T) {
		requestCount := 0
		mockClient := &mockHTTPClient{
			doFunc: func(_ *http.Request) (*http.Response, error) {
				requestCount++
				// Full address succeeds immediately
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`[{"lat":"50.4501","lon":"30.5234"}]`)),
				}, nil
			},
		}

		provider := geocoding.NewNominatimProviderWithClient(mockClient, logger)
		coords, err := provider.Geocode(ctx, "м. Київ, вул. Хрещатик, 1")

		require.NoError(t, err)
		require.NotNil(t, coords)
		assert.Equal(t, 1, requestCount, "should succeed on first try")
	})

	t.Run("all fallbacks fail", func(t *testing.T) {
		mockClient := &mockHTTPClient{
			doFunc: func(_ *http.Request) (*http.Response, error) {
				// All requests return empty
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`[]`)),
				}, nil
			},
		}

		provider := geocoding.NewNominatimProviderWithClient(mockClient, logger)
		coords, err := provider.Geocode(ctx, "с. Невідоме, вул. Невідома, 999")

		require.Error(t, err)
		require.Nil(t, coords)
		assert.ErrorIs(t, err, geocoding.ErrNominatimEmptyResponse)
	})

	t.Run("single-part address no fallback", func(t *testing.T) {
		requestCount := 0
		mockClient := &mockHTTPClient{
			doFunc: func(_ *http.Request) (*http.Response, error) {
				requestCount++
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`[{"lat":"48.9226","lon":"24.7111"}]`)),
				}, nil
			},
		}

		provider := geocoding.NewNominatimProviderWithClient(mockClient, logger)
		coords, err := provider.Geocode(ctx, "Івано-Франківськ")

		require.NoError(t, err)
		require.NotNil(t, coords)
		assert.Equal(t, 1, requestCount, "single-part address should only try once")
	})
}

func TestNewNominatimProvider(t *testing.T) {
	logger := slog.Default()

	provider := geocoding.NewNominatimProvider(logger)

	require.NotNil(t, provider)
}
