package geocoding_test

import (
	"log/slog"
	"testing"

	"github.com/UnknownOlympus/atlas/internal/geocoding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	logger := slog.Default()

	t.Run("create Google provider successfully", func(t *testing.T) {
		config := geocoding.ProviderConfig{
			Type:      geocoding.ProviderTypeGoogle,
			APIKey:    "test-api-key",
			RateLimit: 10,
			Logger:    logger,
		}

		provider, err := geocoding.NewProvider(config)

		require.NoError(t, err)
		require.NotNil(t, provider)
		// Verify it's a GoogleProvider by type assertion
		_, ok := provider.(*geocoding.GoogleProvider)
		assert.True(t, ok, "expected provider to be *GoogleProvider")
	})

	t.Run("create Google provider without API key fails", func(t *testing.T) {
		config := geocoding.ProviderConfig{
			Type:      geocoding.ProviderTypeGoogle,
			APIKey:    "", // Empty API key
			RateLimit: 10,
			Logger:    logger,
		}

		provider, err := geocoding.NewProvider(config)

		require.Error(t, err)
		require.Nil(t, provider)
		assert.Contains(t, err.Error(), "API key is required for Google provider")
	})

	t.Run("create Google provider with rate limit", func(t *testing.T) {
		config := geocoding.ProviderConfig{
			Type:      geocoding.ProviderTypeGoogle,
			APIKey:    "test-api-key",
			RateLimit: 50,
			Logger:    logger,
		}

		provider, err := geocoding.NewProvider(config)

		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("create Google provider without rate limit", func(t *testing.T) {
		config := geocoding.ProviderConfig{
			Type:      geocoding.ProviderTypeGoogle,
			APIKey:    "test-api-key",
			RateLimit: 0, // No rate limit
			Logger:    logger,
		}

		provider, err := geocoding.NewProvider(config)

		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("create Nominatim provider successfully", func(t *testing.T) {
		config := geocoding.ProviderConfig{
			Type:   geocoding.ProviderTypeNominatim,
			Logger: logger,
		}

		provider, err := geocoding.NewProvider(config)

		require.NoError(t, err)
		require.NotNil(t, provider)
		// Verify it's a NominatimProvider by type assertion
		_, ok := provider.(*geocoding.NominatimProvider)
		assert.True(t, ok, "expected provider to be *NominatimProvider")
	})

	t.Run("create Nominatim provider without API key", func(t *testing.T) {
		// Nominatim doesn't require an API key
		config := geocoding.ProviderConfig{
			Type:   geocoding.ProviderTypeNominatim,
			APIKey: "", // No API key needed
			Logger: logger,
		}

		provider, err := geocoding.NewProvider(config)

		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("unsupported provider type", func(t *testing.T) {
		config := geocoding.ProviderConfig{
			Type:   geocoding.ProviderType("unsupported"),
			Logger: logger,
		}

		provider, err := geocoding.NewProvider(config)

		require.Error(t, err)
		require.Nil(t, provider)
		assert.Contains(t, err.Error(), "unsupported provider type: unsupported")
	})

	t.Run("empty provider type", func(t *testing.T) {
		config := geocoding.ProviderConfig{
			Type:   geocoding.ProviderType(""),
			Logger: logger,
		}

		provider, err := geocoding.NewProvider(config)

		require.Error(t, err)
		require.Nil(t, provider)
		assert.Contains(t, err.Error(), "unsupported provider type")
	})
}

func TestProviderType_Constants(t *testing.T) {
	// Verify that provider type constants are correctly defined
	assert.Equal(t, "google", string(geocoding.ProviderTypeGoogle))
	assert.Equal(t, "nominatim", string(geocoding.ProviderTypeNominatim))
}
