package geocoding

import (
	"context"

	"github.com/Houeta/geocoding-service/internal/models"
)

type Provider interface {
	Geocode(ctx context.Context, address string) (*models.Coordinates, error)
}
