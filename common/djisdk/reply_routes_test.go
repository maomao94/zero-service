package djisdk

import (
	"testing"
	"time"
)

func TestReplyRoutersWithoutTTLReturnsNoRoutes(t *testing.T) {
	routes, opts := replyRouters(0)

	if routes.services != nil || routes.propertySet != nil {
		t.Fatalf("routes = %+v, want empty routes when TTL <= 0", routes)
	}
	if len(opts) != 0 {
		t.Fatalf("client options = %d, want 0", len(opts))
	}
}

func TestReplyRoutersBuildsBothRoutes(t *testing.T) {
	routes, opts := replyRouters(time.Second)
	if routes.services == nil || routes.propertySet == nil {
		t.Fatalf("routes = %+v, want both reply routers", routes)
	}
	defer routes.services.Close()
	defer routes.propertySet.Close()

	if routes.services == routes.propertySet {
		t.Fatal("services and property_set reply routers must be independent")
	}
	if len(opts) != 2 {
		t.Fatalf("client options = %d, want 2 registrations", len(opts))
	}
}
