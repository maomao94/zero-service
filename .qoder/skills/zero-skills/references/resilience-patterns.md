# Resilience Patterns

go-zero implements defense-in-depth with multiple protection layers for production stability.

## Architecture Overview

```
Request → Load Shedding → Rate Limiting → Circuit Breaker → Timeout → Service
```

All resilience features are **enabled by default** and work automatically. They can be configured but not disabled in production mode.

## Circuit Breaker Pattern

### Overview

go-zero uses Google SRE circuit breaker algorithm that tracks success/failure ratios and opens when error threshold is exceeded.

### ✅ Automatic Circuit Breaker

Circuit breaker is **automatic** for:
- All RPC calls
- All database operations (via sqlx)
- All Redis operations
- HTTP client calls

**No configuration needed** - works out of the box:

```go
// Circuit breaker automatically protects this
user, err := l.svcCtx.UsersModel.FindOne(l.ctx, id)
if err != nil {
    // May return "service unavailable" if circuit is open
    return nil, err
}
```

### ✅ Manual Circuit Breaker

For custom operations, use circuit breaker explicitly:

```go
import "github.com/zeromicro/go-zero/core/breaker"

// Create breaker with name
brk := breaker.NewBreaker()

// Wrap operation
err := brk.DoWithAcceptable(func() error {
    // Your potentially failing operation
    return callExternalAPI()
}, func(err error) bool {
    // Return true if error should be accepted (not count as failure)
    // Return false if error should trip breaker
    return err == ErrTemporary  // Accept temporary errors
})

if err != nil {
    // Handle error - may be ErrServiceUnavailable if circuit is open
    return err
}
```

### ✅ Circuit Breaker States

```go
// CLOSED (normal): Requests pass through, failures tracked
// OPEN (protecting): All requests fail fast, no backend calls
// HALF-OPEN (testing): Limited requests allowed to test recovery

// Circuit opens when:
// - Error rate > threshold (default: 50%)
// - Minimum requests met (default: 10)

// Circuit closes when:
// - Success rate returns to normal in half-open state
```

### ✅ Custom Breaker Options

```go
import "github.com/zeromicro/go-zero/core/breaker"

// Create breaker with custom name for monitoring
brk := breaker.NewBreaker(
    breaker.WithName("external-api"),
)

// Use in your code
err := brk.Do(func() error {
    return callExternalAPI()
})
```

## Load Shedding Pattern

### Overview

Adaptive load shedding automatically rejects requests when system is overloaded based on CPU usage.

### ✅ Automatic Load Shedding

**Enabled automatically** in production mode (`Mode: pro` or `Mode: pre`):

```yaml
# Configuration
Name: user-api
Mode: pro  # ✅ Load shedding enabled
# Mode: dev  # ❌ Load shedding disabled for development
```

**Disabled in** `dev`, `test`, `rt` modes for development/testing.

### How It Works

```go
// Automatic behavior:
// 1. Monitors system CPU usage
// 2. When CPU > threshold (default 90%), starts rejecting requests
// 3. Rejection probability increases with CPU usage
// 4. Returns 503 Service Unavailable for rejected requests
// 5. Automatically recovers when CPU decreases
```

### ✅ Load Shedding in Action

```go
// Happens automatically in handlers
func UserHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Load shedding check happens here (automatic)
        // If system is overloaded, request rejected before this code runs

        var req types.UserRequest
        if err := httpx.Parse(r, &req); err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
            return
        }

        // ... normal processing
    }
}
```

### ✅ Metrics and Monitoring

```go
// Load shedding emits metrics automatically
// Monitor these metrics:
// - shed_requests_total: Total requests shed
// - cpu_usage: Current CPU usage
// - requests_in_flight: Current concurrent requests

// View in logs:
// {"level":"warning","content":"dropped request due to high load"}
```

## Rate Limiting Pattern

### Overview

go-zero provides multiple rate limiting strategies to protect services from overload.

### ✅ Token Bucket Rate Limiter (Redis-based)

For distributed rate limiting across multiple instances:

```go
import (
    "github.com/zeromicro/go-zero/core/limit"
    "github.com/zeromicro/go-zero/core/stores/redis"
)

// In service context
type ServiceContext struct {
    Config      config.Config
    RateLimiter *limit.TokenLimitHandler
}

func NewServiceContext(c config.Config) *ServiceContext {
    rds := redis.MustNewRedis(c.Redis)

    return &ServiceContext{
        Config: c,
        RateLimiter: limit.NewTokenLimiter(
            100,           // rate: 100 requests
            100,           // burst: 100 capacity
            rds,           // Redis store
            "api-limiter", // key prefix
        ),
    }
}

// In handler or middleware
func (l *SomeLogic) Process(req *types.Request) (*types.Response, error) {
    // Check rate limit
    code := l.svcCtx.RateLimiter.Allow()
    if code != limit.Allowed {
        return nil, errors.New("rate limit exceeded")
    }

    // Continue processing...
    return &types.Response{}, nil
}
```

### ✅ Period Limiter

For limiting requests within a time window:

```go
import "github.com/zeromicro/go-zero/core/limit"

// In service context
type ServiceContext struct {
    PeriodLimiter *limit.PeriodLimit
}

func NewServiceContext(c config.Config) *ServiceContext {
    rds := redis.MustNewRedis(c.Redis)

    return &ServiceContext{
        PeriodLimiter: limit.NewPeriodLimit(
            60,           // period: 60 seconds
            100,          // quota: 100 requests per period
            rds,
            "period-limiter",
        ),
    }
}

// Usage
func (l *SomeLogic) Process(userId int64, req *types.Request) error {
    // Limit per user
    key := fmt.Sprintf("user:%d", userId)

    code, err := l.svcCtx.PeriodLimiter.Take(key)
    if err != nil {
        return err
    }

    switch code {
    case limit.OverQuota:
        return errors.New("rate limit exceeded, try again later")
    case limit.Allowed:
        // Continue processing
        return l.processRequest(req)
    case limit.HitQuota:
        // Last allowed request in this period
        return l.processRequest(req)
    default:
        return errors.New("unknown limit status")
    }
}
```

### ✅ Rate Limiting Middleware

```go
// middleware/ratelimitmiddleware.go
type RateLimitMiddleware struct {
    limiter *limit.PeriodLimit
}

func NewRateLimitMiddleware(rds *redis.Redis) *RateLimitMiddleware {
    return &RateLimitMiddleware{
        limiter: limit.NewPeriodLimit(60, 1000, rds, "api"),
    }
}

func (m *RateLimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Rate limit by IP
        clientIP := r.RemoteAddr

        code, err := m.limiter.Take(clientIP)
        if err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
            return
        }

        if code == limit.OverQuota {
            httpx.ErrorCtx(r.Context(), w, errors.New("rate limit exceeded"))
            return
        }

        // Add rate limit headers
        w.Header().Set("X-RateLimit-Limit", "1000")
        w.Header().Set("X-RateLimit-Remaining", fmt.Sprint(1000-code))

        next.ServeHTTP(w, r)
    }
}
```

### ✅ Per-User Rate Limiting

```go
func (m *RateLimitMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Extract user ID from context (set by auth middleware)
        userId, ok := r.Context().Value("userId").(int64)
        if !ok {
            httpx.ErrorCtx(r.Context(), w, errors.New("unauthorized"))
            return
        }

        // Rate limit per user
        key := fmt.Sprintf("user:%d", userId)
        code, err := m.limiter.Take(key)
        if err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
            return
        }

        if code == limit.OverQuota {
            httpx.ErrorCtx(r.Context(), w, errors.New("rate limit exceeded"))
            return
        }

        next.ServeHTTP(w, r)
    }
}
```

## Timeout Pattern

### Overview

Cascading timeouts via context ensure requests don't hang indefinitely.

### ✅ Service-Level Timeout

```yaml
# Configuration
Name: user-api
Timeout: 30000  # 30 seconds (in milliseconds)
```

This sets the **default timeout for all requests**. Context automatically times out after this duration.

### ✅ Handler-Level Timeout

```go
// In handler
func UserHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Create timeout context (shorter than service timeout)
        ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
        defer cancel()

        // Use timeout context
        l := logic.NewSomeLogic(ctx, svcCtx)
        resp, err := l.Process(&req)

        // Handle timeout
        if errors.Is(err, context.DeadlineExceeded) {
            httpx.ErrorCtx(r.Context(), w, errors.New("request timeout"))
            return
        }

        if err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
            return
        }

        httpx.OkJsonCtx(r.Context(), w, resp)
    }
}
```

### ✅ Operation-Level Timeout

```go
func (l *SomeLogic) ProcessWithTimeout(req *types.Request) (*types.Response, error) {
    // Create timeout for specific operation
    ctx, cancel := context.WithTimeout(l.ctx, 3*time.Second)
    defer cancel()

    // Channel for result
    type result struct {
        data *types.Response
        err  error
    }
    ch := make(chan result, 1)

    // Run operation in goroutine
    go func() {
        data, err := l.doSlowOperation(ctx, req)
        ch <- result{data, err}
    }()

    // Wait for result or timeout
    select {
    case res := <-ch:
        return res.data, res.err
    case <-ctx.Done():
        return nil, errors.New("operation timeout")
    }
}
```

### ✅ RPC Client Timeout

```yaml
# RPC client configuration
UserRpc:
  Etcd:
    Hosts:
      - 127.0.0.1:2379
    Key: user.rpc
  Timeout: 5000  # 5 seconds for RPC calls
```

```go
// Timeout applied automatically
resp, err := l.svcCtx.UserRpc.GetUser(l.ctx, &user.GetUserRequest{
    Id: 123,
})
// Times out after 5 seconds
```

## Retry Pattern

### ✅ Simple Retry with Backoff

```go
import "github.com/zeromicro/go-zero/core/retry"

func (l *SomeLogic) CallWithRetry() error {
    return retry.Do(
        l.ctx,
        func() error {
            return l.callExternalAPI()
        },
        retry.WithAttempts(3),                           // Retry up to 3 times
        retry.WithDelay(time.Second),                    // Initial delay
        retry.WithBackoff(retry.BackoffExponential),     // Exponential backoff
    )
}
```

### ✅ Retry with Custom Logic

```go
func (l *SomeLogic) SmartRetry() error {
    var lastErr error

    for attempt := 0; attempt < 3; attempt++ {
        err := l.callExternalAPI()
        if err == nil {
            return nil
        }

        // Check if error is retryable
        if !isRetryable(err) {
            return err
        }

        lastErr = err

        // Exponential backoff
        backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second

        select {
        case <-time.After(backoff):
            continue
        case <-l.ctx.Done():
            return l.ctx.Err()
        }
    }

    return lastErr
}

func isRetryable(err error) bool {
    // Retry on specific errors
    switch {
    case errors.Is(err, context.DeadlineExceeded):
        return false  // Don't retry timeouts
    case errors.Is(err, ErrRateLimited):
        return true   // Retry rate limits
    case errors.Is(err, ErrTemporaryFailure):
        return true   // Retry temporary failures
    default:
        return false
    }
}
```

## Bulkhead Pattern

### ✅ Worker Pool

Limit concurrent operations to prevent resource exhaustion:

```go
import "github.com/zeromicro/go-zero/core/threading"

func (l *BatchLogic) ProcessBatch(items []*Item) error {
    // Create worker pool with max 10 concurrent workers
    workers := threading.NewTaskRunner(10)

    for _, item := range items {
        item := item  // Capture for goroutine
        workers.Schedule(func() {
            if err := l.processItem(item); err != nil {
                l.Logger.Errorf("failed to process item %d: %v", item.Id, err)
            }
        })
    }

    // Wait for all workers to complete
    workers.Wait()
    return nil
}
```

### ✅ MapReduce for Parallel Processing

```go
import "github.com/zeromicro/go-zero/core/mr"

func (l *DataLogic) ProcessLargeDataset(userIds []int64) ([]*UserData, error) {
    // Process in parallel with controlled concurrency
    results, err := mr.MapReduce(
        // Generate function - produces tasks
        func(source chan<- interface{}) {
            for _, id := range userIds {
                source <- id
            }
        },
        // Map function - processes each task
        func(item interface{}, writer mr.Writer, cancel func(error)) {
            id := item.(int64)

            userData, err := l.fetchUserData(id)
            if err != nil {
                l.Logger.Errorf("failed to fetch user %d: %v", id, err)
                return
            }

            writer.Write(userData)
        },
        // Reduce function - aggregates results
        func(pipe <-chan interface{}, writer mr.Writer, cancel func(error)) {
            var results []*UserData
            for item := range pipe {
                results = append(results, item.(*UserData))
            }
            writer.Write(results)
        },
        // Options
        mr.WithWorkers(10),  // 10 concurrent workers
    )

    if err != nil {
        return nil, err
    }

    return results.([]*UserData), nil
}
```

## Best Practices Summary

### ✅ DO:
- Use circuit breakers for all external calls
- Set appropriate timeouts at each layer
- Enable load shedding in production (`Mode: pro`)
- Use rate limiting for public APIs
- Implement retry with exponential backoff
- Use worker pools for batch operations
- Monitor CPU and request metrics
- Test resilience patterns under load
- Use context for timeout propagation
- Log circuit breaker state changes

### ❌ DON'T:
- Disable load shedding in production
- Set infinite timeouts
- Retry non-idempotent operations blindly
- Ignore context cancellation
- Create unbounded goroutines
- Share circuit breakers across unrelated services
- Use aggressive retry without backoff
- Forget to set rate limit headers
- Block on operations without timeout
- Ignore circuit breaker open state

## Configuration Examples

### Complete Resilience Configuration

```yaml
# REST API with full resilience
Name: user-api
Mode: pro  # Enable load shedding
Host: 0.0.0.0
Port: 8888
Timeout: 30000  # 30s service timeout

# RPC client with resilience
UserRpc:
  Etcd:
    Hosts:
      - 127.0.0.1:2379
    Key: user.rpc
  Timeout: 5000  # 5s RPC timeout
  # Circuit breaker and load balancing automatic

# Redis for rate limiting
Redis:
  Host: localhost:6379
  Type: node

# Database with circuit breaker (automatic)
DataSource: "user:pass@tcp(localhost:3306)/db"
```

## Monitoring Resilience

### Key Metrics to Monitor

```go
// Circuit breaker metrics
// - breaker_requests_total{name, state}
// - breaker_state_changes{name, from, to}

// Load shedding metrics
// - shed_requests_total
// - cpu_usage_percent

// Rate limiting metrics
// - rate_limit_exceeded_total{endpoint}
// - rate_limit_allowed_total{endpoint}

// Timeout metrics
// - request_timeout_total{endpoint}
// - request_duration_seconds{endpoint}
```

### Logging Best Practices

```go
// Log circuit breaker events
l.Logger.Infof("circuit breaker opened for %s", serviceName)
l.Logger.Infof("circuit breaker half-open for %s", serviceName)
l.Logger.Infof("circuit breaker closed for %s", serviceName)

// Log rate limiting
l.Logger.Warnf("rate limit exceeded for user %d", userId)

// Log load shedding
l.Logger.Warnf("request shed due to high CPU: %.2f%%", cpuUsage)

// Log timeouts
l.Logger.Errorf("request timeout after %v", timeout)
```

## Testing Resilience

```go
// Test circuit breaker
func TestCircuitBreaker(t *testing.T) {
    brk := breaker.NewBreaker()

    // Trigger breaker
    for i := 0; i < 20; i++ {
        brk.Do(func() error {
            return errors.New("failure")
        })
    }

    // Should be open now
    err := brk.Do(func() error {
        return nil
    })
    assert.Equal(t, breaker.ErrServiceUnavailable, err)
}

// Test timeout
func TestTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    err := longRunningOperation(ctx)
    assert.Equal(t, context.DeadlineExceeded, err)
}
```

For related patterns, see:
- [REST API Patterns](./rest-api-patterns.md) for API-level resilience
- [RPC Patterns](./rpc-patterns.md) for RPC resilience
- [Caching Strategies](./caching-strategies.md) for cache-based resilience
