package djisdk

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHmsResolverEmbeddedDictionaryAndOfficialDomains(t *testing.T) {
	resolver, err := NewHmsResolver(HmsConfig{})
	if err != nil {
		t.Fatalf("NewHmsResolver() error = %v", err)
	}

	tests := []struct {
		name       string
		deviceType string
		code       string
		wantKey    string
	}{
		{name: "dock", deviceType: "3-3-0", code: "0x12040000", wantKey: "dock_tip_0x12040000"},
		{name: "aircraft", deviceType: "0-91-0", code: "0x16100083", wantKey: "fpv_tip_0x16100083"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.Resolve(HmsItem{DeviceType: tt.deviceType, Code: tt.code})
			if result.Key != tt.wantKey {
				t.Fatalf("Key = %q, want %q", result.Key, tt.wantKey)
			}
			if result.Language != "zh" || result.Message == "" {
				t.Fatalf("result = %+v, want non-empty zh message", result)
			}
		})
	}
}

func TestHmsResolverSelectsOfficialKey(t *testing.T) {
	resolver := newTestHmsResolver(t, HmsConfig{Language: "en"}, map[string]map[string]string{
		"fpv_tip_code":             {"en": "ground"},
		"fpv_tip_code_in_the_sky":  {"en": "air"},
		"fpv_tip_fallback":         {"en": "fallback"},
		"dock_tip_code":            {"en": "dock"},
		"dock_tip_code_in_the_sky": {"en": "unsupported dock sky"},
		"remote_tip_code":          {"en": "unsupported remote"},
	})

	tests := []struct {
		name    string
		item    HmsItem
		wantKey string
		wantMsg string
	}{
		{name: "aircraft ground", item: HmsItem{DeviceType: "0-67-0", Code: "code"}, wantKey: "fpv_tip_code", wantMsg: "ground"},
		{name: "aircraft sky", item: HmsItem{DeviceType: "0-67-0", Code: "code", InTheSky: 1}, wantKey: "fpv_tip_code_in_the_sky", wantMsg: "air"},
		{name: "aircraft sky fallback", item: HmsItem{DeviceType: "0-67-0", Code: "fallback", InTheSky: 1}, wantKey: "fpv_tip_fallback", wantMsg: "fallback"},
		{name: "dock ignores sky key", item: HmsItem{DeviceType: "3-3-0", Code: "code", InTheSky: 1}, wantKey: "dock_tip_code", wantMsg: "dock"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.Resolve(tt.item)
			if result.Key != tt.wantKey || result.Message != tt.wantMsg {
				t.Fatalf("Resolve() = %+v, want key %q and message %q", result, tt.wantKey, tt.wantMsg)
			}
		})
	}

	for _, deviceType := range []string{"1-83-0", "2-174-0", "dock", "4-1-0", "invalid"} {
		result := resolver.Resolve(HmsItem{DeviceType: deviceType, Code: "code", InTheSky: 1})
		if result.Key != "" || !strings.Contains(result.Message, "code") {
			t.Errorf("Resolve(%q) = %+v, want unknown alert without dictionary key", deviceType, result)
		}
	}
}

func TestHmsResolverMatchesConfiguredLanguageExactly(t *testing.T) {
	resolver := newTestHmsResolver(t, HmsConfig{Language: "fr"}, map[string]map[string]string{
		"dock_tip_preferred": {"fr": "francais", "zh": "中文", "en": "english"},
		"dock_tip_zh":        {"fr": "", "zh": "中文", "en": "english"},
		"dock_tip_en":        {"fr": "", "zh": "", "en": "english"},
		"dock_tip_first":     {"de": "deutsch", "es": "espanol"},
	})

	preferred := resolver.Resolve(HmsItem{DeviceType: "3-3-0", Code: "preferred"})
	if preferred.Language != "fr" || preferred.Message != "francais" {
		t.Fatalf("Resolve(preferred) = %+v", preferred)
	}

	for _, code := range []string{"zh", "en", "first"} {
		result := resolver.Resolve(HmsItem{DeviceType: "3-3-0", Code: code})
		if result.Language != "fr" || result.Template != "" || !strings.Contains(result.Message, code) {
			t.Errorf("Resolve(%q) = %+v, want fr unknown message", code, result)
		}
	}
}

func TestHmsResolverRendersOfficialArguments(t *testing.T) {
	resolver := newTestHmsResolver(t, HmsConfig{}, map[string]map[string]string{
		"fpv_tip_0x16100001": {"zh": "%component_index/%index/%battery_index/%dock_cover_index/%charging_rod_index/%alarmid/%gimbal_index/%lidar_index/%lte_index/%s/%1$d"},
	})
	args := decodeHmsArgs(t, `{"component_index":99,"sensor_index":2,"alarmid":"0x00AbCdEf","gimbal_index":2,"lidar_index":"3","lte_index":4}`)

	result := resolver.Resolve(HmsItem{DeviceType: "0-67-0", Code: "0x16100001", Args: args})
	if result.Message != "100/3/右/右/左/0x00AbCdEf/2/3/4/%s/%1$d" {
		t.Fatalf("Message = %q", result.Message)
	}
}

func TestHmsResolverRendersAlarmID(t *testing.T) {
	resolver := newTestHmsResolver(t, HmsConfig{}, map[string]map[string]string{
		"fpv_tip_0x16100001": {"zh": "%alarmid"},
	})

	result := resolver.Resolve(HmsItem{
		DeviceType: "0-67-0",
		Code:       "0x16100001",
		Args:       decodeHmsArgs(t, `{"alarmid":"0x00AbCdEf"}`),
	})
	if result.Message != "0x00AbCdEf" {
		t.Fatalf("Message = %q, want %q", result.Message, "0x00AbCdEf")
	}
}

func TestHmsResolverSidePlaceholders(t *testing.T) {
	resolver := newTestHmsResolver(t, HmsConfig{}, map[string]map[string]string{
		"dock_tip_side": {"zh": "%battery_index/%dock_cover_index"},
	})
	tests := []struct {
		sensorIndex int
		want        string
	}{
		{sensorIndex: 0, want: "左/左"},
		{sensorIndex: 1, want: "右/右"},
		{sensorIndex: 8, want: "右/右"},
	}
	for _, tt := range tests {
		result := resolver.Resolve(HmsItem{DeviceType: "3-3-0", Code: "side", Args: HmsArgs{"sensor_index": tt.sensorIndex}})
		if result.Message != tt.want {
			t.Errorf("sensor_index=%d Message = %q, want %q", tt.sensorIndex, result.Message, tt.want)
		}
	}
}

func TestHmsResolverChargingRodDirections(t *testing.T) {
	resolver := newTestHmsResolver(t, HmsConfig{}, map[string]map[string]string{
		"dock_tip_direction": {"zh": "%charging_rod_index"},
	})
	wants := map[int]string{0: "前", 1: "后", 2: "左", 3: "右", 4: "%charging_rod_index", -1: "%charging_rod_index"}
	for sensorIndex, want := range wants {
		result := resolver.Resolve(HmsItem{DeviceType: "3-3-0", Code: "direction", Args: HmsArgs{"sensor_index": sensorIndex}})
		if result.Message != want {
			t.Errorf("sensor_index=%d Message = %q, want %q", sensorIndex, result.Message, want)
		}
	}
}

func TestHmsResolverKeepsMissingArgumentsAndUnknownCode(t *testing.T) {
	resolver := newTestHmsResolver(t, HmsConfig{}, map[string]map[string]string{
		"fpv_tip_missing": {"zh": "%component_index/%index/%battery_index/%dock_cover_index/%charging_rod_index/%alarmid"},
	})

	template := "%component_index/%index/%battery_index/%dock_cover_index/%charging_rod_index/%alarmid"
	tests := []struct {
		name string
		args HmsArgs
	}{
		{name: "missing", args: nil},
		{name: "explicit nil", args: HmsArgs{"component_index": nil, "sensor_index": nil, "alarmid": nil}},
		{name: "illegal types", args: HmsArgs{"component_index": []int{1}, "sensor_index": []int{0}, "alarmid": map[string]any{"raw": "0x1"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolver.Resolve(HmsItem{DeviceType: "0-67-0", Code: "missing", Args: tt.args})
			if result.Message != template {
				t.Fatalf("Message = %q, want unresolved placeholders preserved", result.Message)
			}
		})
	}

	unknown := resolver.Resolve(HmsItem{DeviceType: "3-3-0", Code: "0xDEADBEEF"})
	if unknown.Message == "" || !strings.Contains(unknown.Message, "0xDEADBEEF") {
		t.Fatalf("unknown result = %+v, want non-empty message containing code", unknown)
	}
}

func newTestHmsResolver(t *testing.T, cfg HmsConfig, dictionary map[string]map[string]string) *HmsResolver {
	t.Helper()
	data, err := json.Marshal(dictionary)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	path := filepath.Join(t.TempDir(), "hms.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	cfg.DictionaryPath = path
	resolver, err := NewHmsResolver(cfg)
	if err != nil {
		t.Fatalf("NewHmsResolver() error = %v", err)
	}
	return resolver
}

func decodeHmsArgs(t *testing.T, raw string) HmsArgs {
	t.Helper()
	var args HmsArgs
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return args
}
