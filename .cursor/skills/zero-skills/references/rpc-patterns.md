# RPC Patterns

## Core Architecture

go-zero uses gRPC for RPC communication with built-in:
- Service discovery via etcd
- Load balancing (default: p2c_ewma)
- Circuit breaker
- Rate limiting
- Distributed tracing

## Basic RPC Service Pattern

### 1. Define Protocol Buffer

```protobuf
// user.proto
syntax = "proto3";

package user;
option go_package = "./user";

message CreateUserRequest {
  string name = 1;
  string email = 2;
  int32 age = 3;
}

message CreateUserResponse {
  int64 id = 1;
  string message = 2;
}

message GetUserRequest {
  int64 id = 1;
}

message GetUserResponse {
  int64 id = 1;
  string name = 2;
  string email = 3;
  int32 age = 4;
}

service UserService {
  rpc CreateUser(CreateUserRequest) returns(CreateUserResponse);
  rpc GetUser(GetUserRequest) returns(GetUserResponse);
}
```

### 2. Generate Code

```bash
goctl rpc protoc user.proto --go_out=. --go-grpc_out=. --zrpc_out=.
```

Generated structure:
```
.
├── etc/
│   └── user.yaml           # Configuration
├── internal/
│   ├── config/
│   │   └── config.go       # Config struct
│   ├── logic/
│   │   ├── createuserlogic.go
│   │   └── getuserlogic.go
│   ├── server/
│   │   └── userserviceserver.go  # gRPC server implementation
│   └── svc/
│       └── servicecontext.go
├── user/
│   ├── user.pb.go          # Protocol buffer generated
│   ├── user_grpc.pb.go     # gRPC generated
│   └── userservice.go      # Client interface
├── user.go                 # Entry point
└── user.proto
```

### 3. Implement Logic

```go
// internal/logic/createuserlogic.go
type CreateUserLogic struct {
    ctx    context.Context
    svcCtx *svc.ServiceContext
    logx.Logger
}

func NewCreateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserLogic {
    return &CreateUserLogic{
        ctx:    ctx,
        svcCtx: svcCtx,
        Logger: logx.WithContext(ctx),
    }
}

func (l *CreateUserLogic) CreateUser(in *user.CreateUserRequest) (*user.CreateUserResponse, error) {
    // Validation
    if err := l.validateRequest(in); err != nil {
        return nil, status.Error(codes.InvalidArgument, err.Error())
    }

    // Business logic
    userModel := &model.User{
        Name:  in.Name,
        Email: in.Email,
        Age:   int64(in.Age),
    }

    result, err := l.svcCtx.UserModel.Insert(l.ctx, userModel)
    if err != nil {
        l.Logger.Errorf("failed to insert user: %v", err)
        return nil, status.Error(codes.Internal, "failed to create user")
    }

    userId, _ := result.LastInsertId()

    return &user.CreateUserResponse{
        Id:      userId,
        Message: "User created successfully",
    }, nil
}

func (l *CreateUserLogic) validateRequest(in *user.CreateUserRequest) error {
    if in.Name == "" {
        return errors.New("name is required")
    }
    if in.Age < 18 {
        return errors.New("age must be at least 18")
    }
    return nil
}
```

## RPC Configuration Pattern

### ✅ Correct Pattern

```go
// internal/config/config.go
type Config struct {
    zrpc.RpcServerConf  // ✅ Always embed for RPC services

    // Database
    DataSource string

    // Cache
    Cache cache.CacheConf

    // Custom settings
    MaxConnections int `json:",default=1000"`
}
```

### Server Configuration (YAML)

```yaml
# etc/user.yaml
Name: user.rpc
ListenOn: 0.0.0.0:8080

# Service discovery with etcd
Etcd:
  Hosts:
    - 127.0.0.1:2379
  Key: user.rpc

# OR use direct endpoints (no service discovery)
# Endpoints:
#   - 127.0.0.1:8080

# Telemetry
Telemetry:
  Name: user.rpc
  Endpoint: http://localhost:4318
  Sampler: 1.0
  Batcher: otlpgrpc

# Logging
Log:
  Mode: console
  Level: info
  Encoding: json

# Database
DataSource: "user:pass@tcp(localhost:3306)/users?parseTime=true"

# Cache
Cache:
  - Host: localhost:6379
    Type: node

# Circuit breaker enabled by default
# Health check enabled by default
Timeout: 5000  # milliseconds
```

## RPC Client Pattern

### ✅ Correct Pattern - With Service Discovery

```go
// Client with etcd service discovery
type UserClient struct {
    client user.UserServiceClient
}

func NewUserClient(conf zrpc.RpcClientConf) *UserClient {
    conn := zrpc.MustNewClient(conf)
    return &UserClient{
        client: user.NewUserServiceClient(conn.Conn()),
    }
}

func (c *UserClient) CreateUser(ctx context.Context, req *user.CreateUserRequest) (*user.CreateUserResponse, error) {
    return c.client.CreateUser(ctx, req)
}
```

### Client Configuration

```yaml
# Client with etcd
Etcd:
  Hosts:
    - 127.0.0.1:2379
  Key: user.rpc

# OR direct endpoints
Endpoints:
  - 127.0.0.1:8080
  - 127.0.0.1:8081

# OR custom target
Target: dns:///user-service:8080

Timeout: 5000
NonBlock: true  # Non-blocking connection

# Circuit breaker enabled by default
```

### Usage in Another Service

```go
// In API service calling RPC service
type ServiceContext struct {
    Config     config.Config
    UserRpc    user.UserServiceClient
}

func NewServiceContext(c config.Config) *ServiceContext {
    return &ServiceContext{
        Config: c,
        UserRpc: user.NewUserServiceClient(
            zrpc.MustNewClient(c.UserRpc).Conn(),
        ),
    }
}

// In logic layer
func (l *CreateOrderLogic) CreateOrder(req *types.CreateOrderRequest) (*types.CreateOrderResponse, error) {
    // Call user RPC service
    userResp, err := l.svcCtx.UserRpc.GetUser(l.ctx, &user.GetUserRequest{
        Id: req.UserId,
    })
    if err != nil {
        return nil, err
    }

    // Continue with order creation...
}
```

## Error Handling Pattern

### ✅ Correct Pattern

Use gRPC status codes for errors:

```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

func (l *GetUserLogic) GetUser(in *user.GetUserRequest) (*user.GetUserResponse, error) {
    // Validation error
    if in.Id <= 0 {
        return nil, status.Error(codes.InvalidArgument, "invalid user id")
    }

    // Find user
    userModel, err := l.svcCtx.UserModel.FindOne(l.ctx, in.Id)
    if err != nil {
        if errors.Is(err, model.ErrNotFound) {
            return nil, status.Error(codes.NotFound, "user not found")
        }
        l.Logger.Errorf("failed to find user: %v", err)
        return nil, status.Error(codes.Internal, "internal server error")
    }

    return &user.GetUserResponse{
        Id:    userModel.Id,
        Name:  userModel.Name,
        Email: userModel.Email,
        Age:   int32(userModel.Age),
    }, nil
}
```

### gRPC Status Code Mapping

```go
// Common error mappings
var (
    // 400 Bad Request
    ErrInvalidArgument = status.Error(codes.InvalidArgument, "invalid argument")

    // 401 Unauthorized
    ErrUnauthenticated = status.Error(codes.Unauthenticated, "unauthenticated")

    // 403 Forbidden
    ErrPermissionDenied = status.Error(codes.PermissionDenied, "permission denied")

    // 404 Not Found
    ErrNotFound = status.Error(codes.NotFound, "not found")

    // 409 Conflict
    ErrAlreadyExists = status.Error(codes.AlreadyExists, "already exists")

    // 429 Too Many Requests
    ErrResourceExhausted = status.Error(codes.ResourceExhausted, "rate limit exceeded")

    // 500 Internal Server Error
    ErrInternal = status.Error(codes.Internal, "internal server error")

    // 503 Service Unavailable
    ErrUnavailable = status.Error(codes.Unavailable, "service unavailable")
)
```

### Client-Side Error Handling

```go
resp, err := userClient.GetUser(ctx, &user.GetUserRequest{Id: 123})
if err != nil {
    st, ok := status.FromError(err)
    if ok {
        switch st.Code() {
        case codes.NotFound:
            // Handle not found
            return nil, errors.New("user not found")
        case codes.InvalidArgument:
            // Handle validation error
            return nil, errors.New("invalid request")
        case codes.Unavailable:
            // Handle service unavailable
            return nil, errors.New("service temporarily unavailable")
        default:
            // Handle other errors
            return nil, err
        }
    }
    return nil, err
}
```

## Interceptor Pattern

### ✅ Server Interceptor

```go
// Unary interceptor
func UnaryAuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    // Extract metadata
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "missing metadata")
    }

    // Validate token
    tokens := md.Get("authorization")
    if len(tokens) == 0 {
        return nil, status.Error(codes.Unauthenticated, "missing authorization token")
    }

    userId, err := validateToken(tokens[0])
    if err != nil {
        return nil, status.Error(codes.Unauthenticated, "invalid token")
    }

    // Add user info to context
    ctx = context.WithValue(ctx, "userId", userId)

    // Call handler
    return handler(ctx, req)
}

// Register interceptor
func main() {
    c := config.Config{}
    conf.MustLoad(*configFile, &c)

    server := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
        user.RegisterUserServiceServer(grpcServer, srv)
    })

    // Add interceptor
    server.AddUnaryInterceptors(UnaryAuthInterceptor)

    server.Start()
}
```

### ✅ Client Interceptor

```go
// Unary client interceptor
func UnaryClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
    // Add metadata to outgoing request
    md := metadata.Pairs(
        "authorization", "Bearer token",
        "request-id", generateRequestId(),
    )
    ctx = metadata.NewOutgoingContext(ctx, md)

    // Log request
    logx.Infof("calling method: %s", method)

    // Call RPC
    err := invoker(ctx, method, req, reply, cc, opts...)

    // Log response
    if err != nil {
        logx.Errorf("method %s failed: %v", method, err)
    }

    return err
}

// Register client interceptor
conn := zrpc.MustNewClient(c.UserRpc, zrpc.WithUnaryClientInterceptor(UnaryClientInterceptor))
```

## Streaming Pattern

### ✅ Server Streaming

```protobuf
// Protocol buffer definition
service UserService {
  rpc ListUsers(ListUsersRequest) returns(stream UserInfo);
}

message ListUsersRequest {
  int32 page_size = 1;
}

message UserInfo {
  int64 id = 1;
  string name = 2;
  string email = 3;
}
```

```go
// Implementation
func (l *ListUsersLogic) ListUsers(in *user.ListUsersRequest, stream user.UserService_ListUsersServer) error {
    offset := 0
    pageSize := int(in.PageSize)
    if pageSize <= 0 {
        pageSize = 10
    }

    for {
        users, err := l.svcCtx.UserModel.FindMany(l.ctx, offset, pageSize)
        if err != nil {
            return status.Error(codes.Internal, "failed to fetch users")
        }

        if len(users) == 0 {
            break
        }

        for _, u := range users {
            if err := stream.Send(&user.UserInfo{
                Id:    u.Id,
                Name:  u.Name,
                Email: u.Email,
            }); err != nil {
                return err
            }
        }

        offset += pageSize
    }

    return nil
}
```

### ✅ Client Streaming

```protobuf
service UserService {
  rpc BatchCreateUsers(stream CreateUserRequest) returns(BatchCreateResponse);
}

message BatchCreateResponse {
  int32 total = 1;
  repeated int64 ids = 2;
}
```

```go
func (l *BatchCreateUsersLogic) BatchCreateUsers(stream user.UserService_BatchCreateUsersServer) error {
    var ids []int64
    count := 0

    for {
        req, err := stream.Recv()
        if err == io.EOF {
            // Client finished sending
            return stream.SendAndClose(&user.BatchCreateResponse{
                Total: int32(count),
                Ids:   ids,
            })
        }
        if err != nil {
            return err
        }

        // Process each user
        userModel := &model.User{
            Name:  req.Name,
            Email: req.Email,
            Age:   int64(req.Age),
        }

        result, err := l.svcCtx.UserModel.Insert(l.ctx, userModel)
        if err != nil {
            l.Logger.Errorf("failed to insert user: %v", err)
            continue
        }

        userId, _ := result.LastInsertId()
        ids = append(ids, userId)
        count++
    }
}
```

### ✅ Bidirectional Streaming

```protobuf
service ChatService {
  rpc Chat(stream ChatMessage) returns(stream ChatMessage);
}
```

```go
func (l *ChatLogic) Chat(stream user.ChatService_ChatServer) error {
    for {
        msg, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return err
        }

        // Process message
        response := &user.ChatMessage{
            UserId:  msg.UserId,
            Content: fmt.Sprintf("Echo: %s", msg.Content),
            Time:    time.Now().Unix(),
        }

        if err := stream.Send(response); err != nil {
            return err
        }
    }
}
```

## Service Discovery Pattern

### ✅ With etcd

```yaml
# Server configuration
Name: user.rpc
ListenOn: 0.0.0.0:8080
Etcd:
  Hosts:
    - 127.0.0.1:2379
    - 127.0.0.1:22379
    - 127.0.0.1:32379
  Key: user.rpc  # Service will register under this key
```

```yaml
# Client configuration
Etcd:
  Hosts:
    - 127.0.0.1:2379
  Key: user.rpc  # Discover services registered under this key
```

### ✅ Direct Endpoints (No Discovery)

```yaml
# Client with static endpoints
Endpoints:
  - 127.0.0.1:8080
  - 127.0.0.1:8081
  - 127.0.0.1:8082

# Load balancing still works across endpoints
```

### ✅ Custom Target

```yaml
# Kubernetes DNS
Target: dns:///user-service.default.svc.cluster.local:8080

# OR custom resolver
Target: my-resolver:///user-service
```

## Load Balancing

go-zero uses **p2c_ewma** (Power of 2 Choices with EWMA) by default:

```go
// Automatically applied, no configuration needed
// Selects between 2 random servers based on:
// - Response latency (EWMA)
// - Active request count
// - Server health
```

To use a different load balancer:

```yaml
# Client configuration
Etcd:
  Hosts:
    - 127.0.0.1:2379
  Key: user.rpc

# Optional: override load balancer
# Options: round_robin, pick_first, grpclb
# Default is p2c_ewma (recommended)
```

## Best Practices Summary

### ✅ DO:
- Always embed `zrpc.RpcServerConf` in server config
- Use gRPC status codes for errors
- Implement proper context propagation
- Use service discovery (etcd) for dynamic environments
- Enable circuit breaker (enabled by default)
- Use interceptors for cross-cutting concerns
- Log errors with context
- Return meaningful error messages
- Use streaming for large data sets
- Set appropriate timeouts

### ❌ DON'T:
- Return `nil, nil` - always return proper response or error
- Use plain `error` types - use `status.Error()`
- Block in streaming handlers
- Ignore context cancellation
- Log sensitive data (passwords, tokens)
- Create goroutines without cleanup
- Use global variables for connections
- Disable circuit breaker in production
- Set infinite timeouts
- Forget to handle io.EOF in streaming

## When to Use RPC vs REST

### Use RPC when:
- Service-to-service communication
- High performance requirements
- Strong typing needed
- Streaming data
- Microservices architecture
- Internal APIs

### Use REST when:
- Public APIs
- Browser clients
- Third-party integrations
- Simple CRUD operations
- HTTP standards required

For REST patterns, see [REST API Patterns](./rest-api-patterns.md).
For combining both, see [Gateway Patterns](./gateway-patterns.md).
