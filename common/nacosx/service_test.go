package nacosx

import (
	"testing"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"google.golang.org/grpc/resolver"
)

func TestServiceRegistrationOptions(t *testing.T) {
	t.Setenv(envPodIP, "10.0.0.5")

	c := ServiceConfig{Host: "127.0.0.1", Port: 8848, NamespaceId: "test"}
	rpc, err := c.buildOptions("openapi.rpc", "0.0.0.0:21018", "0.0.0.0:21019")
	if err != nil {
		t.Fatal(err)
	}
	if rpc.AdvertiseIP != "10.0.0.5" {
		t.Fatalf("unexpected advertise IP: %q", rpc.AdvertiseIP)
	}
	if rpc.ServiceName != "openapi.rpc" || rpc.ClientConfig.NamespaceId != "test" {
		t.Fatalf("unexpected registration options: %s, %s", rpc.ServiceName, rpc.ClientConfig.NamespaceId)
	}
	if rpc.ListenOn != "0.0.0.0:21018" || rpc.Metadata["gRPC_port"] != "21019" {
		t.Fatalf("unexpected dual-port registration: %s, %v", rpc.ListenOn, rpc.Metadata)
	}
	addrs := extractHealthyGRPCInstances([]model.Instance{
		{Ip: "127.0.0.1", Port: 21018, Healthy: true, Enable: true, Metadata: rpc.Metadata},
		{Ip: "127.0.0.2", Port: 21020, Healthy: true, Enable: true},
	})
	want := map[string]bool{"127.0.0.1:21019": true, "127.0.0.2:21020": true}
	if len(addrs) != len(want) || resolver.Get("nacos") == nil {
		t.Fatalf("unexpected resolved addresses: %v", addrs)
	}
	for _, addr := range addrs {
		if !want[addr] {
			t.Fatalf("unexpected resolved address %q", addr)
		}
	}

	c.ServiceName = "gateway-custom"
	http, err := c.buildOptions("opengtw", "0.0.0.0:11005", "")
	if err != nil {
		t.Fatal(err)
	}
	if http.ServiceName != "gateway-custom" || http.ListenOn != "0.0.0.0:11005" || http.Metadata["gRPC_port"] != "" {
		t.Fatalf("unexpected HTTP registration options: %s, %s, %v", http.ServiceName, http.ListenOn, http.Metadata)
	}
}

func TestServiceConfigWithMetadata(t *testing.T) {
	c := ServiceConfig{Metadata: map[string]string{"deployMode": "standalone", "owner": "ops"}}
	merged := c.WithMetadata(map[string]string{"deployMode": "cluster", "node": "n1"})

	if merged.Metadata["deployMode"] != "cluster" || merged.Metadata["owner"] != "ops" ||
		merged.Metadata["node"] != "n1" {
		t.Fatalf("unexpected merged metadata: %v", merged.Metadata)
	}
	if c.Metadata["deployMode"] != "standalone" || len(c.Metadata) != 2 {
		t.Fatalf("WithMetadata mutated the original config: %v", c.Metadata)
	}
	if got := c.WithMetadata(nil); len(got.Metadata) != 2 {
		t.Fatalf("unexpected metadata after empty merge: %v", got.Metadata)
	}
}

func TestServiceRegistrationKeepsConfiguredMetadata(t *testing.T) {
	t.Setenv(envPodIP, "10.0.0.5")

	c := ServiceConfig{Metadata: map[string]string{"owner": "ops"}}
	opts, err := c.buildOptions("gis.rpc", "0.0.0.0:25006", "0.0.0.0:25006")
	if err != nil {
		t.Fatal(err)
	}
	if opts.Metadata["owner"] != "ops" || opts.Metadata["gRPC_port"] != "25006" {
		t.Fatalf("configured metadata was not preserved: %v", opts.Metadata)
	}
}
