# HTTP Tracing Service

A lightweight HTTP service that provides tracing, debugging, and health check capabilities.

## Features

- Request tracing with customizable header prefixes
- Request debugging with detailed request information
- Health check endpoint
- TLS support
- Graceful shutdown on SIGTERM

## Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | Server port | `8443` (TLS) or `8080` (non-TLS) | No |
| `HOST` | Server host address | `""` | No |
| `ENABLE_TLS` | Enable TLS support | `true` | No |
| `TARGET_HOST` | Host where traced requests will be forwarded | Current host | No |
| `TRACE_PREFIX` | Prefix for tracing headers to be filtered | `X-B3` | No |
| `TRACE_ROUTE` | Base path for tracing endpoint | `/trace` | No |

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
```

### Docker
```bash
# Build the Docker image
make build.image

# Push the Docker image
make image.push
```

### Docker Environment Variables
```bash
DOCKER_REPO=your-repo/name  # Default: datawire.io/tracer
TAG=your-tag               # Default: latest
```

## TLS Configuration

When TLS is enabled (`ENABLE_TLS=true`), the service expects the following files:
- `/certs/cert.pem`: TLS certificate
- `/certs/key.pem`: TLS private key

## Graceful Shutdown

The service implements graceful shutdown by:
1. Listening for SIGTERM signals
2. Marking the service as unhealthy (failing health checks)
3. Waiting for the container orchestrator to stop sending traffic
4. Terminating the service
