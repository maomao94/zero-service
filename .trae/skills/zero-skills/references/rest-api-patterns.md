# REST API Patterns

## Core Architecture

### Three-Layer Pattern

go-zero REST APIs follow a strict three-layer architecture:

1. **Handler Layer** (`internal/handler/`) - HTTP concerns only
2. **Logic Layer** (`internal/logic/`) - Business logic implementation
3. **Service Context** (`internal/svc/`) - Dependency injection

```
HTTP Request → Handler → Logic → External Services/Database
                  ↓
            Service Context (dependencies)
```

## Handler Pattern

### ✅ Correct Pattern

Handlers should only handle HTTP-specific concerns:

```go
// internal/handler/userhandler.go
func CreateUserHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req types.CreateUserRequest
        if err := httpx.Parse(r, &req); err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
            return
        }

        l := logic.NewCreateUserLogic(r.Context(), svcCtx)
        resp, err := l.CreateUser(&req)
        if err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
        } else {
            httpx.OkJsonCtx(r.Context(), w, resp)
        }
    }
}
```

**Key Points:**
- Parse request with `httpx.Parse(r, &req)`
- Create logic instance with context
- Use `httpx.ErrorCtx` for errors (proper context propagation)
- Use `httpx.OkJsonCtx` for success responses
- No business logic in handler

### ❌ Common Mistakes

```go
// DON'T: Business logic in handler
func BadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // ❌ Database operations in handler
        user, err := svcCtx.UserModel.FindOne(ctx, id)

        // ❌ Complex validation in handler
        if user.Age < 18 {
            // validation logic
        }

        // ❌ Multiple service calls in handler
        profile, _ := svcCtx.ProfileModel.FindOne(ctx, user.ProfileId)
    }
}

// DON'T: Direct error responses
httpx.Error(w, err)  // ❌ Missing context
http.Error(w, "error", 500)  // ❌ Use httpx package

// DON'T: Manual JSON marshaling
json.NewEncoder(w).Encode(resp)  // ❌ Use httpx.OkJsonCtx
```

## Logic Pattern

### ✅ Correct Pattern

All business logic belongs in the logic layer:

```go
// internal/logic/createuserlogic.go
type CreateUserLogic struct {
    logx.Logger
    ctx    context.Context
    svcCtx *svc.ServiceContext
}

func NewCreateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserLogic {
    return &CreateUserLogic{
        Logger: logx.WithContext(ctx),
        ctx:    ctx,
        svcCtx: svcCtx,
    }
}

func (l *CreateUserLogic) CreateUser(req *types.CreateUserRequest) (*types.CreateUserResponse, error) {
    // Validation
    if err := l.validateUser(req); err != nil {
        return nil, err
    }

    // Business logic
    user := &model.User{
        Name:  req.Name,
        Email: req.Email,
        Age:   req.Age,
    }

    // Database operation
    result, err := l.svcCtx.UserModel.Insert(l.ctx, user)
    if err != nil {
        l.Logger.Errorf("failed to insert user: %v", err)
        return nil, err
    }

    userId, _ := result.LastInsertId()

    return &types.CreateUserResponse{
        Id:      userId,
        Message: "User created successfully",
    }, nil
}

func (l *CreateUserLogic) validateUser(req *types.CreateUserRequest) error {
    if req.Age < 18 {
        return errors.New("user must be at least 18 years old")
    }
    // More validation...
    return nil
}
```

**Key Points:**
- Always pass and use `context.Context`
- Use embedded `logx.Logger` for structured logging
- Access dependencies through `svcCtx`
- Return domain errors, let middleware handle HTTP status codes
- Private helper methods for complex validation/processing

## Configuration Pattern

### ✅ Correct Pattern

Always embed `service.ServiceConf` for REST services:

```go
// internal/config/config.go
type Config struct {
    rest.RestConf  // ✅ Always embed for REST services

    // Database configuration
    DataSource string

    // Redis configuration
    Cache cache.CacheConf

    // Custom settings
    MaxFileSize int64 `json:",default=10485760"` // 10MB default

    // Optional field
    FeatureFlag string `json:",optional"`

    // Validated options
    Environment string `json:",default=prod,options=[dev|test|prod]"`
}
```

### Configuration File (YAML)

```yaml
# etc/api.yaml
Name: user-api
Host: 0.0.0.0
Port: 8888
Timeout: 30000  # milliseconds

Log:
  Mode: console
  Level: info

DataSource: "user:pass@tcp(localhost:3306)/users?parseTime=true"

Cache:
  - Host: localhost:6379
    Type: node

MaxFileSize: 52428800  # 50MB
Environment: prod
```

## Middleware Pattern

### ✅ Correct Pattern

Middlewares wrap handlers and can be chained:

```go
// internal/middleware/authmiddleware.go
type AuthMiddleware struct {
    secret string
}

func NewAuthMiddleware(secret string) *AuthMiddleware {
    return &AuthMiddleware{secret: secret}
}

func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Pre-processing: validate token
        token := r.Header.Get("Authorization")
        if token == "" {
            httpx.ErrorCtx(r.Context(), w, errors.New("missing authorization"))
            return
        }

        // Verify token and extract user info
        userId, err := m.verifyToken(token)
        if err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
            return
        }

        // Add user info to context
        ctx := context.WithValue(r.Context(), "userId", userId)

        // Call next handler with updated context
        next.ServeHTTP(w, r.WithContext(ctx))

        // Post-processing (if needed)
        // Can add logging, metrics, etc.
    }
}

func (m *AuthMiddleware) verifyToken(token string) (int64, error) {
    // JWT verification logic
    return userId, nil
}
```

### Registering Middleware

```go
// main function or route registration
server := rest.MustNewServer(c.RestConf, rest.WithChain(
    // Built-in middlewares
    rest.WithNotAllowedHandler(handler.CorsHandler()),  // CORS
    rest.WithUnauthorizedCallback(unauthorizedCallback),
))
defer server.Stop()

// Custom middleware
authMiddleware := middleware.NewAuthMiddleware(c.Secret)

// Apply to specific routes
handler.RegisterHandlers(server, serverCtx, authMiddleware)
```

## Request/Response Types

### ✅ Correct Pattern

Define clear types with proper validation tags:

```go
// API definition (.api file)
type (
    CreateUserRequest {
        Name     string `json:"name" validate:"required,min=2,max=50"`
        Email    string `json:"email" validate:"required,email"`
        Age      int    `json:"age" validate:"required,gte=18,lte=120"`
        Password string `json:"password" validate:"required,min=8"`
    }

    CreateUserResponse {
        Id      int64  `json:"id"`
        Message string `json:"message"`
    }

    GetUserRequest {
        Id int64 `path:"id" validate:"required,gt=0"`
    }

    GetUserResponse {
        Id    int64  `json:"id"`
        Name  string `json:"name"`
        Email string `json:"email"`
        Age   int    `json:"age"`
    }

    ListUsersRequest {
        Page     int    `form:"page,default=1" validate:"gte=1"`
        PageSize int    `form:"page_size,default=10" validate:"gte=1,lte=100"`
        Keyword  string `form:"keyword,optional"`
    }

    ListUsersResponse {
        Total int64       `json:"total"`
        Users []UserInfo  `json:"users"`
    }

    UserInfo {
        Id    int64  `json:"id"`
        Name  string `json:"name"`
        Email string `json:"email"`
    }
)
```

**Tag Reference:**
- `json` - JSON field name
- `path` - Path parameter (e.g., `/users/:id`)
- `form` - Query parameter or form data
- `header` - HTTP header
- `validate` - Validation rules
- `optional` - Field is optional
- `default` - Default value

## Error Handling

### ✅ Correct Pattern

```go
// Define custom errors
var (
    ErrUserNotFound     = errors.New("user not found")
    ErrInvalidInput     = errors.New("invalid input")
    ErrUnauthorized     = errors.New("unauthorized")
    ErrDuplicateEmail   = errors.New("email already exists")
)

// In logic layer
func (l *CreateUserLogic) CreateUser(req *types.CreateUserRequest) (*types.CreateUserResponse, error) {
    // Check for duplicate
    existing, err := l.svcCtx.UserModel.FindOneByEmail(l.ctx, req.Email)
    if err != nil && !errors.Is(err, model.ErrNotFound) {
        return nil, fmt.Errorf("failed to check existing user: %w", err)
    }
    if existing != nil {
        return nil, ErrDuplicateEmail
    }

    // Insert user
    result, err := l.svcCtx.UserModel.Insert(l.ctx, user)
    if err != nil {
        l.Logger.Errorf("failed to insert user: %v", err)
        return nil, fmt.Errorf("failed to create user: %w", err)
    }

    return &types.CreateUserResponse{
        Id:      userId,
        Message: "User created successfully",
    }, nil
}
```

### Custom Error Handler

```go
// Register custom error handler
httpx.SetErrorHandler(func(err error) (int, any) {
    switch {
    case errors.Is(err, ErrUserNotFound):
        return http.StatusNotFound, map[string]string{"error": err.Error()}
    case errors.Is(err, ErrInvalidInput):
        return http.StatusBadRequest, map[string]string{"error": err.Error()}
    case errors.Is(err, ErrUnauthorized):
        return http.StatusUnauthorized, map[string]string{"error": err.Error()}
    case errors.Is(err, ErrDuplicateEmail):
        return http.StatusConflict, map[string]string{"error": err.Error()}
    default:
        return http.StatusInternalServerError, map[string]string{"error": "internal server error"}
    }
})
```

## Service Context Pattern

### ✅ Correct Pattern

Centralize all dependencies in service context:

```go
// internal/svc/servicecontext.go
type ServiceContext struct {
    Config        config.Config
    UserModel     model.UserModel
    Cache         cache.Cache
    AuthMiddleware rest.Middleware
    Logger        logx.Logger
}

func NewServiceContext(c config.Config) *ServiceContext {
    // Initialize database connection
    conn := sqlx.NewMysql(c.DataSource)

    // Initialize Redis cache
    rds := redis.MustNewRedis(c.Cache[0].RedisConf)

    return &ServiceContext{
        Config:    c,
        UserModel: model.NewUserModel(conn, c.Cache),
        Cache:     cache.New(rds),
        AuthMiddleware: middleware.NewAuthMiddleware(c.Secret).Handle,
        Logger:    logx.WithContext(context.Background()),
    }
}
```

**Key Points:**
- Initialize all shared resources once
- Share database connections and cache clients
- Create middleware instances
- Configure logging

## Complete API Definition Example

```go
// user.api
syntax = "v1"

info(
    title: "User API"
    desc: "User management API"
    author: "go-zero"
    version: "v1"
)

type (
    CreateUserRequest {
        Name     string `json:"name" validate:"required"`
        Email    string `json:"email" validate:"required,email"`
        Password string `json:"password" validate:"required,min=8"`
    }

    CreateUserResponse {
        Id int64 `json:"id"`
    }

    GetUserRequest {
        Id int64 `path:"id"`
    }

    GetUserResponse {
        Id    int64  `json:"id"`
        Name  string `json:"name"`
        Email string `json:"email"`
    }

    UpdateUserRequest {
        Id   int64  `path:"id"`
        Name string `json:"name,optional"`
    }

    DeleteUserRequest {
        Id int64 `path:"id"`
    }
)

@server(
    prefix: /api/v1
    group: user
    middleware: Auth
)
service user-api {
    @doc "Create a new user"
    @handler CreateUser
    post /users (CreateUserRequest) returns (CreateUserResponse)

    @doc "Get user by ID"
    @handler GetUser
    get /users/:id (GetUserRequest) returns (GetUserResponse)

    @doc "Update user"
    @handler UpdateUser
    put /users/:id (UpdateUserRequest)

    @doc "Delete user"
    @handler DeleteUser
    delete /users/:id (DeleteUserRequest)
}
```

## Best Practices Summary

### ✅ DO:
- Keep handlers thin - only HTTP concerns
- Put all business logic in logic layer
- Use `httpx.ErrorCtx` and `httpx.OkJsonCtx` for responses
- Always pass and use `context.Context`
- Embed `rest.RestConf` in config structs
- Define clear request/response types
- Use structured logging with `logx`
- Handle errors properly with wrapping
- Initialize dependencies in service context

### ❌ DON'T:
- Put business logic in handlers
- Use `httpx.Error` without context (use `ErrorCtx`)
- Ignore context in database operations
- Use `any` type in API definitions
- Create global variables for dependencies
- Log sensitive information (passwords, tokens)
- Ignore errors or use `_` carelessly
- Make handlers do multiple responsibilities

## When to Use This Pattern

Use the standard three-layer REST pattern for:
- CRUD APIs
- RESTful web services
- API gateways
- Backend-for-frontend (BFF) services
- Microservice APIs

For RPC services, see [RPC Patterns](./rpc-patterns.md).
For complex workflows, see [Service Orchestration Patterns](./orchestration-patterns.md).
