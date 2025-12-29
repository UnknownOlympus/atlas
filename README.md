# Atlas - Geocoding Service

Atlas is a scalable geocoding microservice for the UnknownOlympus project. It scans database records containing tasks and performs geocoding for addresses that don't have geolocation data.

## Features

- **Multiple Geocoding Providers**: Support for Google Maps and OpenStreetMap (Nominatim)
- **Configuration-Driven**: Runtime provider selection via environment variables
- **Clean Architecture**: Dependency injection and provider abstraction
- **High Performance**: Concurrent worker pool for parallel geocoding
- **Observability**: Prometheus metrics and structured logging
- **Production-Ready**: Health checks, graceful shutdown, and comprehensive testing

## Supported Geocoding Providers

### Google Maps Geocoding API
- **Type**: `google`
- **Requirements**: API key (paid service)
- **Rate Limit**: Configurable per worker
- **Best For**: Production environments requiring high accuracy

### OpenStreetMap Nominatim
- **Type**: `nominatim`
- **Requirements**: None (free service)
- **Rate Limit**: 1 request/second (fair use policy)
- **Best For**: Development, testing, or low-volume production

## Configuration

Atlas is configured using environment variables:

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ATLAS_ENV` | Environment (local/development/production) | `production` | No |
| `ATLAS_PROVIDER_TYPE` | Geocoding provider (`google` or `nominatim`) | `google` | No |
| `ATLAS_PROVIDER_API_KEY` | API key for geocoding provider | - | Yes (for Google) |
| `ATLAS_WORKERS` | Number of concurrent workers | `10` | No |
| `ATLAS_INTERVAL` | Polling interval for new tasks | `10m` | No |
| `ATLAS_HEALTH_PORT` | Port for health/metrics endpoints | `8080` | No |
| `ATLAS_ADDRESS_PREFIX` | Prefix added to addresses for better accuracy | - | No |
| `DB_HOST` | PostgreSQL host | - | Yes |
| `DB_PORT` | PostgreSQL port | - | Yes |
| `DB_USERNAME` | PostgreSQL username | - | Yes |
| `DB_PASSWORD` | PostgreSQL password | - | Yes |
| `DB_NAME` | PostgreSQL database name | - | Yes |

### Example: Using Google Maps (Default)

```bash
export ATLAS_ENV=production
export ATLAS_PROVIDER_TYPE=google
export ATLAS_PROVIDER_API_KEY=your-google-api-key
export ATLAS_WORKERS=10
export ATLAS_INTERVAL=5m
export DB_HOST=localhost
export DB_PORT=5432
export DB_USERNAME=postgres
export DB_PASSWORD=secret
export DB_NAME=radioguru
```

### Example: Using Nominatim (Free)

```bash
export ATLAS_ENV=development
export ATLAS_PROVIDER_TYPE=nominatim
# No API key required for Nominatim
export ATLAS_WORKERS=5
export ATLAS_INTERVAL=10m
export DB_HOST=localhost
export DB_PORT=5432
export DB_USERNAME=postgres
export DB_PASSWORD=secret
export DB_NAME=radioguru
```

## Building and Running

### Build

```bash
go build -o atlas ./cmd/main.go
```

### Run

```bash
./atlas
```

### Run with Docker

```bash
docker build -t atlas:latest .
docker run -d \
  -e ATLAS_PROVIDER_TYPE=nominatim \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_USERNAME=postgres \
  -e DB_PASSWORD=secret \
  -e DB_NAME=radioguru \
  -p 8080:8080 \
  atlas:latest
```

## Testing

Run all tests:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test ./... -cover
```

Generate detailed coverage report:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Monitoring

### Health Check
```bash
curl http://localhost:8080/healthz
```

### Prometheus Metrics
```bash
curl http://localhost:8080/metrics
```

## Architecture

### Clean Architecture Principles

The service follows clean architecture with clear separation of concerns:

- **`internal/geocoding`**: Provider interface and implementations
  - `provider.go`: Provider interface definition
  - `google.go`: Google Maps provider implementation
  - `nominatim.go`: Nominatim provider implementation
  - `factory.go`: Provider factory for runtime selection

- **`internal/service`**: Business logic (provider-agnostic)
  - `geocoding.go`: Core geocoding service with worker pool

- **`internal/repository`**: Database access layer
- **`internal/config`**: Configuration management
- **`internal/metrics`**: Prometheus metrics
- **`cmd`**: Application entry point

### Adding a New Provider

To add a new geocoding provider:

1. Create a new file in `internal/geocoding/` (e.g., `newprovider.go`)
2. Implement the `Provider` interface:
   ```go
   type Provider interface {
       Geocode(ctx context.Context, address string) (*models.Coordinates, error)
   }
   ```
3. Add the provider type to `factory.go`:
   ```go
   const ProviderTypeNew ProviderType = "newprovider"
   ```
4. Update the factory's `NewProvider()` function to handle the new type
5. Write comprehensive unit tests
