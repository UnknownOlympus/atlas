package geocoding_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/UnknownOlympus/atlas/internal/geocoding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestVisicomProvider_Geocoe(t *testing.T) {
	ctx := t.Context()
	logger := slog.Default()
	apiKey := "test-api-key"
	defaultRL := rate.NewLimiter(rate.Inf, 0)

	t.Run("successfull geocoding", func(t *testing.T) {
		mockClient := &mockHTTPClient{
			doFunc: func(req *http.Request) (*http.Response, error) {
				// Verify request parameters
				assert.Equal(t, "GET", req.Method)
				assert.Contains(t, req.URL.String(), geocoding.VisicomBaseURL)
				assert.Equal(t, "1600 Amphitheatre Parkway, Mountain View, CA", req.URL.Query().Get("text"))
				assert.Equal(t, apiKey, req.URL.Query().Get("key"))
				assert.Equal(t, "1", req.URL.Query().Get("limit"))
				assert.Equal(t, "application/json", req.Header.Get("Accept"))

				// Return ,ock response
				responseBody := `{"geo_centroid":{"coordinates":[-122.0842499,37.4224764]}}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(responseBody)),
				}, nil
			},
		}

		provider := geocoding.NewVisicomProviderWithClient(mockClient, apiKey, defaultRL, logger)
		coords, err := provider.Geocode(ctx, "1600 Amphitheatre Parkway, Mountain View, CA")

		require.NoError(t, err)
		require.NotNil(t, coords)
		assert.InEpsilon(t, 37.4224764, coords.Latitude, 0.0001)
		assert.InEpsilon(t, -122.0842499, coords.Longitude, 0.0001)
	})

	t.Run("empty response", func(t *testing.T) {
		mockClient := &mockHTTPClient{
			doFunc: func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
				}, nil
			},
		}

		provider := geocoding.NewVisicomProviderWithClient(mockClient, apiKey, defaultRL, logger)
		coords, err := provider.Geocode(ctx, "some address")

		require.Error(t, err)
		assert.Nil(t, coords)
		assert.ErrorIs(t, err, geocoding.ErrVisicomEmptyResponse)
	})

	t.Run("invalid coordinates", func(t *testing.T) {
		mockClient := &mockHTTPClient{
			doFunc: func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"geo_centroid":{"coordinates":[30.5]}}`)),
				}, nil
			},
		}

		provider := geocoding.NewVisicomProviderWithClient(mockClient, apiKey, defaultRL, logger)
		coords, err := provider.Geocode(ctx, "bad coords")

		require.Error(t, err)
		assert.Nil(t, coords)
		assert.ErrorIs(t, err, geocoding.ErrVisicomInvalidCoords)
	})

	t.Run("Unathorized", func(t *testing.T) {
		mockClient := &mockHTTPClient{
			doFunc: func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(bytes.NewBufferString(`unathorized`)),
				}, nil
			},
		}

		provider := geocoding.NewVisicomProviderWithClient(mockClient, apiKey, defaultRL, logger)
		coords, err := provider.Geocode(ctx, "some address")

		require.Error(t, err)
		assert.Nil(t, coords)
		assert.ErrorIs(t, err, geocoding.ErrVisicomUnathorized)
	})

	t.Run("rate limit exceeded", func(t *testing.T) {
		rateCtx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately
		mockClient := &mockHTTPClient{
			doFunc: func(_ *http.Request) (*http.Response, error) {
				t.Fatal("HTTP client should not be called when rate limit blocks")
				return &http.Response{}, nil
			},
		}

		limiter := rate.NewLimiter(rate.Every(time.Second), 1)

		provider := geocoding.NewVisicomProviderWithClient(mockClient, apiKey, limiter, logger)
		coords, err := provider.Geocode(rateCtx, "some address")

		require.Error(t, err)
		assert.Nil(t, coords)
		assert.ErrorContains(t, err, "rate limit exceeded")
	})

	t.Run("empty address", func(t *testing.T) {
		mockClient := &mockHTTPClient{
			doFunc: func(_ *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
				}, nil
			},
		}

		provider := geocoding.NewVisicomProviderWithClient(mockClient, apiKey, defaultRL, logger)
		coords, err := provider.Geocode(ctx, "")

		require.Error(t, err)
		assert.Nil(t, coords)
		assert.ErrorIs(t, err, geocoding.ErrVisicomEmptyAddress)
	})
}
