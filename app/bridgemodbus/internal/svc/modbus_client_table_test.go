package svc

import (
	"context"
	"errors"
	"testing"
	"zero-service/common/errx"
	"zero-service/common/gormx"
	"zero-service/common/modbusx"
	"zero-service/model/gormmodel"
	"zero-service/third_party/extproto"
)

func TestGetModbusClientTableRoutingAndRemoval(t *testing.T) {
	db, err := gormx.Open("file:" + t.Name() + "?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := db.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	// 仅创建本测试所需的查询列，避免依赖业务模型 ID 生成 hook。
	if err := db.Exec(`CREATE TABLE modbus_slave_config (id TEXT PRIMARY KEY, modbus_code TEXT, slave_address TEXT, slave INTEGER, status INTEGER, idle_timeout INTEGER, connect_delay INTEGER, is_deleted INTEGER DEFAULT 0)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO modbus_slave_config (id, modbus_code, slave_address, slave, status, idle_timeout, connect_delay) VALUES ('1', 'device', '127.0.0.1:1', 2, 1, 60000, 100), ('2', 'disabled', '127.0.0.1:2', 3, 2, 0, 0)`).Error; err != nil {
		t.Fatal(err)
	}
	m := modbusx.NewClientManager()
	defer m.Close()
	s := &ServiceContext{DB: db, Manager: m, ModbusConfigConverter: gormmodel.NewModbusConfigConverter()}
	s.Config.ModbusClientConf = modbusx.ModbusClientConf{Address: "127.0.0.1:3"}
	ctx := context.Background()
	first, err := s.GetModbusClient(ctx, "device")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.GetModbusClient(ctx, "device")
	if err != nil || first != second {
		t.Fatalf("table reuse: %v", err)
	}
	for code, want := range map[string]extproto.Code{
		"missing":  extproto.Code__1_00_RECORD_NOT_EXIST,
		"disabled": extproto.Code__1_00_BIZ,
	} {
		cli, err := s.GetModbusClient(ctx, code)
		if err == nil || cli != nil {
			t.Fatalf("%s fell back to default", code)
		}
		if got, _ := errx.CodeFromStatus(err); got != int(want) {
			t.Fatalf("%s code = %d, want %d", code, got, want)
		}
	}
	// 立即失效路径：SaveConfig / DeleteConfig 成功后调用 RemoveClient，下一次取用重建。
	if err := db.Exec(`UPDATE modbus_slave_config SET slave_address='127.0.0.1:4' WHERE modbus_code='device'`).Error; err != nil {
		t.Fatal(err)
	}
	if !m.RemoveClient("device") {
		t.Fatal("cached client not removed")
	}
	next, err := s.GetModbusClient(ctx, "device")
	if err != nil || next == first {
		t.Fatalf("table reload: %v", err)
	}

	// 配置不存在/未启用用内部标记错误表示，后台校验据此移除客户端。
	for _, code := range []string{"missing", "disabled"} {
		if _, err := s.loadModbusClientConfig(ctx, code); !errors.Is(err, modbusx.ErrConfigUnavailable) {
			t.Fatalf("%s error = %v, want ErrConfigUnavailable", code, err)
		}
	}
	if err := db.Exec(`UPDATE modbus_slave_config SET status=2 WHERE modbus_code='device'`).Error; err != nil {
		t.Fatal(err)
	}
	m.RemoveClient("device")
	if cli, err := s.GetModbusClient(ctx, "device"); err == nil || cli != nil {
		t.Fatal("disabled device returned a client")
	}
	if err := db.Exec(`DELETE FROM modbus_slave_config WHERE modbus_code='device'`).Error; err != nil {
		t.Fatal(err)
	}
	m.RemoveClient("device")
	if cli, err := s.GetModbusClient(ctx, "device"); err == nil || cli != nil {
		t.Fatal("deleted device returned a client")
	}
}

// 非法从站号和管理器关闭都必须带平台码，不能把原始 error 透给 RPC 层。
func TestGetModbusClientMapsClientErrors(t *testing.T) {
	db, err := gormx.Open("file:" + t.Name() + "?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := db.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	if err := db.Exec(`CREATE TABLE modbus_slave_config (id TEXT PRIMARY KEY, modbus_code TEXT, slave_address TEXT, slave INTEGER, status INTEGER, idle_timeout INTEGER, connect_delay INTEGER, is_deleted INTEGER DEFAULT 0)`).Error; err != nil {
		t.Fatal(err)
	}
	// 历史脏数据：从站号 248 会被 NewModbusClient 拒绝；0 必须在加载阶段拒绝，
	// 不能静默套用文件默认的从站 1。
	if err := db.Exec(`INSERT INTO modbus_slave_config (id, modbus_code, slave_address, slave, status) VALUES ('1', 'bad-slave', '127.0.0.1:1', 248, 1), ('2', 'zero-slave', '127.0.0.1:1', 0, 1)`).Error; err != nil {
		t.Fatal(err)
	}

	m := modbusx.NewClientManager()
	defer m.Close()
	s := &ServiceContext{DB: db, Manager: m, ModbusConfigConverter: gormmodel.NewModbusConfigConverter()}

	for _, code := range []string{"bad-slave", "zero-slave"} {
		cli, err := s.GetModbusClient(context.Background(), code)
		if cli != nil || err == nil {
			t.Fatalf("%s returned client=%v err=%v", code, cli, err)
		}
		if got, _ := errx.CodeFromStatus(err); got != int(extproto.Code__1_00_BIZ) {
			t.Fatalf("%s code = %d, want %d: %v", code, got, extproto.Code__1_00_BIZ, err)
		}
		if !errors.Is(err, modbusx.ErrConfigUnavailable) {
			t.Fatalf("%s error = %v, want ErrConfigUnavailable", code, err)
		}
	}

	m.Close()
	if _, err := s.GetModbusClient(context.Background(), "bad-slave"); err == nil {
		t.Fatal("closed manager returned a client")
	} else if code, _ := errx.CodeFromStatus(err); code != int(extproto.Code__1_00_BIZ) {
		t.Fatalf("closed manager code = %d, want %d: %v", code, extproto.Code__1_00_BIZ, err)
	}
}

func TestGetModbusClientDefaultFileConfig(t *testing.T) {
	m := modbusx.NewClientManager()
	defer m.Close()
	s := &ServiceContext{Manager: m}
	s.Config.ModbusClientConf = modbusx.ModbusClientConf{Address: "127.0.0.1:1", Slave: 2}
	// DB 未设置：空 code 路径必须只读取文件配置。
	first, err := s.GetModbusClient(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.GetModbusClient(context.Background(), "")
	if err != nil || first != second {
		t.Fatalf("default reuse: %v", err)
	}
}
