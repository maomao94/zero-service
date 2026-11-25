package nacosx

import (
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"google.golang.org/grpc/resolver"
)

func TestServiceRegistrationOptions(t *testing.T) {
	c := ServiceConfig{Host: "127.0.0.1", Port: 8848, NamespaceId: "test"}
	rpc, err := c.registrationOptions("openapi.rpc", "0.0.0.0:21018", true)
	if err != nil {
		t.Fatal(err)
	}
	if rpc.ServiceName != "openapi.rpc" || rpc.ClientConfig.NamespaceId != "test" {
		t.Fatalf("unexpected registration options: %s, %s", rpc.ServiceName, rpc.ClientConfig.NamespaceId)
	}
	addrs := extractHealthyGRPCInstances([]model.Instance{{Ip: "127.0.0.1", Healthy: true, Enable: true, Metadata: rpc.Metadata}})
	if len(addrs) != 1 || addrs[0] != "127.0.0.1:21018" || resolver.Get("nacos") == nil {
		t.Fatalf("gRPC registration does not match the Nacos resolver: %v", addrs)
	}
	c.ServiceName = "gateway-custom"
	http, err := c.registrationOptions("opengtw", "0.0.0.0:11005", false)
	if err != nil {
		t.Fatal(err)
	}
	if http.ServiceName != "gateway-custom" || http.ListenOn != "0.0.0.0:11005" || http.Metadata["gRPC_port"] != "" {
		t.Fatalf("unexpected HTTP registration options: %s, %s, %v", http.ServiceName, http.ListenOn, http.Metadata)
	}
}
