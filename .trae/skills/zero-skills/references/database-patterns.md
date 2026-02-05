# Database Patterns

## SQL Database with go-zero

go-zero provides `sqlx` and `sqlc` packages for SQL operations with built-in connection pooling, caching, and resilience.

## Basic SQL Operations Pattern

### ✅ Model Generation from SQL

```bash
# Generate model from existing database
goctl model mysql datasource \
  -url="user:pass@tcp(localhost:3306)/database" \
  -table="users" \
  -dir="./model"

# Generate model from SQL DDL file
goctl model mysql ddl \
  -src="./schema.sql" \
  -dir="./model"
```

### Example SQL Schema

```sql
CREATE TABLE `users` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `email` varchar(255) NOT NULL UNIQUE,
  `age` int NOT NULL DEFAULT 0,
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### Generated Model Structure

```go
// model/usersmodel.go
package model

import (
    "context"
    "database/sql"
    "github.com/zeromicro/go-zero/core/stores/cache"
    "github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UsersModel = (*customUsersModel)(nil)

type (
    // Interface for Users model operations
    UsersModel interface {
        usersModel
        // Add custom methods here
    }

    customUsersModel struct {
        *defaultUsersModel
    }

    // Generated struct
    Users struct {
        Id        int64          `db:"id"`
        Name      string         `db:"name"`
        Email     string         `db:"email"`
        Age       int64          `db:"age"`
        CreatedAt sql.NullTime   `db:"created_at"`
        UpdatedAt sql.NullTime   `db:"updated_at"`
    }
)

// NewUsersModel returns a model for Users
func NewUsersModel(conn sqlx.SqlConn, c cache.CacheConf) UsersModel {
    return &customUsersModel{
        defaultUsersModel: newUsersModel(conn, c),
    }
}

// Generated methods (in usersmodel_gen.go):
// - Insert(ctx context.Context, data *Users) (sql.Result, error)
// - FindOne(ctx context.Context, id int64) (*Users, error)
// - FindOneByEmail(ctx context.Context, email string) (*Users, error)
// - Update(ctx context.Context, data *Users) error
// - Delete(ctx context.Context, id int64) error
```

## CRUD Operations Pattern

### ✅ Insert

```go
func (l *CreateUserLogic) CreateUser(req *types.CreateUserRequest) (*types.CreateUserResponse, error) {
    user := &model.Users{
        Name:  req.Name,
        Email: req.Email,
        Age:   int64(req.Age),
    }

    result, err := l.svcCtx.UsersModel.Insert(l.ctx, user)
    if err != nil {
        l.Logger.Errorf("failed to insert user: %v", err)
        return nil, err
    }

    userId, err := result.LastInsertId()
    if err != nil {
        return nil, err
    }

    return &types.CreateUserResponse{
        Id: userId,
    }, nil
}
```

### ✅ Find by Primary Key

```go
func (l *GetUserLogic) GetUser(req *types.GetUserRequest) (*types.GetUserResponse, error) {
    user, err := l.svcCtx.UsersModel.FindOne(l.ctx, req.Id)
    if err != nil {
        if errors.Is(err, model.ErrNotFound) {
            return nil, errors.New("user not found")
        }
        return nil, err
    }

    return &types.GetUserResponse{
        Id:    user.Id,
        Name:  user.Name,
        Email: user.Email,
        Age:   int(user.Age),
    }, nil
}
```

### ✅ Find by Unique Index

```go
func (l *GetUserByEmailLogic) GetUserByEmail(email string) (*model.Users, error) {
    user, err := l.svcCtx.UsersModel.FindOneByEmail(l.ctx, email)
    if err != nil {
        if errors.Is(err, model.ErrNotFound) {
            return nil, errors.New("user not found")
        }
        return nil, err
    }
    return user, nil
}
```

### ✅ Update

```go
func (l *UpdateUserLogic) UpdateUser(req *types.UpdateUserRequest) error {
    // Find existing user first
    user, err := l.svcCtx.UsersModel.FindOne(l.ctx, req.Id)
    if err != nil {
        return err
    }

    // Update fields
    if req.Name != "" {
        user.Name = req.Name
    }
    if req.Age > 0 {
        user.Age = int64(req.Age)
    }

    // Save changes
    err = l.svcCtx.UsersModel.Update(l.ctx, user)
    if err != nil {
        l.Logger.Errorf("failed to update user: %v", err)
        return err
    }

    return nil
}
```

### ✅ Delete

```go
func (l *DeleteUserLogic) DeleteUser(req *types.DeleteUserRequest) error {
    err := l.svcCtx.UsersModel.Delete(l.ctx, req.Id)
    if err != nil {
        l.Logger.Errorf("failed to delete user: %v", err)
        return err
    }
    return nil
}
```

## Custom Query Pattern

### ✅ Add Custom Methods to Model

```go
// model/usersmodel.go
type (
    UsersModel interface {
        usersModel
        // Custom methods
        FindByAgeRange(ctx context.Context, minAge, maxAge int64) ([]*Users, error)
        FindActiveUsers(ctx context.Context, limit int64) ([]*Users, error)
        CountByAge(ctx context.Context, age int64) (int64, error)
    }

    customUsersModel struct {
        *defaultUsersModel
    }
)

func (m *customUsersModel) FindByAgeRange(ctx context.Context, minAge, maxAge int64) ([]*Users, error) {
    query := `SELECT * FROM users WHERE age BETWEEN ? AND ? ORDER BY created_at DESC`
    var users []*Users
    err := m.QueryRowsNoCacheCtx(ctx, &users, query, minAge, maxAge)
    if err != nil {
        return nil, err
    }
    return users, nil
}

func (m *customUsersModel) FindActiveUsers(ctx context.Context, limit int64) ([]*Users, error) {
    query := `SELECT * FROM users WHERE updated_at > DATE_SUB(NOW(), INTERVAL 30 DAY) LIMIT ?`
    var users []*Users
    err := m.QueryRowsNoCacheCtx(ctx, &users, query, limit)
    if err != nil {
        return nil, err
    }
    return users, nil
}

func (m *customUsersModel) CountByAge(ctx context.Context, age int64) (int64, error) {
    query := `SELECT COUNT(*) FROM users WHERE age = ?`
    var count int64
    err := m.QueryRowNoCacheCtx(ctx, &count, query, age)
    return count, err
}
```

### ✅ Pagination Pattern

```go
func (m *customUsersModel) FindWithPagination(ctx context.Context, page, pageSize int64) ([]*Users, int64, error) {
    // Get total count
    var total int64
    countQuery := `SELECT COUNT(*) FROM users`
    err := m.QueryRowNoCacheCtx(ctx, &total, countQuery)
    if err != nil {
        return nil, 0, err
    }

    // Get paginated results
    offset := (page - 1) * pageSize
    query := `SELECT * FROM users ORDER BY id DESC LIMIT ? OFFSET ?`
    var users []*Users
    err = m.QueryRowsNoCacheCtx(ctx, &users, query, pageSize, offset)
    if err != nil {
        return nil, 0, err
    }

    return users, total, nil
}
```

## Transaction Pattern

### ✅ Simple Transaction

```go
func (l *TransferLogic) Transfer(from, to int64, amount float64) error {
    // Start transaction
    err := l.svcCtx.DB.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
        // Debit from account
        debitQuery := `UPDATE accounts SET balance = balance - ? WHERE id = ? AND balance >= ?`
        result, err := session.ExecCtx(ctx, debitQuery, amount, from, amount)
        if err != nil {
            return err
        }

        affected, err := result.RowsAffected()
        if err != nil {
            return err
        }
        if affected == 0 {
            return errors.New("insufficient balance")
        }

        // Credit to account
        creditQuery := `UPDATE accounts SET balance = balance + ? WHERE id = ?`
        _, err = session.ExecCtx(ctx, creditQuery, amount, to)
        if err != nil {
            return err
        }

        // Record transaction
        recordQuery := `INSERT INTO transactions(from_id, to_id, amount) VALUES(?, ?, ?)`
        _, err = session.ExecCtx(ctx, recordQuery, from, to, amount)
        return err
    })

    return err
}
```

### ✅ Complex Transaction with Multiple Models

```go
func (l *CreateOrderLogic) CreateOrder(req *types.CreateOrderRequest) (*types.CreateOrderResponse, error) {
    var orderId int64

    err := l.svcCtx.DB.TransactCtx(l.ctx, func(ctx context.Context, session sqlx.Session) error {
        // 1. Create order
        orderQuery := `INSERT INTO orders(user_id, total_amount, status) VALUES(?, ?, ?)`
        result, err := session.ExecCtx(ctx, orderQuery, req.UserId, req.TotalAmount, "pending")
        if err != nil {
            return fmt.Errorf("failed to create order: %w", err)
        }

        orderId, err = result.LastInsertId()
        if err != nil {
            return err
        }

        // 2. Create order items
        itemQuery := `INSERT INTO order_items(order_id, product_id, quantity, price) VALUES(?, ?, ?, ?)`
        for _, item := range req.Items {
            _, err = session.ExecCtx(ctx, itemQuery, orderId, item.ProductId, item.Quantity, item.Price)
            if err != nil {
                return fmt.Errorf("failed to create order item: %w", err)
            }
        }

        // 3. Update inventory
        inventoryQuery := `UPDATE products SET stock = stock - ? WHERE id = ? AND stock >= ?`
        for _, item := range req.Items {
            result, err = session.ExecCtx(ctx, inventoryQuery, item.Quantity, item.ProductId, item.Quantity)
            if err != nil {
                return fmt.Errorf("failed to update inventory: %w", err)
            }

            affected, _ := result.RowsAffected()
            if affected == 0 {
                return fmt.Errorf("insufficient stock for product %d", item.ProductId)
            }
        }

        return nil
    })

    if err != nil {
        l.Logger.Errorf("transaction failed: %v", err)
        return nil, err
    }

    return &types.CreateOrderResponse{
        OrderId: orderId,
    }, nil
}
```

## Caching Pattern

### ✅ Cache Configuration

```yaml
# Configuration file
Cache:
  - Host: localhost:6379
    Type: node
    Pass: ""  # Redis password (optional)
  # For Redis cluster
  # - Host: localhost:6379,localhost:6380,localhost:6381
  #   Type: cluster
```

```go
// Configuration struct
type Config struct {
    rest.RestConf
    DataSource string
    Cache      cache.CacheConf
}
```

### ✅ Model with Cache

When you use `NewUsersModel(conn, c.Cache)`, caching is automatic for:
- `FindOne` - Cached by primary key
- `FindOneByXxx` - Cached by unique index
- `Update` / `Delete` - Automatically invalidates cache

```go
// Service context with cache
func NewServiceContext(c config.Config) *ServiceContext {
    conn := sqlx.NewMysql(c.DataSource)

    return &ServiceContext{
        Config:     c,
        UsersModel: model.NewUsersModel(conn, c.Cache),  // ✅ Cache enabled
    }
}
```

### ✅ Custom Cache Keys

```go
func (m *customUsersModel) FindByEmailWithCache(ctx context.Context, email string) (*Users, error) {
    // Custom cache key
    cacheKey := fmt.Sprintf("user:email:%s", email)

    var user Users
    err := m.QueryRowCtx(ctx, &user, cacheKey, func(ctx context.Context, conn sqlx.SqlConn, v interface{}) error {
        query := `SELECT * FROM users WHERE email = ? LIMIT 1`
        return conn.QueryRowCtx(ctx, v, query, email)
    })

    if err != nil {
        return nil, err
    }

    return &user, nil
}
```

### ✅ Manual Cache Operations

```go
// Get from cache
var user Users
err := m.CachedConn.GetCacheCtx(ctx, "user:123", &user)

// Set cache with expiration
err := m.CachedConn.SetCacheCtx(ctx, "user:123", user, time.Hour)

// Delete cache
err := m.CachedConn.DelCacheCtx(ctx, "user:123")

// Delete multiple cache keys
err := m.CachedConn.DelCacheCtx(ctx, "user:123", "user:email:test@test.com")
```

## Connection Pooling Pattern

### ✅ Default Pool Configuration

go-zero uses sensible defaults:
```go
// Default connection pool settings
MaxIdleConns: 64
MaxOpenConns: 64
ConnMaxLifetime: time.Minute
```

### ✅ Custom Pool Configuration

```go
func NewServiceContext(c config.Config) *ServiceContext {
    // Create connection with custom settings
    conn := sqlx.NewMysql(c.DataSource)

    // Customize pool (if needed)
    db, err := conn.RawDB()
    if err == nil {
        db.SetMaxIdleConns(100)
        db.SetMaxOpenConns(100)
        db.SetConnMaxLifetime(time.Minute * 5)
    }

    return &ServiceContext{
        Config:     c,
        UsersModel: model.NewUsersModel(conn, c.Cache),
    }
}
```

## Error Handling Pattern

### ✅ Handle Common Errors

```go
import (
    "github.com/zeromicro/go-zero/core/stores/sqlc"
)

func (l *GetUserLogic) GetUser(req *types.GetUserRequest) (*types.GetUserResponse, error) {
    user, err := l.svcCtx.UsersModel.FindOne(l.ctx, req.Id)
    if err != nil {
        // Check for not found
        if errors.Is(err, sqlc.ErrNotFound) {
            return nil, errors.New("user not found")
        }

        // Check for database errors
        if errors.Is(err, sql.ErrConnDone) {
            l.Logger.Error("database connection error")
            return nil, errors.New("database connection error")
        }

        // Generic error
        l.Logger.Errorf("failed to find user: %v", err)
        return nil, err
    }

    return &types.GetUserResponse{
        Id:    user.Id,
        Name:  user.Name,
        Email: user.Email,
    }, nil
}
```

### ✅ Handle Duplicate Key Errors

```go
import (
    "github.com/go-sql-driver/mysql"
)

func (l *CreateUserLogic) CreateUser(req *types.CreateUserRequest) (*types.CreateUserResponse, error) {
    user := &model.Users{
        Name:  req.Name,
        Email: req.Email,
    }

    result, err := l.svcCtx.UsersModel.Insert(l.ctx, user)
    if err != nil {
        // Check for duplicate key error
        if mysqlErr, ok := err.(*mysql.MySQLError); ok {
            if mysqlErr.Number == 1062 { // Duplicate entry
                return nil, errors.New("email already exists")
            }
        }
        return nil, err
    }

    userId, _ := result.LastInsertId()
    return &types.CreateUserResponse{Id: userId}, nil
}
```

## MongoDB Pattern

### ✅ MongoDB Configuration

```yaml
Mongo:
  Host: localhost:27017
  Type: mongo  # or "mongos" for sharded cluster
  User: username
  Pass: password
  Db: mydb
```

```go
type Config struct {
    rest.RestConf
    Mongo struct {
        Host string
        Type string
        User string `json:",optional"`
        Pass string `json:",optional"`
        Db   string
    }
}
```

### ✅ MongoDB Model

```go
// model/usermodel.go
package model

import (
    "context"
    "github.com/zeromicro/go-zero/core/stores/mon"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
    ID       primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
    Name     string             `bson:"name" json:"name"`
    Email    string             `bson:"email" json:"email"`
    Age      int                `bson:"age" json:"age"`
    CreateAt int64              `bson:"create_at" json:"create_at"`
    UpdateAt int64              `bson:"update_at" json:"update_at"`
}

type UserModel interface {
    Insert(ctx context.Context, user *User) error
    FindOne(ctx context.Context, id string) (*User, error)
    FindOneByEmail(ctx context.Context, email string) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
}

type defaultUserModel struct {
    conn *mon.Model
}

func NewUserModel(url, db, collection string) UserModel {
    return &defaultUserModel{
        conn: mon.MustNewModel(url, db, collection),
    }
}

func (m *defaultUserModel) Insert(ctx context.Context, user *User) error {
    user.ID = primitive.NewObjectID()
    _, err := m.conn.InsertOne(ctx, user)
    return err
}

func (m *defaultUserModel) FindOne(ctx context.Context, id string) (*User, error) {
    oid, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        return nil, err
    }

    var user User
    err = m.conn.FindOne(ctx, &user, bson.M{"_id": oid})
    return &user, err
}

func (m *defaultUserModel) FindOneByEmail(ctx context.Context, email string) (*User, error) {
    var user User
    err := m.conn.FindOne(ctx, &user, bson.M{"email": email})
    return &user, err
}

func (m *defaultUserModel) Update(ctx context.Context, user *User) error {
    oid, err := primitive.ObjectIDFromHex(user.ID.Hex())
    if err != nil {
        return err
    }

    _, err = m.conn.UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": user})
    return err
}

func (m *defaultUserModel) Delete(ctx context.Context, id string) error {
    oid, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        return err
    }

    _, err = m.conn.DeleteOne(ctx, bson.M{"_id": oid})
    return err
}
```

## Best Practices Summary

### ✅ DO:
- Use `goctl` to generate models from database schema
- Always pass `context.Context` to database operations
- Use transactions for operations that must be atomic
- Enable caching for read-heavy models
- Handle `sqlc.ErrNotFound` explicitly
- Use connection pooling (automatic by default)
- Add custom methods to model interface
- Log database errors with context
- Use parameterized queries (automatic with go-zero)
- Validate data before database operations

### ❌ DON'T:
- Execute raw SQL without parameterization
- Ignore errors from database operations
- Use `_` to discard errors
- Create database connections in handlers/logic
- Keep transactions open longer than necessary
- Query in loops (use batch operations)
- Store sensitive data unencrypted
- Use `SELECT *` in production code (be explicit)
- Cache write-heavy data unnecessarily
- Forget to close result sets/cursors

## When to Use Each Database Type

### MySQL/PostgreSQL (SQL):
- Structured data with relationships
- ACID transactions required
- Complex queries with JOINs
- Strong consistency needed
- Traditional CRUD operations

### MongoDB:
- Flexible schema
- Horizontal scaling
- Document-oriented data
- High write throughput
- Hierarchical data

### Redis (Cache):
- Session storage
- Rate limiting
- Real-time leaderboards
- Pub/sub messaging
- Hot data caching

For Redis-specific patterns, see [Caching Strategies](./caching-strategies.md).
