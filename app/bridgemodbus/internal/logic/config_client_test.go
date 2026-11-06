package logic

import (
	"context"
	"testing"

	"zero-service/app/bridgemodbus/bridgemodbus"
	"zero-service/app/bridgemodbus/internal/svc"
	"zero-service/common/errx"
	"zero-service/common/gormx"
	"zero-service/common/modbusx"
	"zero-service/model/gormmodel"
	"zero-service/third_party/extproto"
)

// 保存/删除配置成功后必须清掉该 code 的缓存客户端，下一次取用重建或报错。
func TestSaveAndDeleteConfigInvalidateCachedClient(t *testing.T) {
	db, err := gormx.Open("file:config_client_test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := db.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	db.MustAutoMigrate(&gormmodel.ModbusSlaveConfig{})

	manager := modbusx.NewClientManager()
	defer manager.Close()
	svcCtx := &svc.ServiceContext{
		DB:                    db,
		Manager:               manager,
		ModbusConfigConverter: gormmodel.NewModbusConfigConverter(),
	}
	ctx := context.Background()

	saved, err := NewSaveConfigLogic(ctx, svcCtx).SaveConfig(&bridgemodbus.SaveConfigReq{
		ModbusCode: "device", SlaveAddress: "127.0.0.1:1", Slave: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := svcCtx.GetModbusClient(ctx, "device")
	if err != nil {
		t.Fatal(err)
	}

	// 再次保存：旧客户端必须失效，下一次取用重建。
	if _, err := NewSaveConfigLogic(ctx, svcCtx).SaveConfig(&bridgemodbus.SaveConfigReq{
		ModbusCode: "device", SlaveAddress: "127.0.0.1:1", Slave: 3,
	}); err != nil {
		t.Fatal(err)
	}
	second, err := svcCtx.GetModbusClient(ctx, "device")
	if err != nil {
		t.Fatal(err)
	}
	if second == first {
		t.Fatal("SaveConfig 未清除缓存客户端")
	}

	// 删除后不能继续返回缓存客户端。
	if _, err := NewDeleteConfigLogic(ctx, svcCtx).DeleteConfig(&bridgemodbus.DeleteConfigReq{
		Ids: []string{saved.Id},
	}); err != nil {
		t.Fatal(err)
	}
	cli, err := svcCtx.GetModbusClient(ctx, "device")
	if err == nil || cli != nil {
		t.Fatalf("DeleteConfig 后仍返回客户端: cli=%v err=%v", cli, err)
	}
	if code, _ := errx.CodeFromStatus(err); code != int(extproto.Code__1_00_RECORD_NOT_EXIST) {
		t.Fatalf("code = %d, want %d: %v", code, extproto.Code__1_00_RECORD_NOT_EXIST, err)
	}
}
