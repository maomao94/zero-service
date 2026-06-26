package logic

import (
	"testing"
	"time"

	"zero-service/common/djisdk"
	"zero-service/common/mqttx"
)

func TestToDrcMqttBroker_Tcp(t *testing.T) {
	cfg := mqttx.MqttConfig{
		Broker:   []string{"tcp://mqtt.example.com:1883"},
		ClientID: "client-1",
		Username: "user",
		Password: "pass",
	}
	got := toDrcMqttBroker(cfg, "")
	// DRC 必须使用独立 MQTT 连接，ClientID 每次重新生成，不复用主 MQTT ClientID
	if got.ClientID == cfg.ClientID {
		t.Errorf("DRC ClientID must be independent, got reused main client_id=%q", got.ClientID)
	}
	assertDrcBroker(t, got, djisdk.DrcMqttBroker{
		Address:    "mqtt.example.com:1883",
		ClientID:   got.ClientID,
		Username:   "user",
		Password:   "pass",
		EnableTLS:  false,
		ExpireTime: got.ExpireTime,
	})
	if got.ExpireTime < time.Now().Add(6*24*time.Hour).Unix() {
		t.Errorf("ExpireTime too early: got %d, want at least 7 days from now", got.ExpireTime)
	}
}

func TestToDrcMqttBroker_TlsTcps(t *testing.T) {
	cfg := mqttx.MqttConfig{
		Broker:   []string{"tcps://mqtt.example.com:8883"},
		ClientID: "client-2",
		Username: "user2",
		Password: "pass2",
	}
	got := toDrcMqttBroker(cfg, "")
	if got.Address != "mqtt.example.com:8883" {
		t.Errorf("Address = %q, want %q", got.Address, "mqtt.example.com:8883")
	}
	if !got.EnableTLS {
		t.Error("EnableTLS = false, want true for tcps://")
	}
}

func TestToDrcMqttBroker_TlsMqtts(t *testing.T) {
	cfg := mqttx.MqttConfig{
		Broker: []string{"mqtts://mqtt.example.com:8883"},
	}
	got := toDrcMqttBroker(cfg, "")
	if got.Address != "mqtt.example.com:8883" {
		t.Errorf("Address = %q, want %q", got.Address, "mqtt.example.com:8883")
	}
	if !got.EnableTLS {
		t.Error("EnableTLS = false, want true for mqtts://")
	}
}

func TestToDrcMqttBroker_NoScheme(t *testing.T) {
	cfg := mqttx.MqttConfig{
		Broker: []string{"mqtt.example.com:1883"},
	}
	got := toDrcMqttBroker(cfg, "")
	if got.Address != "mqtt.example.com:1883" {
		t.Errorf("Address = %q, want %q", got.Address, "mqtt.example.com:1883")
	}
	if got.EnableTLS {
		t.Error("EnableTLS = true, want false for bare address")
	}
}

func TestToDrcMqttBroker_EmptyBroker(t *testing.T) {
	cfg := mqttx.MqttConfig{
		ClientID: "client-empty",
		Username: "user",
		Password: "pass",
	}
	got := toDrcMqttBroker(cfg, "")
	if got.Address != "" {
		t.Errorf("Address = %q, want empty", got.Address)
	}
	// DRC 必须使用独立 MQTT 连接，ClientID 每次重新生成
	if got.ClientID == cfg.ClientID {
		t.Errorf("DRC ClientID must be independent, got reused main client_id=%q", got.ClientID)
	}
}

func TestToDrcMqttBroker_Passthrough(t *testing.T) {
	cfg := mqttx.MqttConfig{
		Broker:   []string{"tcp://host:1883"},
		ClientID: "custom-client",
		Username: "custom-user",
		Password: "custom-pass",
	}
	got := toDrcMqttBroker(cfg, "")
	// DRC 必须使用独立 MQTT 连接，ClientID 每次重新生成
	if got.ClientID == cfg.ClientID {
		t.Errorf("DRC ClientID must be independent, got reused main client_id=%q", got.ClientID)
	}
	if got.Username != "custom-user" {
		t.Errorf("Username = %q", got.Username)
	}
	if got.Password != "custom-pass" {
		t.Errorf("Password = %q", got.Password)
	}
}

func TestToDrcMqttBroker_DrcAddressOverride(t *testing.T) {
	cfg := mqttx.MqttConfig{
		Broker:   []string{"tcp://10.10.1.103:1883"},
		Username: "user",
		Password: "pass",
	}
	got := toDrcMqttBroker(cfg, "public.example.com:1883")
	if got.Address != "public.example.com:1883" {
		t.Errorf("Address = %q, want %q", got.Address, "public.example.com:1883")
	}
	if got.EnableTLS {
		t.Error("EnableTLS = true, want false for bare drc address")
	}
	if got.Username != "user" {
		t.Errorf("Username = %q", got.Username)
	}
	if got.Password != "pass" {
		t.Errorf("Password = %q", got.Password)
	}
}

func TestToDrcMqttBroker_DrcAddressOverrideWithScheme(t *testing.T) {
	cfg := mqttx.MqttConfig{
		Broker:   []string{"tcp://10.10.1.103:1883"},
		Username: "user",
		Password: "pass",
	}
	got := toDrcMqttBroker(cfg, "tcps://public.example.com:8883")
	if got.Address != "public.example.com:8883" {
		t.Errorf("Address = %q, want %q", got.Address, "public.example.com:8883")
	}
	if !got.EnableTLS {
		t.Error("EnableTLS = false, want true for tcps:// drc address")
	}
}

func TestToDrcMqttBroker_DrcAddressEmptyFallsBack(t *testing.T) {
	cfg := mqttx.MqttConfig{
		Broker:   []string{"tcp://mqtt.internal:1883"},
		Username: "user",
		Password: "pass",
	}
	got := toDrcMqttBroker(cfg, "")
	if got.Address != "mqtt.internal:1883" {
		t.Errorf("Address = %q, want fallback to broker %q", got.Address, "mqtt.internal:1883")
	}
}

func assertDrcBroker(t *testing.T, got, want djisdk.DrcMqttBroker) {
	t.Helper()
	if got.Address != want.Address {
		t.Errorf("Address = %q, want %q", got.Address, want.Address)
	}
	if got.ClientID != want.ClientID {
		t.Errorf("ClientID = %q, want %q", got.ClientID, want.ClientID)
	}
	if got.Username != want.Username {
		t.Errorf("Username = %q, want %q", got.Username, want.Username)
	}
	if got.Password != want.Password {
		t.Errorf("Password = %q, want %q", got.Password, want.Password)
	}
	if got.EnableTLS != want.EnableTLS {
		t.Errorf("EnableTLS = %v, want %v", got.EnableTLS, want.EnableTLS)
	}
}
