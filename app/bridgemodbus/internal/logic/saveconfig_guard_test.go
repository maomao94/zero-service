package logic

import (
	"context"
	"testing"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"
	"zero-service/common/gormx"
	"zero-service/common/modbusx"
	"zero-service/model/gormmodel"

	"gorm.io/gorm"
)

func newModbusConfigTestServiceContext(t *testing.T) (*gormx.DB, *svc.ServiceContext) {
	t.Helper()
	db, err := gormx.Open("file:" + t.Name() + "?mode=memory&cache=shared&parseTime=true")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if raw, err := db.DB.DB(); err == nil {
			_ = raw.Close()
		}
	})
	if err := db.AutoMigrate(&gormmodel.ModbusSlaveConfig{}); err != nil {
		t.Fatal(err)
	}
	manager := modbusx.NewClientManager()
	t.Cleanup(manager.Close)
	return db, &svc.ServiceContext{
		DB:                    db,
		Manager:               manager,
		ModbusConfigConverter: gormmodel.NewModbusConfigConverter(),
	}
}

func createModbusConfig(t *testing.T, db *gormx.DB) gormmodel.ModbusSlaveConfig {
	t.Helper()
	cfg := gormmodel.ModbusSlaveConfig{
		ModbusCode:   "device",
		SlaveAddress: "127.0.0.1:1",
		Slave:        2,
		Timeout:      3000,
		IdleTimeout:  60000,
	}
	cfg.Id = "cfg-pk"
	if err := db.Create(&cfg).Error; err != nil {
		t.Fatal(err)
	}
	return cfg
}

// injectModbusDeleteAfterLoad 在保存逻辑读到配置之后、写回之前，模拟同事务内的并发删除。
// 用语句所在的事务句柄执行，避免 sqlite 内存库在跨连接写时整库锁表。
func injectModbusDeleteAfterLoad(t *testing.T, db *gorm.DB, id string) {
	t.Helper()
	deleted := false
	err := db.Callback().Query().After("gorm:query").Register("test:delete_modbus_config_after_load", func(tx *gorm.DB) {
		if deleted || tx.Statement.Table != "modbus_slave_config" {
			return
		}
		deleted = true
		if err := tx.Session(&gorm.Session{NewDB: true}).Delete(&gormmodel.ModbusSlaveConfig{}, "id = ?", id).Error; err != nil {
			t.Errorf("注入并发删除失败: %v", err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSaveConfigUpdateKeepsUntouchedColumns(t *testing.T) {
	db, svcCtx := newModbusConfigTestServiceContext(t)
	cfg := createModbusConfig(t, db)

	if _, err := NewSaveConfigLogic(context.Background(), svcCtx).SaveConfig(&bridgemodbus.SaveConfigReq{
		ModbusCode:   "device",
		SlaveAddress: "127.0.0.1:2",
		Slave:        3,
	}); err != nil {
		t.Fatal(err)
	}

	var got gormmodel.ModbusSlaveConfig
	if err := db.Where("id = ?", cfg.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.SlaveAddress != "127.0.0.1:2" || got.Slave != 3 {
		t.Fatalf("slaveAddress/slave = %q/%d, want 127.0.0.1:2/3", got.SlaveAddress, got.Slave)
	}
	// 请求没有携带的列不能被旧快照覆盖。
	if got.Timeout != 3000 || got.IdleTimeout != 60000 {
		t.Fatalf("timeout/idleTimeout = %d/%d, want 3000/60000", got.Timeout, got.IdleTimeout)
	}
}

func TestSaveConfigOnConcurrentlyDeletedConfigDoesNotResurrect(t *testing.T) {
	db, svcCtx := newModbusConfigTestServiceContext(t)
	cfg := createModbusConfig(t, db)
	injectModbusDeleteAfterLoad(t, db.DB, cfg.Id)

	saved, err := NewSaveConfigLogic(context.Background(), svcCtx).SaveConfig(&bridgemodbus.SaveConfigReq{
		ModbusCode:   "device",
		SlaveAddress: "127.0.0.1:2",
		Slave:        3,
	})
	if err != nil {
		t.Fatalf("SaveConfig() error = %v, want nil（0 行按幂等保存成功处理）", err)
	}
	if saved.Id != cfg.Id {
		t.Fatalf("SaveConfig() id = %q, want %q", saved.Id, cfg.Id)
	}

	// 并发删除的记录不能被保存逻辑复活，也不能被写入新值。
	var got gormmodel.ModbusSlaveConfig
	if err := db.Unscoped().Where("id = ?", cfg.Id).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if !got.Deleted() {
		t.Fatal("并发删除的 Modbus 配置被保存逻辑复活")
	}
	if got.SlaveAddress != "127.0.0.1:1" || got.Slave != 2 {
		t.Fatalf("slaveAddress/slave = %q/%d, want 127.0.0.1:1/2", got.SlaveAddress, got.Slave)
	}
}
