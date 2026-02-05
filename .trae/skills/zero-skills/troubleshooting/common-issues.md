# Common Issues and Solutions

## Installation Issues

### Issue: goctl command not found

**Symptoms:**
```bash
$ goctl --version
zsh: command not found: goctl
```

**Solution:**
```bash
# Install goctl
go install github.com/zeromicro/go-zero/tools/goctl@latest

# Ensure $GOPATH/bin is in PATH
export PATH=$PATH:$(go env GOPATH)/bin

# Verify installation
goctl --version
```

### Issue: go-zero version mismatch

**Symptoms:**
```
undefined: rest.RestConf
```

**Solution:**
```bash
# Update go-zero to latest
go get -u github.com/zeromicro/go-zero

# Clean module cache
go clean -modcache

# Tidy dependencies
go mod tidy
```

## Code Generation Issues

### Issue: goctl api generate fails

**Symptoms:**
```bash
$ goctl api go -api user.api -dir .
Error: invalid syntax at line 10
```

**Solution:**

Check API syntax - common mistakes:

```go
// ❌ Wrong: Missing syntax declaration
type Request {
    Name string `json:"name"`
}

// ✅ Correct: Include syntax
syntax = "v1"

type Request {
    Name string `json:"name"`
}

// ❌ Wrong: Using 'any' type
type Request {
    Data any `json:"data"`
}

// ✅ Correct: Use concrete types
type Request {
    Data map[string]string `json:"data"`
}

// ❌ Wrong: Missing return type
@handler GetUser
get /users/:id (GetUserRequest)

// ✅ Correct: Include returns
@handler GetUser
get /users/:id (GetUserRequest) returns (GetUserResponse)
```

### Issue: goctl rpc generate fails

**Symptoms:**
```bash
$ goctl rpc protoc user.proto --go_out=. --go-grpc_out=. --zrpc_out=.
Error: protoc-gen-go: program not found
```

**Solution:**
```bash
# Install required tools
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Ensure they're in PATH
export PATH=$PATH:$(go env GOPATH)/bin

# Regenerate
goctl rpc protoc user.proto --go_out=. --go-grpc_out=. --zrpc_out=.
```

## Runtime Issues

### Issue: Service won't start - port already in use

**Symptoms:**
```
Error: listen tcp :8888: bind: address already in use
```

**Solution:**
```bash
# Find process using the port
lsof -i :8888

# Kill the process
kill -9 <PID>

# Or change port in config
# etc/config.yaml
Port: 8889  # Use different port
```

### Issue: Database connection fails

**Symptoms:**
```
Error: dial tcp: lookup localhost: no such host
Error: access denied for user
```

**Solution:**

Check connection string format:

```go
// ✅ Correct MySQL connection string
DataSource: "user:password@tcp(localhost:3306)/database?parseTime=true"

// Common mistakes:
// ❌ Missing parseTime parameter (for time.Time)
DataSource: "user:password@tcp(localhost:3306)/database"

// ❌ Wrong host
DataSource: "user:password@tcp(127.0.0.1:3306)/database?parseTime=true"  // Use actual host

// ❌ Special characters in password not encoded
DataSource: "user:p@ssw0rd@tcp(localhost:3306)/database"  // @ in password breaks parsing
// ✅ Encode special characters
DataSource: "user:p%40ssw0rd@tcp(localhost:3306)/database"
```

Verify connection:
```bash
# Test MySQL connection
mysql -h localhost -u user -p database

# Test from container
docker exec -it mysql mysql -u user -p database
```

### Issue: Redis connection fails

**Symptoms:**
```
Error: dial tcp: connect: connection refused
Error: NOAUTH Authentication required
```

**Solution:**

```yaml
# etc/config.yaml

# ✅ Correct Redis config
Redis:
  Host: localhost:6379
  Type: node
  Pass: ""  # Empty if no password

# With password
Redis:
  Host: localhost:6379
  Type: node
  Pass: "your-redis-password"

# Redis Cluster
Redis:
  Host: localhost:6379,localhost:6380,localhost:6381
  Type: cluster
  Pass: ""
```

Test connection:
```bash
# Test Redis
redis-cli -h localhost -p 6379 ping

# With password
redis-cli -h localhost -p 6379 -a password ping
```

## API Issues

### Issue: 404 Not Found for valid endpoint

**Symptoms:**
```bash
$ curl http://localhost:8888/api/users
404 page not found
```

**Solution:**

Check route registration:

```go
// API definition
@server(
    prefix: /api/v1  // ✅ Check prefix
    group: user
)
service user-api {
    @handler GetUsers
    get /users (GetUsersRequest) returns (GetUsersResponse)
}

// Actual URL will be: /api/v1/users (not /api/users)
```

Verify routes are registered:
```go
// In main function
func main() {
    // Enable route logging
    logx.DisableStat()

    // Routes will be logged on startup
    server.Start()
}
```

### Issue: Request body not parsed

**Symptoms:**
```go
// All request fields are empty/zero
req.Name == ""
req.Age == 0
```

**Solution:**

Check Content-Type header:

```bash
# ❌ Wrong: Missing Content-Type
curl -X POST http://localhost:8888/api/users \
  -d '{"name":"John"}'

# ✅ Correct: Include Content-Type
curl -X POST http://localhost:8888/api/users \
  -H "Content-Type: application/json" \
  -d '{"name":"John"}'
```

Check JSON tags:

```go
// ❌ Wrong: Missing json tags
type Request struct {
    Name string
    Age  int
}

// ✅ Correct: Include json tags
type Request struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}
```

### Issue: Path parameter not parsed

**Symptoms:**
```go
// Path parameter is 0 or empty
req.Id == 0
```

**Solution:**

Check tag in type definition:

```go
// ❌ Wrong: Using json tag for path parameter
type GetUserRequest struct {
    Id int64 `json:"id"`
}

// ✅ Correct: Use path tag
type GetUserRequest struct {
    Id int64 `path:"id"`
}

// API definition
@handler GetUser
get /users/:id (GetUserRequest) returns (GetUserResponse)
```

## RPC Issues

### Issue: RPC service not discovered

**Symptoms:**
```
Error: rpc error: code = Unavailable desc = connection error
```

**Solution:**

Check etcd configuration:

```yaml
# Server configuration (etc/user-rpc.yaml)
Name: user.rpc
ListenOn: 0.0.0.0:8080
Etcd:
  Hosts:
    - 127.0.0.1:2379  # ✅ Ensure etcd is running
  Key: user.rpc        # ✅ Consistent key

# Client configuration
UserRpc:
  Etcd:
    Hosts:
      - 127.0.0.1:2379  # ✅ Same etcd host
    Key: user.rpc        # ✅ Same key
```

Verify etcd:
```bash
# Check etcd is running
etcdctl version

# List registered services
etcdctl get --prefix user.rpc

# Start etcd if not running
etcd
```

### Issue: RPC call timeout

**Symptoms:**
```
Error: rpc error: code = DeadlineExceeded desc = context deadline exceeded
```

**Solution:**

Increase timeout:

```yaml
# Client configuration
UserRpc:
  Etcd:
    Hosts:
      - 127.0.0.1:2379
    Key: user.rpc
  Timeout: 10000  # ✅ Increase to 10 seconds (was 5000)
```

Or pass context with timeout:

```go
// Create timeout context
ctx, cancel := context.WithTimeout(l.ctx, 10*time.Second)
defer cancel()

// Use timeout context
resp, err := l.svcCtx.UserRpc.GetUser(ctx, &user.GetUserRequest{
    Id: 123,
})
```

### Issue: gRPC status error not handled

**Symptoms:**
```go
// Error doesn't match expected type
if err == ErrNotFound {  // Never true
    // ...
}
```

**Solution:**

Use gRPC status:

```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// ❌ Wrong: Direct error comparison
if err == model.ErrNotFound {
    return nil, err
}

// ✅ Correct: Check gRPC status
st, ok := status.FromError(err)
if ok {
    switch st.Code() {
    case codes.NotFound:
        return nil, ErrUserNotFound
    case codes.InvalidArgument:
        return nil, ErrInvalidInput
    default:
        return nil, err
    }
}
```

## Database Issues

### Issue: SQL syntax error

**Symptoms:**
```
Error: Error 1064: You have an error in your SQL syntax
```

**Solution:**

Check generated model:

```go
// Custom query in model
func (m *customUsersModel) FindByEmail(ctx context.Context, email string) (*Users, error) {
    // ❌ Wrong: Missing WHERE clause
    query := `SELECT * FROM users email = ?`

    // ✅ Correct: Include WHERE
    query := `SELECT * FROM users WHERE email = ?`

    var user Users
    err := m.QueryRowNoCacheCtx(ctx, &user, query, email)
    return &user, err
}
```

### Issue: Cache key conflict

**Symptoms:**
```
Error: wrong type returned from cache
Stale data returned after update
```

**Solution:**

Regenerate model with proper cache keys:

```bash
# Regenerate model to fix cache keys
goctl model mysql datasource \
  -url="user:pass@tcp(localhost:3306)/db" \
  -table="users" \
  -dir="./model" \
  -c  # ✅ Enable cache
```

Or manually delete cache:

```go
// After update/delete
err := m.DelCacheCtx(ctx,
    fmt.Sprintf("user:%d", userId),
    fmt.Sprintf("user:email:%s", email),
)
```

### Issue: Transaction rollback not working

**Symptoms:**
```
Error: data committed despite error
```

**Solution:**

Return error from transaction:

```go
// ❌ Wrong: Error not returned
err := l.svcCtx.DB.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
    _, err := session.ExecCtx(ctx, query1)
    if err != nil {
        l.Logger.Error(err)  // ❌ Logged but not returned
    }

    _, err = session.ExecCtx(ctx, query2)
    return nil  // ❌ Returns nil, transaction commits
})

// ✅ Correct: Return error to rollback
err := l.svcCtx.DB.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
    _, err := session.ExecCtx(ctx, query1)
    if err != nil {
        return fmt.Errorf("query1 failed: %w", err)  // ✅ Returns error, rollback
    }

    _, err = session.ExecCtx(ctx, query2)
    if err != nil {
        return fmt.Errorf("query2 failed: %w", err)  // ✅ Returns error, rollback
    }

    return nil  // ✅ Only commits if all succeed
})
```

## Middleware Issues

### Issue: Middleware not applied

**Symptoms:**
```
Auth middleware not checking tokens
CORS not working
```

**Solution:**

Check middleware registration:

```go
// In API definition
@server(
    prefix: /api/v1
    group: user
    middleware: Auth  // ✅ Register middleware
)
service user-api {
    @handler GetUser
    get /users/:id (GetUserRequest) returns (GetUserResponse)
}

// In handler registration (generated)
func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
    server.AddRoutes(
        []rest.Route{
            {
                Method:  http.MethodGet,
                Path:    "/users/:id",
                Handler: GetUserHandler(serverCtx),
            },
        },
        rest.WithPrefix("/api/v1"),
        rest.WithMiddlewares([]rest.Middleware{
            serverCtx.Auth,  // ✅ Middleware applied
        }),
    )
}
```

### Issue: Middleware order wrong

**Symptoms:**
```
Auth middleware runs after logging
Context not available in middleware
```

**Solution:**

Control middleware order:

```go
// ✅ Correct order: Auth before other middlewares
@server(
    prefix: /api/v1
    group: user
    middleware: Auth, RateLimit, Logging  // Auth runs first
)
```

Or use chain:

```go
import "github.com/zeromicro/go-zero/rest/chain"

// Control explicit order
middlewares := chain.New(
    authMiddleware,      // Runs first
    rateLimitMiddleware, // Runs second
    loggingMiddleware,   // Runs last
)
```

## Configuration Issues

### Issue: Config not loaded

**Symptoms:**
```
Error: config file not found
All config values are zero/empty
```

**Solution:**

Check config file path:

```bash
# ❌ Wrong: Relative path may not work
./service -f config.yaml

# ✅ Correct: Use absolute path or explicit relative
./service -f etc/config.yaml
./service -f /app/etc/config.yaml

# Or set working directory
cd /app && ./service -f etc/config.yaml
```

### Issue: Config validation fails

**Symptoms:**
```
Error: invalid config: missing required field
```

**Solution:**

Check required fields:

```go
// In config struct
type Config struct {
    rest.RestConf  // ✅ Must embed

    DataSource string  // ✅ Required (no default, optional, or omitempty)

    // Optional field
    Optional string `json:",optional"`  // ✅ Can be missing

    // With default
    MaxSize int64 `json:",default=1048576"`  // ✅ Has default value
}
```

In YAML:

```yaml
# ✅ Required fields must be present
Name: user-api  # Required from RestConf
Host: 0.0.0.0   # Required from RestConf
Port: 8888      # Required from RestConf
DataSource: "..."  # Required from Config

# Optional fields can be omitted
# MaxSize will use default if not specified
```

## Performance Issues

### Issue: High memory usage

**Symptoms:**
```
OOM (Out of Memory) errors
Memory steadily increasing
```

**Solution:**

Check for goroutine leaks:

```go
// ❌ Wrong: Goroutine never stops
go func() {
    for {
        doWork()
        time.Sleep(time.Second)
    }
}()

// ✅ Correct: Goroutine respects context
go func() {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            doWork()
        case <-ctx.Done():
            return  // ✅ Exit goroutine
        }
    }
}()
```

Check connection leaks:

```go
// Use connection pooling (automatic with go-zero)
// Limit concurrent connections
server := rest.MustNewServer(c.RestConf,
    rest.WithMaxConns(1000),  // ✅ Limit connections
)
```

### Issue: Slow database queries

**Symptoms:**
```
High response times
Database CPU at 100%
```

**Solution:**

Add indexes:

```sql
-- Check slow queries
SHOW FULL PROCESSLIST;

-- Add indexes for frequent queries
CREATE INDEX idx_email ON users(email);
CREATE INDEX idx_created_at ON users(created_at);
```

Use pagination:

```go
// ❌ Wrong: Loading all records
users, _ := l.svcCtx.UsersModel.FindAll(l.ctx)

// ✅ Correct: Paginated queries
users, total, _ := l.svcCtx.UsersModel.FindWithPagination(l.ctx, page, pageSize)
```

Enable caching:

```go
// ✅ Use cache for read-heavy tables
model.NewUsersModel(conn, c.Cache)  // Enable cache
```

## Logging Issues

### Issue: Logs not showing

**Symptoms:**
```
No log output despite logger calls
```

**Solution:**

Check log level:

```yaml
Log:
  Level: info  # ✅ Ensure level allows your logs
  # debug < info < error < severe
```

Check log mode:

```yaml
Log:
  Mode: console  # ✅ For development (stdout)
  # Mode: file   # For production (writes to file)
```

Ensure proper logger usage:

```go
// ❌ Wrong: Using standard library
log.Println("message")  // Doesn't use go-zero logging

// ✅ Correct: Use logx
l.Logger.Info("message")
logx.Info("message")  // Package level
```

For more help:
- [go-zero Documentation](https://go-zero.dev)
- [GitHub Discussions](https://github.com/zeromicro/go-zero/discussions)
- [GitHub Issues](https://github.com/zeromicro/go-zero/issues)
