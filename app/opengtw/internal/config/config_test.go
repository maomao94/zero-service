package config

import (
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
)

func TestNacosDiscoveryConfig(t *testing.T) {
	var c Config
	if err := conf.LoadFromYamlBytes([]byte(`
Name: opengtw
Port: 11005
JwtAuth:
  AccessSecret: example-only
NacosConfig:
  IsRegister: true
  ServiceName: opengtw
OpenApiConf:
  Target: nacos://127.0.0.1:8848/openapi.rpc?namespaceid=public
`), &c); err != nil {
		t.Fatal(err)
	}
	if !c.NacosConfig.IsRegister || c.NacosConfig.ServiceName != "opengtw" {
		t.Fatal("Nacos registration config was not loaded")
	}
	target, err := c.OpenApiConf.BuildTarget()
	if err != nil || target != "nacos://127.0.0.1:8848/openapi.rpc?namespaceid=public" {
		t.Fatalf("unexpected discovery target %q: %v", target, err)
	}
}
