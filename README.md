# HTTP Tracing Service

A lightweight HTTP service that provides tracing, debugging, and health check capabilities.

## Features

- Request tracing with customizable header prefixes
- Request debugging with detailed request information
- Health check endpoint
- TLS support

## Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | `8443` (TLS) or `8080` (non-TLS) | No |
| `HOST` | Server host address | `""` | No |
| `ENABLE_TLS` | Enable TLS support | `false` | No |
| `TARGET_HOST` | Host where traced requests will be forwarded | Current host | No |
| `TRACE_PREFIX` | Prefix for tracing headers to be filtered | `X-B3` | No |
| `TRACE_ROUTE` | Base path for tracing endpoint | `/init` | No |

## API Endpoints

### Health Check
```
GET/POST /health
```
Returns `200 OK` if the service is healthy, `500 Internal Server Error` if not.

### Debug
```
GET/POST /debug
```
Returns detailed information about the incoming request, including:
- Headers
- Query Parameters
- Request Body

Response format: JSON

### Trace
```
ANY /trace/*
```
Forwards the request to the target host while:
- Stripping tracing headers (configurable via `TRACE_PREFIX`)
- Adding unique trace ID (`x-client-trace-id`)
- Forcing trace collection (`x-envoy-force-trace`)

## Building and Running

### Local Development
```bash
# Build the binary
make build

# Run the service
make run

# Run the tests
make test.fast

# Format the code
make fmt

# Clean up
make clean
```

### Docker
The following command will build the Docker images for amd64 and arm64 and push it to the repository.
```bash
# Build the Docker image
make build.image
```

### Docker Environment Variables
```bash
DOCKER_REPO=your-repo/name  # Default: datawire/tracer
TAG=your-tag               # Default: latest
```

## TLS Configuration

When TLS is enabled (`ENABLE_TLS=true`), the service expects the following files:
- `/certs/cert.pem`: TLS certificate
- `/certs/key.pem`: TLS private key