# Provisioning Service

A predictive node provisioning service that ensures GPU nodes are ready before users request them, minimizing wait time while controlling costs.

## Architecture

The service follows clean architecture principles with clear separation between domain and infrastructure:

### Domain Layer (`internal/domain`)
Contains core business logic with no external dependencies:
- **Models**: `Node`, `NodePool`, `UserState`, `UserTracker` - Pure Go structs and domain logic
- **Services**:
  - `Predictor` - Implements the predictive scaling algorithm
  - `NodeAllocator` - Handles node allocation to users
- **Events**: Event definitions for Redis pub/sub channels

### Infrastructure Layer (`internal/infra`)
Contains implementations that interact with external systems:
- **Config** (`internal/infra/config`) - Configuration management using Koanf
- **HTTP** (`internal/infra/http`) - Fiber v3 HTTP server for health checks and metrics
- **Redis** (`internal/infra/redis`) - Redis client and pub/sub subscriber
- **Node API** (`internal/infra/nodeapi`) - HTTP client for Node Management API

### Service Layer (`internal/service`)
Contains the `Provisioner` orchestrator that ties everything together.

### Application Layer (`internal/app`)
Contains dependency injection wiring using uber.go/fx.

## Predictive Algorithm

The service uses a heuristic-based predictive algorithm:

### Key Metrics
- **Activity Window**: 2 minutes - Time window to track user activity
- **Activity Threshold**: 3 - Number of activities needed to predict connection
- **Min Ready Nodes**: 1 - Minimum nodes kept ready
- **Max Ready Nodes**: 5 - Maximum nodes to prevent runaway costs
- **Idle Termination**: 5 minutes - Time before idle nodes are terminated
- **Scaling Check Interval**: 10 seconds - How often to evaluate scaling decisions

### Scaling Logic

**Scale Up When:**
1. Predicted demand (users with >= 3 activities in 2 min) exceeds available capacity (ready + booting nodes)
2. Ready nodes fall below minimum threshold

**Scale Down When:**
1. Ready nodes have been idle for > 5 minutes
2. No predicted demand exists
3. Ensures we never go below minimum ready nodes

**Emergency Provisioning:**
- If a user connects and no ready node exists, immediately provision a new node
- Logs as CRITICAL event for monitoring

### Trade-offs

**Cost vs. Latency:**
- Maintains minimum 1 ready node at all times (small baseline cost)
- Caps at 5 nodes maximum to prevent cost explosion
- 5-minute idle timeout balances cost savings with readiness

**Prediction Accuracy:**
- Simple activity-count heuristic is easy to understand and tune
- May over-provision during browsing-heavy periods
- May under-provision during sudden traffic spikes

**Reliability:**
- Handles stuck booting nodes (2-minute timeout)
- Concurrent-safe with mutex-protected data structures
- Graceful shutdown with proper lifecycle management

## Configuration

Environment variables (prefix with `APP_`):

```bash
# Server
APP_SERVER_PORT=8081

# Redis
APP_REDIS_ADDR=localhost:6379
APP_REDIS_PASSWORD=
APP_REDIS_DB=0

# Node Management API
APP_NODE_API_BASE_URL=http://localhost:8080
APP_NODE_API_TIMEOUT=10s

# Prediction Algorithm
APP_PREDICTION_ACTIVITY_WINDOW=2m
APP_PREDICTION_ACTIVITY_THRESHOLD=3
APP_PREDICTION_MIN_READY_NODES=1
APP_PREDICTION_MAX_READY_NODES=5
APP_PREDICTION_IDLE_TERMINATION_TIMEOUT=5m
APP_PREDICTION_BOOTING_NODE_TIMEOUT=2m
APP_PREDICTION_SCALING_CHECK_INTERVAL=10s
```

## Building and Running

### Local Development

```bash
# Build
go build -o provisioning-service ./cmd/server

# Run
./provisioning-service
```

### Docker

```bash
# Build
docker build -t provisioning-service .

# Run
docker run -e APP_REDIS_ADDR=redis:6379 -e APP_NODE_API_BASE_URL=http://node-api:8080 provisioning-service
```

### Docker Compose

```bash
# Start all services
docker-compose up

# Start provisioning service only
docker-compose up provisioning-service
```

## API Endpoints

- `GET /health` - Health check endpoint
- `GET /metrics` - Node and user metrics (JSON)
- `GET /status` - Detailed status of all nodes and users (JSON)

## Monitoring

The service logs important events:
- **INFO**: Normal operations (scaling, allocation, deallocation)
- **WARN**: Stuck booting nodes
- **ERROR**: Failed operations (provision, terminate, allocation failures)
- **CRITICAL**: No ready node available for connecting user (major service failure)

## What I Would Improve With More Time

1. **Smarter Prediction**:
   - Machine learning model trained on historical activity patterns
   - Time-of-day awareness (peak hours vs. off-hours)
   - Per-user connection probability scoring
   - Exponential smoothing for activity rates

2. **Better State Management**:
   - Persist state to Redis for multi-instance deployments
   - Leader election for active-passive setup
   - State snapshots for faster recovery

3. **Advanced Metrics**:
   - Prometheus metrics export
   - Connection miss rate tracking
   - Cost optimization metrics (idle time, utilization)
   - Prediction accuracy metrics

4. **Testing**:
   - Unit tests for all domain logic
   - Integration tests with Redis and Node API
   - Load testing with realistic traffic patterns
   - Chaos testing for failure scenarios

5. **Operational Improvements**:
   - Graceful handling of Redis connection loss
   - Circuit breaker for Node API calls
   - Rate limiting for provisioning operations
   - Dynamic configuration updates without restart

6. **User Prioritization**:
   - Priority queues for different user tiers
   - Reserve capacity for premium users
   - SLA-aware provisioning

7. **Advanced Scaling**:
   - Multi-region support with geo-awareness
   - Different node types/sizes based on workload
   - Predictive pre-warming based on scheduled events
