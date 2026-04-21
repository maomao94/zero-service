package gormx

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	ErrMissingDB        = errors.New("missing db in context")
	ErrMissingDependent = errors.New("missing dependent value in context")
	ErrDBTypeMismatch   = errors.New("db type mismatch")
)

const (
	InitOrderSystem   = 10
	InitOrderInternal = 1000
	InitOrderExternal = 100000
)

type Initializer interface {
	Name() string
	TableCreated(ctx context.Context, db *gorm.DB) bool
	MigrateTable(ctx context.Context, db *gorm.DB) (nextCtx context.Context, err error)
	DataInserted(ctx context.Context, db *gorm.DB) bool
	InitializeData(ctx context.Context, db *gorm.DB) (nextCtx context.Context, err error)
}

type orderedInit struct {
	order int
	Initializer
}

type initSlice []*orderedInit

type initDBKey struct{}

var (
	initMu       sync.Mutex
	initializers = initSlice{}
	initCache    = make(map[string]*orderedInit)
)

func RegisterInit(order int, init Initializer) {
	initMu.Lock()
	defer initMu.Unlock()

	name := init.Name()
	if _, ok := initCache[name]; ok {
		logx.Errorf("initializer %s already registered", name)
		return
	}
	initializers = append(initializers, &orderedInit{order: order, Initializer: init})
	initCache[name] = initializers[len(initializers)-1]
	logx.Infof("registered initializer: %s (order: %d)", name, order)
}

func UnregisterInit(name string) {
	initMu.Lock()
	defer initMu.Unlock()

	delete(initCache, name)
	for i, init := range initializers {
		if init.Name() == name {
			initializers = append(initializers[:i], initializers[i+1:]...)
			break
		}
	}
}

func ClearAllInits() {
	initMu.Lock()
	defer initMu.Unlock()

	initializers = initSlice{}
	initCache = make(map[string]*orderedInit)
}

func (a initSlice) Len() int {
	return len(a)
}

func (a initSlice) Less(i, j int) bool {
	return a[i].order < a[j].order
}

func (a initSlice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func AutoInit(db *gorm.DB) error {
	initMu.Lock()
	sorted := make(initSlice, len(initializers))
	copy(sorted, initializers)
	initMu.Unlock()

	if len(sorted) == 0 {
		logx.Info("no initializers registered, skip auto init")
		return nil
	}

	sort.Sort(&sorted)

	ctx := context.Background()
	for _, init := range sorted {
		if init.TableCreated(ctx, db) {
			logx.Infof("table for %s already exists, skip", init.Name())
			continue
		}

		logx.Infof("migrating table for %s...", init.Name())
		var err error
		ctx, err = init.MigrateTable(ctx, db)
		if err != nil {
			return err
		}

		if init.DataInserted(ctx, db) {
			logx.Infof("data for %s already exists, skip", init.Name())
			continue
		}

		logx.Infof("initializing data for %s...", init.Name())
		ctx, err = init.InitializeData(ctx, db)
		if err != nil {
			return err
		}
	}

	logx.Info("auto init completed successfully")
	return nil
}

type TableInitializer struct {
	tableName   string
	models      []any
	initialData []any
	dataFn      func(ctx context.Context, db *gorm.DB) error
}

func NewTableInitializer(tableName string, models ...any) *TableInitializer {
	return &TableInitializer{tableName: tableName, models: models}
}

func (t *TableInitializer) Name() string {
	return t.tableName
}

func (t *TableInitializer) TableCreated(ctx context.Context, db *gorm.DB) bool {
	return db.Migrator().HasTable(t.tableName)
}

func (t *TableInitializer) MigrateTable(ctx context.Context, db *gorm.DB) (context.Context, error) {
	if len(t.models) == 0 {
		return ctx, nil
	}
	return ctx, db.AutoMigrate(t.models...)
}

func (t *TableInitializer) DataInserted(ctx context.Context, db *gorm.DB) bool {
	if len(t.initialData) == 0 && t.dataFn == nil {
		return true
	}
	var count int64
	db.Model(t.initialData[0]).Count(&count)
	return count > 0
}

func (t *TableInitializer) InitializeData(ctx context.Context, db *gorm.DB) (context.Context, error) {
	if t.dataFn != nil {
		return ctx, t.dataFn(ctx, db)
	}
	if len(t.initialData) == 0 {
		return ctx, nil
	}
	return ctx, db.CreateInBatches(t.initialData, 100).Error
}

func (t *TableInitializer) WithData(data ...any) *TableInitializer {
	t.initialData = data
	return t
}

func (t *TableInitializer) WithDataFn(fn func(ctx context.Context, db *gorm.DB) error) *TableInitializer {
	t.dataFn = fn
	return t
}

func (t *TableInitializer) WithModels(models ...any) *TableInitializer {
	t.models = models
	return t
}

type InitHandler interface {
	EnsureDB(ctx context.Context, dbName string, conf Config) (context.Context, *gorm.DB, error)
	InitTables(ctx context.Context, db *gorm.DB, inits initSlice) error
	InitData(ctx context.Context, db *gorm.DB, inits initSlice) error
}

type MysqlInitHandler struct{}

func (h *MysqlInitHandler) EnsureDB(ctx context.Context, dbName string, conf Config) (context.Context, *gorm.DB, error) {
	if dbName == "" {
		return ctx, nil, nil
	}
	dsn := conf.DataSource
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   QuietGormLogger(),
	})
	if err != nil {
		return ctx, nil, err
	}
	createSQL := "CREATE DATABASE IF NOT EXISTS `" + dbName + "` DEFAULT CHARACTER SET utf8mb4 DEFAULT COLLATE utf8mb4_general_ci"
	if err := db.Exec(createSQL).Error; err != nil {
		return ctx, nil, err
	}
	ctx = context.WithValue(ctx, initDBKey{}, db)
	return ctx, db, nil
}

func (h *MysqlInitHandler) InitTables(ctx context.Context, db *gorm.DB, inits initSlice) error {
	return createTables(ctx, db, inits)
}

func (h *MysqlInitHandler) InitData(ctx context.Context, db *gorm.DB, inits initSlice) error {
	return initData(ctx, db, inits)
}

type PostgresInitHandler struct{}

func (h *PostgresInitHandler) EnsureDB(ctx context.Context, dbName string, conf Config) (context.Context, *gorm.DB, error) {
	if dbName == "" {
		return ctx, nil, nil
	}
	dsn := conf.DataSource
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   QuietGormLogger(),
	})
	if err != nil {
		return ctx, nil, err
	}
	createSQL := "CREATE DATABASE \"" + dbName + "\""
	if err := db.Exec(createSQL).Error; err != nil {
		return ctx, nil, err
	}
	ctx = context.WithValue(ctx, initDBKey{}, db)
	return ctx, db, nil
}

func (h *PostgresInitHandler) InitTables(ctx context.Context, db *gorm.DB, inits initSlice) error {
	return createTables(ctx, db, inits)
}

func (h *PostgresInitHandler) InitData(ctx context.Context, db *gorm.DB, inits initSlice) error {
	return initData(ctx, db, inits)
}

type SqliteInitHandler struct{}

func (h *SqliteInitHandler) EnsureDB(ctx context.Context, dbName string, conf Config) (context.Context, *gorm.DB, error) {
	dsn := conf.DataSource
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   QuietGormLogger(),
	})
	if err != nil {
		return ctx, nil, err
	}
	ctx = context.WithValue(ctx, initDBKey{}, db)
	return ctx, db, nil
}

func (h *SqliteInitHandler) InitTables(ctx context.Context, db *gorm.DB, inits initSlice) error {
	return createTables(ctx, db, inits)
}

func (h *SqliteInitHandler) InitData(ctx context.Context, db *gorm.DB, inits initSlice) error {
	return initData(ctx, db, inits)
}

func createTables(ctx context.Context, db *gorm.DB, inits initSlice) error {
	for _, init := range inits {
		if init.TableCreated(ctx, db) {
			continue
		}
		var err error
		ctx, err = init.MigrateTable(ctx, db)
		if err != nil {
			return err
		}
	}
	return nil
}

func initData(ctx context.Context, db *gorm.DB, inits initSlice) error {
	for _, init := range inits {
		if init.DataInserted(ctx, db) {
			logx.Infof("data for %s already exists, skip", init.Name())
			continue
		}
		var err error
		ctx, err = init.InitializeData(ctx, db)
		if err != nil {
			return err
		}
	}
	return nil
}
