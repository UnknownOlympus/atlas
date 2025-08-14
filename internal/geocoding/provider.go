package geocoding

import (
	"context"

	"github.com/UnknownOlympus/atlas/internal/models"
)

// Provider is an interface that defines a method for geocoding an address.
// The Geocode method takes a context and an address string as input,
// and returns the corresponding coordinates and an error if any occurs.
type Provider interface {
	Geocode(ctx context.Context, address string) (*models.Coordinates, error)
}
