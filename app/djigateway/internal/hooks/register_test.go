package hooks

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"zero-service/common/djisdk"

	"github.com/zeromicro/go-zero/core/collection"
)

func TestRegisterDjiClientRegistersHandlersAndOnlineChecker(t *testing.T) {
	onlineCache, err := collection.NewCache(time.Minute)
	if err != nil {
		t.Fatalf("NewCache online error = %v", err)
	}
	progressCache, err := collection.NewCache(time.Minute)
	if err != nil {
		t.Fatalf("NewCache progress error = %v", err)
	}
	client := djisdk.NewClient(nil, djisdk.WithPendingTTL(time.Second), djisdk.WithReplyOptions(djisdk.ReplyOptions{}))

	RegisterDjiClient(client, RegisterDjiClientOptions{
		OnlineCache:         onlineCache,
		FlightProgressCache: progressCache,
	})

	ctx := context.Background()
	statusPayload := []byte(`{"tid":"tid-1","bid":"bid-1","timestamp":1710000000000,"method":"update_topo","data":{"sub_devices":[]}}`)
	if err := client.HandleStatus(ctx, statusPayload, djisdk.StatusTopic("gateway-1"), ""); err != nil {
		t.Fatalf("HandleStatus() error = %v", err)
	}
	if !IsOnline(onlineCache, "gateway-1") {
		t.Fatal("expected status handler to refresh online cache")
	}
	if _, err := client.SendCommand(ctx, "offline-gateway", djisdk.MethodReturnHome, nil); err == nil {
		t.Fatal("expected offline checker to reject unknown gateway")
	} else if err.Error() != "[dji-sdk] device offline: sn=offline-gateway, command rejected" {
		t.Fatalf("SendCommand() error = %v, want offline checker rejection", err)
	}

	progressPayload := []byte(`{"tid":"tid-2","bid":"bid-2","gateway":"gateway-1","need_reply":0,"method":"flighttask_progress","data":{"ext":{"flight_id":"flight-1","wayline_mission_state":1,"current_waypoint_index":2,"media_count":3,"track_id":"track-1"}}}`)
	if err := client.HandleEvents(ctx, progressPayload, djisdk.EventsTopic("gateway-1"), ""); err != nil {
		t.Fatalf("HandleEvents() error = %v", err)
	}
	found, _, progressJSON := GetFlightTaskProgressLast(progressCache, "gateway-1")
	if !found {
		t.Fatal("expected flight task progress to be cached")
	}
	var progress djisdk.FlightTaskProgressEvent
	if err := json.Unmarshal([]byte(progressJSON), &progress); err != nil {
		t.Fatalf("unmarshal cached progress: %v", err)
	}
	if progress.Ext.FlightID != "flight-1" {
		t.Fatalf("progress = %+v, want flight-1", progress)
	}

	requestsPayload := []byte(`{"tid":"tid-3","bid":"bid-3","timestamp":1710000000000,"method":"airport_bind_status","data":{"status":1}}`)
	if err := client.HandleRequests(ctx, requestsPayload, "thing/product/gateway-1/requests", ""); err != nil {
		t.Fatalf("HandleRequests() error = %v", err)
	}
}

func TestRegisterDjiClientWithoutOnlineCacheHandlesUpstreamWithoutOnlineChecker(t *testing.T) {
	client := djisdk.NewClient(nil, djisdk.WithPendingTTL(time.Second), djisdk.WithReplyOptions(djisdk.ReplyOptions{}))

	RegisterDjiClient(client, RegisterDjiClientOptions{})

	requestsPayload := []byte(`{"tid":"tid-1","bid":"bid-1","timestamp":1710000000000,"method":"airport_bind_status","data":{"status":1}}`)
	if err := client.HandleRequests(context.Background(), requestsPayload, "thing/product/gateway-1/requests", ""); err != nil {
		t.Fatalf("HandleRequests() error = %v", err)
	}
}
