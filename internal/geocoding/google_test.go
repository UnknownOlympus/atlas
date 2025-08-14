package geocoding_test

import (
	"log/slog"
	"testing"

	"github.com/UnknownOlympus/atlas/internal/geocoding"
	"github.com/UnknownOlympus/atlas/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"googlemaps.github.io/maps"
)

func TestGeocode(t *testing.T) {
	mockClient := mocks.NewGoogleAPIClient(t)
	provider := geocoding.NewGoogleProvider(mockClient, slog.Default())
	ctx := t.Context()

	t.Run("api returns error", func(t *testing.T) {
		address := "some invalid place"
		req := &maps.GeocodingRequest{Address: address}

		mockClient.On("Geocode", ctx, req).Return(nil, assert.AnError).Once()

		_, err := provider.Geocode(ctx, address)

		require.Error(t, err)
		require.ErrorIs(t, err, assert.AnError)
		mockClient.AssertExpectations(t)
	})

	t.Run("api return empty response", func(t *testing.T) {
		address := "some invalid place"
		req := &maps.GeocodingRequest{Address: address}

		mockClient.On("Geocode", ctx, req).Return(nil, nil).Once()

		coords, err := provider.Geocode(ctx, address)

		require.Nil(t, coords)
		require.ErrorIs(t, err, geocoding.ErrEmptyResponse)
		mockClient.AssertExpectations(t)
	})

	t.Run("successfull geocoding", func(t *testing.T) {
		address := "1600 Amphitheatre Parkway, Mountain View, CA"
		req := &maps.GeocodingRequest{Address: address}
		mockReponse := []maps.GeocodingResult{
			{Geometry: maps.AddressGeometry{Location: maps.LatLng{Lat: 37.42, Lng: -122.08}}},
		}

		mockClient.On("Geocode", ctx, req).Return(mockReponse, nil).Once()

		coords, err := provider.Geocode(ctx, address)

		require.NoError(t, err)
		require.NotNil(t, coords)
		require.InEpsilon(t, 37.42, coords.Latitude, 0.01)
		require.InEpsilon(t, -122.08, coords.Longitude, 0.01)
		mockClient.AssertExpectations(t)
	})
}
