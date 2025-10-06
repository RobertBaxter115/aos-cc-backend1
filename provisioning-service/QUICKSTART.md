# Quick Start Guide

## Prerequisites

- Docker and Docker Compose
- Go 1.23+ (for local development)

## Running the Service

### With Docker Compose (Recommended)

From the project root:

```bash
# Start all services
docker-compose up --build

# Or run in background
docker-compose up --build -d

# View logs
docker-compose logs -f provisioning-service

# Stop all services
docker-compose down
```

This will start:

- Redis (port 6379)
- Node API (port 8080)
- Provisioning Service (port 8081)
- User Simulator (generates fake traffic)

### Local Development

```bash
cd provisioning-service

# Install dependencies
go mod download

# Run the service
go run ./cmd/server/main.go
```

Make sure Redis and Node API are running separately.

## Testing the Service

### Check Health

```bash
curl http://localhost:8081/health
```

### View Metrics

```bash
curl http://localhost:8081/metrics | jq
```

### View Detailed Status

```bash
curl http://localhost:8081/status | jq
```

### Watch Logs

The service logs all important events:

- User activity tracking
- Scaling decisions (up/down)
- Node provisioning/termination
- User connections/disconnections
- Node status updates

```bash
# If running with docker-compose
docker-compose logs -f provisioning-service

# Look for log messages like:
# - "scaling up nodes" - Predictive algorithm decided to add nodes
# - "node allocated to user" - User got a ready node (SUCCESS!)
# - "CRITICAL: no ready node available" - Connection miss (FAILURE!)
# - "terminating idle node" - Cleaning up unused capacity
```

## Configuration

Environment variables can be set in docker-compose.yml or exported locally:

```bash
export APP_REDIS_ADDR=localhost:6379
export APP_NODE_API_BASE_URL=http://localhost:8080
export APP_PREDICTION_MIN_READY_NODES=2  # Keep 2 nodes ready at minimum
export APP_PREDICTION_MAX_READY_NODES=10 # Allow up to 10 nodes
```

## Monitoring Success

Watch for these indicators of good performance:

✅ **Good Signs:**

- "node allocated to user" messages (users getting nodes immediately)
- Steady number of ready nodes in /metrics
- Low "booting" node count (nodes transition quickly to ready)

❌ **Warning Signs:**

- "CRITICAL: no ready node available" (connection misses)
- High "booting" node count (provisioning too slow)
- Frequent "terminating idle node" right before "scaling up" (thrashing)

## Adjusting the Algorithm

If you see connection misses, try:

1. Increase `MIN_READY_NODES` (more baseline capacity)
2. Decrease `ACTIVITY_THRESHOLD` (more aggressive predictions)
3. Increase `IDLE_TERMINATION_TIMEOUT` (keep nodes ready longer)

If costs are too high:

1. Decrease `MIN_READY_NODES` (less baseline capacity)
2. Increase `ACTIVITY_THRESHOLD` (less aggressive predictions)
3. Decrease `IDLE_TERMINATION_TIMEOUT` (clean up faster)
