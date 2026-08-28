# MinURL - High-Performance URL Shortener with Analytics

A production-ready URL shortening service built with Go, featuring
real-time analytics, distributed caching, and stream processing.
Designed for high throughput and low latency with a focus on scalability
and reliability.

## Features

- **URL Shortening**: Generate unique, short URLs using distributed
  Snowflake IDs
- **Real-time Analytics**: Track clicks, referrers, and user agents with
  ClickHouse
- **Distributed Caching**: Multi-layer caching with Redis and Bloom
  filters for 99%+ cache hit rate
- **Stream Processing**: NATS JetStream for async database writes and
  event processing
- **High Availability**: Batch processing with automatic retries and
  error handling
- **Data Deduplication**: Smart URL deduplication to prevent duplicate
  short URLs
- **API-First Design**: RESTful API with comprehensive validation and
  error handling
- **Containerized**: Docker Compose setup for local development and
  testing

## Architecture

### Key Components

- **API Server**: Chi router with middleware for panic recovery and
  error handling
- **Redis Stack**: Multi-layer caching with Bloom filters for fast
  duplicate detection
- **PostgreSQL**: Primary data store with connection pooling and batch
  inserts
- **ClickHouse**: Columnar database for high-performance analytics
  queries
- **NATS JetStream**: Message queue for reliable async database
  operations
- **Snowflake IDs**: Distributed unique ID generation for URL slugs

## Tech Stack

- **Language**: Go 1.26
- **Web Framework**: Chi v5
- **Databases**:
  - PostgreSQL 18 (primary storage)
  - ClickHouse (analytics)
  - Redis Stack (caching + Bloom filters)
- **Message Queue**: NATS with JetStream
- **Containerization**: Docker & Docker Compose
- **Migrations**: Goose
- **Logging**: structured logging with slog

## Prerequisites

- Docker & Docker Compose
- Go 1.26+ (for local development)
- Make or Just (optional, for build commands)

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/ukiran03/minurl
cd minurl
# Start all services
docker compose up -d
# Check service status
docker compose ps
# View logs
docker compose logs -f app
```

The API will be available at `http://localhost:4000`

### Local Development

```bash
# Install dependencies
go mod download
# Set environment variables
export POSTGRES_DSN="postgres://user:pass@localhost:5432/minurl?sslmode=disable"
export REDIS_ADDR="localhost:6379"
export NATS_URL="nats://localhost:4222"
export CLICKHOUSE_HOST="localhost"
export CLICKHOUSE_DB="minurl"
export CLICKHOUSE_USER="default"
export SNOWFLAKE_NODE_ID="1"
# Run the server
go run cmd/api/main.go
```

## API Documentation

### Create Short URL

```mermaid
flowchart TD
    Client((Client)) -->|"POST /v1/shorten"| API
    API -->|Check URL| Bloom{{Bloom Filter}}
    Bloom -->|Exists| Redis[(Redis Cache)]
    Bloom -->|Not Exists| Create[Create New URL]
    Redis -->|Cache Hit| Return([Return Existing URL])
    Redis -->|Cache Miss| DB[(PostgreSQL)]
    DB -->|Found,<br>Cache Healing| Redis
    DB -->|Not Found| Create
    Create -->|Write to Redis| Redis
    Create -->|Publish| NATS[[NATS JetStream]] --> PGWorker[Postgres Stream Worker] --> DB
    Create -->|Add URL| Bloom
```

```bash
POST /v1/shorten
Content-Type: application/json
{
  "url": "https://example.com",
  "expires_at": "1w"  // optional: 1d, 1w, 1m, 1y
}
```

**Response:**

```json
{
  "short_url": "http://localhost:4000/2uD3f3GldJe",
  "url": "https://example.com/"
}
```

### Redirect to Original URL

```mermaid
flowchart TD
    Client@{ shape: circle, label: "Client" } -->|"GET /{slug}"| API2[API Server]
    API2 -->|Check Slug| Redis2[(Redis Cache)]
    Redis2 -->|Cache Hit| Redirect1@{ shape: rounded, label: "HTTP 302 Redirect" }
    Redis2 -->|Cache Miss| PGCheck2[(PostgreSQL)]
    PGCheck2 -->|Found| Redirect1
    PGCheck2 -->|Not Found| NotFound@{ shape: rounded, label: "Not Found" }
    Redirect1 -->|Publish Click Event| NATS@{ shape: das, label: "NATS JetStream" }
    NATS -->|ch.clicks.event| CHWorker@{ shape: subproc, label: "ClickHouse Stream Worker" }
    CHWorker -->|Batch Write| CH[(ClickHouse)]
```

```bash
GET /{slug}
```

**Response:** 302 redirect to original URL

### Get URL Analytics

```mermaid
flowchart LR
        Client@{ shape: circle, label: "Client" } -->|"GET Analytics"| API3[API Server]
        API3 -->|"Query Analytics"| CH[(ClickHouse)]
        CH -->|Return Stats| API3
        API3 -->|JSON Response| Client
```

```bash
GET /v1/minurls/{slug}?from=2026-08-28T07:00:00Z&to=2026-08-28T08:00:00Z&limit=100
```

**Response:**

```json
{
  "total_clicks": 42,
  "top_referrers": [
    {"referrer": "Direct", "clicks": 30},
    {"referrer": "https://google.com", "clicks": 12}
  ]
}
```

### Delete Short URL

```mermaid
flowchart TD
        Client@{ shape: circle, label: "Client" } -->|DELETE| API[API Server]
        API -->|Delete from Redis| Redis3[(Redis Cache)]
        API -->|Delete from PostgreSQL| PG2[(PostgreSQL)]
        Redis3 -->|Success| API
        API --> DeleteResponse
        PG2 -->|Success| DeleteResponse@{ shape: rounded, label: "Delete Success Response" }
```

```bash
DELETE /v1/minurls/{slug}
```

**Response:**

```json
{
  "message": "minurl deleted successfully"
}
```

### Health Check

```bash
GET /v1/healthcheck
```

**Response:**

```json
{
  "status": "available",
  "system_info": {
    "environment": "development",
    "version": "1.0.0"
  }
}
```

## Configuration

Configuration is handled via environment variables and command-line
flags:

| Variable            | Description                  | Default               |
|---------------------|------------------------------|-----------------------|
| `PORT`              | API server port              | 4000                  |
| `POSTGRES_DSN`      | PostgreSQL connection string | -                     |
| `REDIS_ADDR`        | Redis address                | -                     |
| `NATS_URL`          | NATS connection URL          | -                     |
| `CLICKHOUSE_HOST`   | ClickHouse host              | -                     |
| `CLICKHOUSE_DB`     | ClickHouse database name     | -                     |
| `SNOWFLAKE_NODE_ID` | Snowflake node ID (0-1023)   | -                     |
| `BASE_URL`          | Base URL for short links     | http://localhost:4000 |

## Performance Optimizations

1.  **Multi-layer Caching**: Redis cache with Bloom filter pre-check for
    99%+ cache hit rate
2.  **Batch Processing**: NATS JetStream batches database writes for
    optimal throughput
3.  **Connection Pooling**: Configured connection pools for all database
    connections
4.  **Query Optimization**: Split analytics queries to skip expensive
    operations when possible
5.  **Async Operations**: Non-blocking analytics publishing to prevent
    redirect latency
6.  **Snowflake IDs**: Distributed ID generation eliminates database
    contention

## Testing

```bash
# Run unit tests
go test ./...
# Run with race detector
go test -race ./...
# Run specific package tests
go test ./internal/flake
```

## Monitoring & Logging

- **Structured Logging**: JSON-formatted logs with contextual
  information
- **ClickHouse Debug Logging**: Detailed query execution logs for
  performance analysis
- **Health Checks**: Endpoint for container orchestration and monitoring
- **Error Tracking**: Comprehensive error logging with request context

## Security Features

- **Input Validation**: Comprehensive URL validation and sanitization
- **SSRF Protection**: Blocks internal IP addresses and localhost
- **Scheme Enforcement**: Only allows HTTP/HTTPS protocols
- **Rate Limiting Ready**: Architecture supports rate limiting
  middleware
- **Secure Headers**: Configurable security headers

## Known Issues & Future Improvements

### Current Issues

- Health check doesn't verify database connectivity (shallow check)
- Cache/stream inconsistency edge cases when NATS publish fails after
  cache write

### Planned Enhancements

- [ ] Deep health checks with database connectivity verification
- [ ] Compensation transactions for failed stream publishes
- [ ] Kubernetes deployment manifests
- [ ] Metrics/Prometheus integration
- [ ] Rate limiting middleware
- [ ] Custom slug support
- [ ] ClickHouse background job for expired record cleanup
- [ ] Simple WebUI (templ, htmx)
- [ ] Webhook notifications for analytics events

**Note**: This project was designed to showcase modern Go backend
development practices including clean architecture, distributed systems
concepts, and performance optimization techniques.
