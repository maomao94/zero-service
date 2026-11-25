package config

import (
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/zrpc"
)

func TestNacosDiscoveryConfig(t *testing.T) {
	var c Config
	if err := conf.LoadFromYamlBytes([]byte(`
Name: openapi.rpc
ListenOn: 0.0.0.0:21018
NacosConfig:
  IsRegister: true
PodEngineConf:
  Target: nacos://127.0.0.1:8848/pod-engine?namespaceid=public
BridgeModbusConf:
  Target: nacos://127.0.0.1:8848/bridge-modbus?namespaceid=public
GisConf:
  Target: nacos://127.0.0.1:8848/gis?namespaceid=public
`), &c); err != nil {
		t.Fatal(err)
	}
	if !c.NacosConfig.IsRegister || c.NacosConfig.Port != 8848 || c.GisConf == nil {
		t.Fatal("Nacos registration or GIS config was not loaded")
	}
	for _, downstream := range []zrpc.RpcClientConf{c.PodEngineConf, c.BridgeModbusConf, *c.GisConf} {
		target, err := downstream.BuildTarget()
		if err != nil || downstream.Target == "" || target != downstream.Target {
			t.Fatalf("unexpected discovery target %q: %v", target, err)
		}
	}
}
