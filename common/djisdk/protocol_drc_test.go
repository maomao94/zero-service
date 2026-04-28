package djisdk

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewClientWithOptionsKeepsExplicitReplySwitches(t *testing.T) {
	options := ReplyOptions{
		EnableEventReply:   false,
		EnableStatusReply:  false,
		EnableRequestReply: false,
	}
	client := NewClient(nil, WithPendingTTL(time.Second), WithReplyOptions(options))

	if client.replyOptions != options {
		t.Fatalf("replyOptions = %+v, want %+v", client.replyOptions, options)
	}
}

func TestNewClientWithPendingTTLOption(t *testing.T) {
	client := NewClient(nil, WithPendingTTL(time.Second))
	defer client.pending.Close()

	entry, err := client.pending.Register("ttl-check")
	if err != nil {
		t.Fatalf("register pending entry: %v", err)
	}
	if entry == nil {
		t.Fatal("expected pending entry")
	}
	if !client.pending.Has("ttl-check") {
		t.Fatal("expected pending entry to be registered")
	}
}

func TestNewClientDefaultOptionsEnableReplies(t *testing.T) {
	client := NewClient(nil)

	if client.replyOptions != DefaultReplyOptions() {
		t.Fatalf("replyOptions = %+v, want default reply options", client.replyOptions)
	}
}

func TestDefaultReplyOptionsEnableReplies(t *testing.T) {
	options := DefaultReplyOptions()

	if !options.EnableEventReply || !options.EnableStatusReply || !options.EnableRequestReply {
		t.Fatalf("DefaultReplyOptions() = %+v, want all replies enabled", options)
	}
}

func TestHandleRequestsReplySwitch(t *testing.T) {
	payload := []byte(`{"tid":"tid-1","bid":"bid-1","timestamp":1710000000000,"method":"airport_bind_status","data":{"status":1}}`)
	topic := "thing/product/gateway-1/requests"

	t.Run("enabled", func(t *testing.T) {
		mqtt := &recordingMQTTClient{}
		client := newClient(mqtt, WithPendingTTL(time.Second), WithReplyOptions(ReplyOptions{EnableRequestReply: true}))
		client.OnRequest(func(ctx context.Context, gatewaySn string, req *RequestMessage) (int, any, error) {
			if gatewaySn != "gateway-1" || req.Method != "airport_bind_status" {
				t.Fatalf("unexpected request: gateway=%s method=%s", gatewaySn, req.Method)
			}
			return PlatformResultOK, map[string]any{"accepted": true}, nil
		})

		if err := client.HandleRequests(context.Background(), payload, topic, ""); err != nil {
			t.Fatalf("HandleRequests() error = %v", err)
		}
		if len(mqtt.published) != 1 || mqtt.published[0].topic != RequestsReplyTopic("gateway-1") {
			t.Fatalf("published = %+v, want one requests_reply", mqtt.published)
		}
	})

	t.Run("disabled", func(t *testing.T) {
		mqtt := &recordingMQTTClient{}
		client := newClient(mqtt, WithPendingTTL(time.Second), WithReplyOptions(ReplyOptions{EnableRequestReply: false}))
		called := false
		client.OnRequest(func(ctx context.Context, gatewaySn string, req *RequestMessage) (int, any, error) {
			called = true
			return PlatformResultOK, nil, nil
		})

		if err := client.HandleRequests(context.Background(), payload, topic, ""); err != nil {
			t.Fatalf("HandleRequests() error = %v", err)
		}
		if !called {
			t.Fatal("request handler was not called")
		}
		if len(mqtt.published) != 0 {
			t.Fatalf("published = %+v, want no requests_reply", mqtt.published)
		}
	})
}

func TestHandleStatusReplySwitch(t *testing.T) {
	payload := []byte(`{"tid":"tid-2","bid":"bid-2","timestamp":1710000000000,"method":"update_topo","data":{"sub_devices":[]}}`)
	topic := StatusTopic("gateway-1")

	t.Run("enabled", func(t *testing.T) {
		mqtt := &recordingMQTTClient{}
		client := newClient(mqtt, WithPendingTTL(time.Second), WithReplyOptions(ReplyOptions{EnableStatusReply: true}))
		client.OnStatus(func(ctx context.Context, gatewaySn string, data *StatusMessage) int {
			if gatewaySn != "gateway-1" || data.Method != MethodUpdateTopo {
				t.Fatalf("unexpected status: gateway=%s method=%s", gatewaySn, data.Method)
			}
			return PlatformResultOK
		})

		if err := client.HandleStatus(context.Background(), payload, topic, ""); err != nil {
			t.Fatalf("HandleStatus() error = %v", err)
		}
		if len(mqtt.published) != 1 || mqtt.published[0].topic != StatusReplyTopic("gateway-1") {
			t.Fatalf("published = %+v, want one status_reply", mqtt.published)
		}
	})

	t.Run("disabled", func(t *testing.T) {
		mqtt := &recordingMQTTClient{}
		client := newClient(mqtt, WithPendingTTL(time.Second), WithReplyOptions(ReplyOptions{EnableStatusReply: false}))
		called := false
		client.OnStatus(func(ctx context.Context, gatewaySn string, data *StatusMessage) int {
			called = true
			return PlatformResultOK
		})

		if err := client.HandleStatus(context.Background(), payload, topic, ""); err != nil {
			t.Fatalf("HandleStatus() error = %v", err)
		}
		if !called {
			t.Fatal("status handler was not called")
		}
		if len(mqtt.published) != 0 {
			t.Fatalf("published = %+v, want no status_reply", mqtt.published)
		}
	})
}

func TestRequestAndStatusReplyMarshal(t *testing.T) {
	requestReply := RequestReply{
		Tid:       "tid-1",
		Bid:       "bid-1",
		Timestamp: 123,
		Method:    "airport_bind_status",
		Data:      ServiceReplyData{Result: 0, Output: map[string]any{"ok": true}},
	}
	requestData, err := json.Marshal(requestReply)
	if err != nil {
		t.Fatalf("marshal request reply: %v", err)
	}
	var requestDecoded RequestReply
	if err := json.Unmarshal(requestData, &requestDecoded); err != nil {
		t.Fatalf("unmarshal request reply: %v", err)
	}
	if requestDecoded.Tid != requestReply.Tid || requestDecoded.Bid != requestReply.Bid || requestDecoded.Method != requestReply.Method || requestDecoded.Data.Result != 0 {
		t.Fatalf("unexpected request reply: %+v", requestDecoded)
	}

	statusReply := StatusReply{
		Tid:       "tid-2",
		Bid:       "bid-2",
		Timestamp: 456,
		Data:      EventReplyData{Result: 0},
	}
	statusData, err := json.Marshal(statusReply)
	if err != nil {
		t.Fatalf("marshal status reply: %v", err)
	}
	var statusDecoded StatusReply
	if err := json.Unmarshal(statusData, &statusDecoded); err != nil {
		t.Fatalf("unmarshal status reply: %v", err)
	}
	if statusDecoded.Tid != statusReply.Tid || statusDecoded.Bid != statusReply.Bid || statusDecoded.Data.Result != 0 {
		t.Fatalf("unexpected status reply: %+v", statusDecoded)
	}
}

func TestFlightTaskProgressEventUnmarshalCanonicalStructure(t *testing.T) {
	payload := []byte(`{"ext":{"current_waypoint_index":3,"wayline_mission_state":5,"media_count":8,"track_id":"track-1","flight_id":"flight-1","break_point":{"index":2,"state":1,"progress":66.5,"wayline_id":4,"break_reason":9}}}`)

	var event FlightTaskProgressEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal flight task progress: %v", err)
	}
	if event.Ext.CurrentWaypointIndex != 3 || event.Ext.WaylineMissionState != 5 || event.Ext.MediaCount != 8 || event.Ext.TrackID != "track-1" || event.Ext.FlightID != "flight-1" {
		t.Fatalf("unexpected progress ext: %+v", event.Ext)
	}
	if event.Ext.BreakPoint == nil || event.Ext.BreakPoint.BreakReason != 9 {
		t.Fatalf("unexpected breakpoint: %+v", event.Ext.BreakPoint)
	}
}

func TestHmsEventDataUnmarshalOfficialShape(t *testing.T) {
	payload := []byte(`{"list":[{"level":2,"module":3,"in_the_sky":0,"code":"dock_tip_foo","device_type":"dock","imminent":1,"args":{"component_index":2,"sensor_index":7}}]}`)

	var event HmsEventData
	if err := json.Unmarshal(payload, &event); err != nil {
		t.Fatalf("unmarshal hms event: %v", err)
	}
	if len(event.List) != 1 {
		t.Fatalf("len(list) = %d, want 1", len(event.List))
	}
	item := event.List[0]
	if item.Level != 2 || item.Module != 3 || item.InTheSky != 0 || item.Code != "dock_tip_foo" || item.DeviceType != "dock" || item.Imminent != 1 {
		t.Fatalf("unexpected hms item: %+v", item)
	}
	if item.Args.ComponentIndex != 2 || item.Args.SensorIndex != 7 {
		t.Fatalf("unexpected hms args: %+v", item.Args)
	}
}

func TestHandleStateUsesStateMessage(t *testing.T) {
	client := NewClient(nil)
	called := false
	client.OnState(func(ctx context.Context, deviceSn string, data *StateMessage) {
		called = true
		if deviceSn != "gateway-1" || data.Tid != "tid-state" {
			t.Fatalf("unexpected state callback: sn=%s data=%+v", deviceSn, data)
		}
	})

	payload := []byte(`{"tid":"tid-state","bid":"bid-state","timestamp":1710000000000,"data":{"mode_code":1}}`)
	if err := client.HandleState(context.Background(), payload, StateTopic("gateway-1"), ""); err != nil {
		t.Fatalf("HandleState() error = %v", err)
	}
	if !called {
		t.Fatal("expected state callback")
	}
}

func TestDrcUpMessageFromJSONKeepsRawData(t *testing.T) {
	payload := []byte(`{"tid":"tid-1","bid":"bid-1","timestamp":1710000000000,"method":"hsi_info_push","seq":7,"data":{"around_distance":[1,2,3]}}`)

	msg, err := DrcUpMessageFromJSON(payload)
	if err != nil {
		t.Fatalf("DrcUpMessageFromJSON() error = %v", err)
	}
	if msg.Tid != "tid-1" || msg.Bid != "bid-1" || msg.Method != MethodDrcHsiInfoPush || msg.Seq != 7 {
		t.Fatalf("unexpected message: %+v", msg)
	}
	if !strings.Contains(string(msg.Data), "around_distance") {
		t.Fatalf("raw data not preserved: %s", string(msg.Data))
	}
}

func TestDrcUpMessageFromJSONAllowsNullAndMissingData(t *testing.T) {
	cases := []string{
		`{"timestamp":1710000000000,"method":"heart_beat","data":null}`,
		`{"timestamp":1710000000000,"method":"heart_beat"}`,
	}

	for _, tc := range cases {
		msg, err := DrcUpMessageFromJSON([]byte(tc))
		if err != nil {
			t.Fatalf("DrcUpMessageFromJSON(%s) error = %v", tc, err)
		}
		if msg.Data != nil {
			t.Fatalf("expected nil data for %s, got %s", tc, string(msg.Data))
		}
		parsed, err := DrcUnmarshalUpData(msg.Method, msg.Data)
		if err != nil {
			t.Fatalf("DrcUnmarshalUpData(%s) error = %v", tc, err)
		}
		if parsed != nil {
			t.Fatalf("expected nil parsed for %s, got %T", tc, parsed)
		}
	}
}

func TestDrcUnmarshalUpDataKnownMethod(t *testing.T) {
	parsed, err := DrcUnmarshalUpData(MethodStickControl, json.RawMessage(`{"result":0,"output":{"seq":12}}`))
	if err != nil {
		t.Fatalf("DrcUnmarshalUpData() error = %v", err)
	}
	ack, ok := parsed.(*DrcStickControlAckData)
	if !ok {
		t.Fatalf("expected *DrcStickControlAckData, got %T", parsed)
	}
	if ack.Result != 0 || ack.Output == nil || ack.Output.Seq != 12 {
		t.Fatalf("unexpected ack: %+v", ack)
	}
}

func TestDrcUnmarshalUpDataUnknownMethodKeepsRaw(t *testing.T) {
	raw := json.RawMessage(`{"future_field":"value","count":2}`)

	parsed, err := DrcUnmarshalUpData("future_method", raw)
	if err != nil {
		t.Fatalf("DrcUnmarshalUpData() error = %v", err)
	}
	unknown, ok := parsed.(*DrcUnknownUpData)
	if !ok {
		t.Fatalf("expected *DrcUnknownUpData, got %T", parsed)
	}
	if unknown.Method != "future_method" || string(unknown.Raw) != string(raw) {
		t.Fatalf("unexpected unknown payload: %+v", unknown)
	}
	if summary := DrcUpPayloadSummary("future_method", unknown); summary != "unknown raw_bytes=34" {
		t.Fatalf("unexpected summary: %q", summary)
	}
}

func TestDrcUpPayloadSummaryKnownMethods(t *testing.T) {
	cases := []struct {
		name   string
		method string
		parsed any
		want   string
	}{
		{name: "stick_control", method: MethodStickControl, parsed: &DrcStickControlAckData{Result: 0}, want: "result=0"},
		{name: "emergency_stop", method: MethodDroneEmergencyStop, parsed: &DrcDroneEmergencyStopUpData{Result: 1}, want: "result=1"},
		{name: "heart_beat", method: MethodDrcHeartBeat, parsed: &DrcHeartBeatUpData{Timestamp: 1710000000000}, want: "ts=1710000000000"},
		{name: "hsi", method: MethodDrcHsiInfoPush, parsed: &DrcHsiInfoPushData{UpDistance: 10, DownDistance: 20, AroundDistances: []int{1, 2}}, want: "up=10 down=20 around=2pts"},
		{name: "delay", method: MethodDrcDelayInfoPush, parsed: &DrcDelayInfoPushData{SdrCmdDelay: 30, LiveviewDelayList: []DrcLiveviewDelayItem{{VideoID: "normal", LiveviewDelayTime: 40}}}, want: "sdr_cmd_delay=30 streams=1"},
		{name: "osd", method: MethodDrcOsdInfoPush, parsed: &DrcOsdInfoPushData{Height: 12.34, Latitude: 22.123456, Longitude: 113.654321}, want: "h=12.3 lat=22.1235 lon=113.6543"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DrcUpPayloadSummary(tc.method, tc.parsed); got != tc.want {
				t.Fatalf("DrcUpPayloadSummary() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDrcDownPayloadShapes(t *testing.T) {
	seq := 9
	heartBeat := NewDrcDownMessage("tid", "bid", MethodDrcHeartBeat, DrcHeartBeatDownData{Timestamp: 1710000000000}, &seq)
	payload, err := json.Marshal(heartBeat)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(payload), `"method":"heart_beat"`) || !strings.Contains(string(payload), `"seq":9`) || !strings.Contains(string(payload), `"timestamp":1710000000000`) {
		t.Fatalf("unexpected heart beat payload: %s", string(payload))
	}

	stickSeq := 5
	stick := NewDrcDownMessage("tid", "bid", MethodStickControl, &DrcStickControlData{Roll: 1, Pitch: 2, Throttle: 3, Yaw: 4, GimbalPitch: 5}, &stickSeq)
	payload, err = json.Marshal(stick)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(payload), `"method":"stick_control"`) || !strings.Contains(string(payload), `"seq":5`) || !strings.Contains(string(payload), `"roll":1`) || !strings.Contains(string(payload), `"gimbal_pitch":5`) {
		t.Fatalf("unexpected stick payload: %s", string(payload))
	}
	if strings.Contains(string(payload), `"x"`) || strings.Contains(string(payload), `"drone_control"`) {
		t.Fatalf("stick payload should not use legacy method or fields: %s", string(payload))
	}
	if strings.Contains(string(payload), `"gateway"`) {
		t.Fatalf("stick payload should use DrcDownMessage shape, got services shape: %s", string(payload))
	}
}

func TestDrcDownStickControlPayloadHasTopLevelSeqAndExpectedDataFields(t *testing.T) {
	seq := 42
	msg := NewDrcDownMessage("tid", "bid", MethodStickControl, DrcStickControlData{
		Roll:        1.1,
		Pitch:       -2.2,
		Throttle:    3.3,
		Yaw:         -4.4,
		GimbalPitch: 5.5,
	}, &seq)
	payload, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got["method"] != MethodStickControl {
		t.Fatalf("method = %v, want %s", got["method"], MethodStickControl)
	}
	if got["seq"] != float64(seq) {
		t.Fatalf("top-level seq = %v, want %d; payload=%s", got["seq"], seq, string(payload))
	}
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("data = %T, want object; payload=%s", got["data"], string(payload))
	}
	want := map[string]float64{
		"roll":         1.1,
		"pitch":        -2.2,
		"throttle":     3.3,
		"yaw":          -4.4,
		"gimbal_pitch": 5.5,
	}
	if len(data) != len(want) {
		t.Fatalf("data field count = %d, want %d; data=%v", len(data), len(want), data)
	}
	for k, v := range want {
		if data[k] != v {
			t.Fatalf("data[%s] = %v, want %v; data=%v", k, data[k], v, data)
		}
	}
	for _, legacy := range []string{"seq", "x", "y", "z", "gateway"} {
		if _, ok := data[legacy]; ok {
			t.Fatalf("data should not contain legacy or top-level field %q: %v", legacy, data)
		}
	}
	for _, legacy := range []string{"drone_control", "gateway", "output"} {
		if _, ok := got[legacy]; ok {
			t.Fatalf("payload should not contain legacy services field %q: %v", legacy, got)
		}
	}
}

func TestFlightTaskPrepareDataSerializesSimulateMission(t *testing.T) {
	prepare := FlightTaskPrepareData{
		FlightID: "flight-1",
		TaskType: 0,
		File:     FlightTaskFile{URL: "https://example.com/wayline.kmz"},
		SimulateMission: &SimulateMission{
			IsEnable:  true,
			Latitude:  22.123456,
			Longitude: 113.654321,
		},
	}
	payload, err := json.Marshal(prepare)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	simulateMission, ok := got["simulate_mission"].(map[string]any)
	if !ok {
		t.Fatalf("simulate_mission = %T, want object; payload=%s", got["simulate_mission"], string(payload))
	}
	if simulateMission["is_enable"] != true || simulateMission["latitude"] != 22.123456 || simulateMission["longitude"] != 113.654321 {
		t.Fatalf("unexpected simulate_mission: %v", simulateMission)
	}
	for _, misplaced := range []string{"is_enable", "latitude", "longitude"} {
		if _, ok := got[misplaced]; ok {
			t.Fatalf("simulate_mission field %q should not be serialized at top level: %s", misplaced, string(payload))
		}
	}
}

func TestFlightTaskPrepareDataOmitsNilSimulateMission(t *testing.T) {
	prepare := FlightTaskPrepareData{
		FlightID: "flight-1",
		TaskType: 0,
		File:     FlightTaskFile{URL: "https://example.com/wayline.kmz"},
	}
	payload, err := json.Marshal(prepare)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := got["simulate_mission"]; ok {
		t.Fatalf("simulate_mission should be omitted when nil: %s", string(payload))
	}
}

func TestDroneEmergencyStopPublishesDrcDownTypedMethod(t *testing.T) {
	mqtt := &recordingMQTTClient{}
	client := newClient(mqtt, WithPendingTTL(time.Second), WithReplyOptions(DefaultReplyOptions()))

	if err := client.DroneEmergencyStop(context.Background(), "gateway-1"); err != nil {
		t.Fatalf("DroneEmergencyStop() error = %v", err)
	}
	if len(mqtt.published) != 1 {
		t.Fatalf("published count = %d, want 1", len(mqtt.published))
	}
	published := mqtt.published[0]
	if published.topic != DrcDownTopic("gateway-1") {
		t.Fatalf("topic = %s, want %s", published.topic, DrcDownTopic("gateway-1"))
	}
	var got map[string]any
	if err := json.Unmarshal(published.payload, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got["method"] != MethodDroneEmergencyStop {
		t.Fatalf("method = %v, want %s", got["method"], MethodDroneEmergencyStop)
	}
	data, ok := got["data"].(map[string]any)
	if !ok {
		t.Fatalf("data = %T, want object; payload=%s", got["data"], string(published.payload))
	}
	if len(data) != 0 {
		t.Fatalf("data = %v, want empty object", data)
	}
	for _, legacy := range []string{"drone_control", "gateway", "output"} {
		if _, ok := got[legacy]; ok {
			t.Fatalf("payload should not contain legacy services field %q: %v", legacy, got)
		}
	}
}

func TestLogFieldsDoesNotIncludePayloadOrSensitiveData(t *testing.T) {
	got := logFields("topic", "thing/product/gateway-1/services", "gateway_sn", "gateway-1", "method", MethodDrcModeEnter, "tid", "tid-1", "result", 0)
	for _, want := range []string{"topic=thing/product/gateway-1/services", "gateway_sn=gateway-1", "method=drc_mode_enter", "tid=tid-1", "result=0"} {
		if !strings.Contains(got, want) {
			t.Fatalf("logFields() = %q, want contains %q", got, want)
		}
	}
	for _, sensitive := range []string{"payload", "password", "secret", "token", "certificate", "broker"} {
		if strings.Contains(strings.ToLower(got), sensitive) {
			t.Fatalf("logFields() leaked sensitive field %q in %q", sensitive, got)
		}
	}
}

func TestTask5ModulePayloadSerialization(t *testing.T) {
	cases := []struct {
		name    string
		payload any
		fields  map[string]any
	}{
		{name: "wayline", payload: FlightTaskCancelData{FlightIDs: []string{"flight-1"}}, fields: map[string]any{"flight_ids": []any{"flight-1"}}},
		{name: "drc", payload: TakeoffToPointData{FlightID: "flight-1", TargetLatitude: 22.1, TargetLongitude: 113.1, TargetHeight: 80, SecurityTakeoffHeight: 30}, fields: map[string]any{"flight_id": "flight-1", "target_latitude": 22.1, "target_longitude": 113.1, "target_height": float64(80), "security_takeoff_height": float64(30)}},
		{name: "remote_debug", payload: BatteryMaintenanceSwitchData{Enable: 1}, fields: map[string]any{"enable": float64(1)}},
		{name: "camera", payload: CameraIrMeteringAreaData{PayloadIndex: "53-0", X: 0.1, Y: 0.2, Width: 0.3, Height: 0.4}, fields: map[string]any{"payload_index": "53-0", "x": 0.1, "y": 0.2, "width": 0.3, "height": 0.4}},
		{name: "live", payload: LiveCameraChangeData{VideoID: "dock/53-0/normal", CameraIndex: "53-0-0"}, fields: map[string]any{"video_id": "dock/53-0/normal", "camera_index": "53-0-0"}},
		{name: "media", payload: MediaFastUploadData{FileID: "file-1"}, fields: map[string]any{"file_id": "file-1"}},
		{name: "remote_log", payload: RemoteLogFileUploadStartData{Files: []RemoteLogFile{{DeviceSN: "dock-1", Module: "dock", Key: "log-1", URL: "https://example.com/log.zip"}}}, fields: map[string]any{"files": []any{map[string]any{"device_sn": "dock-1", "module": "dock", "key": "log-1", "url": "https://example.com/log.zip"}}}},
		{name: "config_update", payload: ConfigUpdateData{ConfigScope: "dock", Config: map[string]any{"timezone": "Asia/Shanghai"}}, fields: map[string]any{"config_scope": "dock", "config": map[string]any{"timezone": "Asia/Shanghai"}}},
		{name: "psdk", payload: PsdkUIResourceUploadData{Name: "float-window", URL: "https://example.com/psdk.zip", Fingerprint: "sha256:abc"}, fields: map[string]any{"name": "float-window", "url": "https://example.com/psdk.zip", "fingerprint": "sha256:abc"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload, err := json.Marshal(tc.payload)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			var got map[string]any
			if err := json.Unmarshal(payload, &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			for key, want := range tc.fields {
				if !jsonValueEqual(got[key], want) {
					t.Fatalf("field %s = %#v, want %#v; payload=%s", key, got[key], want, string(payload))
				}
			}
		})
	}
}

func TestRemoteLogEventHooks(t *testing.T) {
	client := newClient(&recordingMQTTClient{}, WithPendingTTL(time.Second), WithReplyOptions(DefaultReplyOptions()))
	resultCalled := false
	progressCalled := false
	client.OnRemoteLogFileUploadResult(func(ctx context.Context, gatewaySn string, data *RemoteLogFileUploadResultEvent) {
		resultCalled = true
		if gatewaySn != "gateway-1" || len(data.Files) != 1 || data.Files[0].Result != 0 {
			t.Fatalf("unexpected result event: gateway=%s data=%+v", gatewaySn, data)
		}
	})
	client.OnRemoteLogFileUploadProgress(func(ctx context.Context, gatewaySn string, data *RemoteLogFileUploadProgressEvent) {
		progressCalled = true
		if gatewaySn != "gateway-1" || len(data.Files) != 1 || data.Files[0].Progress != 50 {
			t.Fatalf("unexpected progress event: gateway=%s data=%+v", gatewaySn, data)
		}
	})

	resultPayload := []byte(`{"tid":"tid-1","bid":"bid-1","gateway":"gateway-1","timestamp":1710000000000,"method":"fileupload_result","data":{"files":[{"key":"log-1","result":0}]}}`)
	if err := client.HandleEvents(context.Background(), resultPayload, EventsTopic("gateway-1"), ""); err != nil {
		t.Fatalf("HandleEvents(result) error = %v", err)
	}
	progressPayload := []byte(`{"tid":"tid-2","bid":"bid-2","gateway":"gateway-1","timestamp":1710000000000,"method":"fileupload_progress","data":{"files":[{"key":"log-1","progress":50}]}}`)
	if err := client.HandleEvents(context.Background(), progressPayload, EventsTopic("gateway-1"), ""); err != nil {
		t.Fatalf("HandleEvents(progress) error = %v", err)
	}
	if !resultCalled || !progressCalled {
		t.Fatalf("hooks called result=%v progress=%v", resultCalled, progressCalled)
	}
}

func TestOtaProgressAndUpdateTopoEventHooks(t *testing.T) {
	client := newClient(&recordingMQTTClient{}, WithPendingTTL(time.Second), WithReplyOptions(DefaultReplyOptions()))
	otaCalled := false
	topoCalled := false
	client.OnOtaProgress(func(ctx context.Context, gatewaySn string, data *OtaProgressEvent) {
		otaCalled = true
		if gatewaySn != "gateway-1" || len(data.Devices) != 1 || data.Devices[0].SN != "dock-1" || data.Devices[0].Progress != 42 {
			t.Fatalf("unexpected ota progress event: gateway=%s data=%+v", gatewaySn, data)
		}
	})
	client.OnUpdateTopo(func(ctx context.Context, gatewaySn string, data *TopoUpdateData) {
		topoCalled = true
		if gatewaySn != "gateway-1" || data.Type != 3 || len(data.SubDevices) != 1 || data.SubDevices[0].SN != "payload-1" {
			t.Fatalf("unexpected topo event: gateway=%s data=%+v", gatewaySn, data)
		}
	})

	otaPayload := []byte(`{"tid":"tid-3","bid":"bid-3","gateway":"gateway-1","timestamp":1710000000000,"method":"ota_progress","data":{"devices":[{"sn":"dock-1","status":2,"progress":42,"result":0}]}}`)
	if err := client.HandleEvents(context.Background(), otaPayload, EventsTopic("gateway-1"), ""); err != nil {
		t.Fatalf("HandleEvents(ota) error = %v", err)
	}
	topoPayload := []byte(`{"tid":"tid-4","bid":"bid-4","gateway":"gateway-1","timestamp":1710000000000,"method":"update_topo","data":{"type":3,"sub_type":0,"device_secret":"secret","sub_devices":[{"sn":"payload-1","type":2,"sub_type":1,"index":"0"}]}}`)
	if err := client.HandleEvents(context.Background(), topoPayload, EventsTopic("gateway-1"), ""); err != nil {
		t.Fatalf("HandleEvents(topo) error = %v", err)
	}
	if !otaCalled || !topoCalled {
		t.Fatalf("hooks called ota=%v topo=%v", otaCalled, topoCalled)
	}
}

func TestPsdkUIResourceUploadPayload(t *testing.T) {
	payload, err := json.Marshal(PsdkUIResourceUploadData{Name: "panel", URL: "https://example.com/panel.zip", Fingerprint: "sha256:abc"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	want := map[string]any{"name": "panel", "url": "https://example.com/panel.zip", "fingerprint": "sha256:abc"}
	if !jsonValueEqual(got, want) {
		t.Fatalf("payload = %#v, want %#v", got, want)
	}
}

func jsonValueEqual(got any, want any) bool {
	gotData, err := json.Marshal(got)
	if err != nil {
		return false
	}
	wantData, err := json.Marshal(want)
	if err != nil {
		return false
	}
	return string(gotData) == string(wantData)
}
