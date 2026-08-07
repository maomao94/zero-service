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

func TestHmsResolverOnlyUsesOfficialCategory(t *testing.T) {
	resolver := newTestHmsResolver(t, HmsConfig{Language: "en"}, map[string]map[string]string{
		"fpv_tip_code":             {"en": "ground"},
		"fpv_tip_code_in_the_sky":  {"en": "air"},
		"dock_tip_code":            {"en": "dock"},
		"dock_tip_code_in_the_sky": {"en": "unsupported dock sky"},
		"remote_tip_code":          {"en": "unsupported remote"},
	})

	if result := resolver.Resolve(HmsItem{DeviceType: "0-67-0", Code: "code", InTheSky: 1}); result.Key != "fpv_tip_code_in_the_sky" || result.Message != "air" {
		t.Fatalf("aircraft sky result = %+v", result)
	}
	if result := resolver.Resolve(HmsItem{DeviceType: "0-67-0", Code: "code"}); result.Key != "fpv_tip_code" || result.Message != "ground" {
		t.Fatalf("aircraft ground result = %+v", result)
	}
	if result := resolver.Resolve(HmsItem{DeviceType: "3-3-0", Code: "code", InTheSky: 1}); result.Key != "dock_tip_code" || result.Message != "dock" {
		t.Fatalf("dock result = %+v", result)
	}
	for _, deviceType := range []string{"1-83-0", "2-174-0", "dock", "4-1-0", "invalid"} {
		result := resolver.Resolve(HmsItem{DeviceType: deviceType, Code: "code", InTheSky: 1})
		if result.Key != "" || !strings.Contains(result.Message, "code") {
			t.Errorf("Resolve(%q) = %+v, want unknown alert without dictionary key", deviceType, result)
		}
	}
}

func TestHmsResolverFallsBackFromMissingSkyKey(t *testing.T) {
	resolver := newTestHmsResolver(t, HmsConfig{}, map[string]map[string]string{
		"fpv_tip_code": {"zh": "地面文案"},
	})
	result := resolver.Resolve(HmsItem{DeviceType: "0-67-0", Code: "code", InTheSky: 1})
	if result.Key != "fpv_tip_code" || result.Message != "地面文案" {
		t.Fatalf("Resolve() = %+v", result)
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
		"fpv_tip_args": {"zh": "%component_index/%index/%battery_index/%dock_cover_index/%charging_rod_index/%alarmid/%gimbal_index/%lidar_index/%lte_index/%s/%1$d"},
	})
	args := decodeHmsArgs(t, `{"component_index":0,"sensor_index":2,"alarmid":"0x16100001","gimbal_index":2,"lidar_index":"3","lte_index":4}`)

	result := resolver.Resolve(HmsItem{DeviceType: "0-67-0", Code: "args", Args: args})
	if result.Message != "1/3/右/右/左/0x16100001/2/3/4/%s/%1$d" {
		t.Fatalf("Message = %q", result.Message)
	}
}

func TestHmsResolverChargingRodDirections(t *testing.T) {
	resolver := newTestHmsResolver(t, HmsConfig{}, map[string]map[string]string{
		"dock_tip_direction": {"zh": "%charging_rod_index"},
	})
	wants := map[int]string{0: "前", 1: "后", 2: "左", 3: "右", 4: "%charging_rod_index", -1: "%charging_rod_index"}
	for sensorIndex, want := range wants {
		result := resolver.Resolve(HmsItem{DeviceType: "3-3-0", Code: "direction", Args: map[string]any{"sensor_index": sensorIndex}})
		if result.Message != want {
			t.Errorf("sensor_index=%d Message = %q, want %q", sensorIndex, result.Message, want)
		}
	}
}

func TestHmsResolverKeepsMissingArgumentsAndUnknownCode(t *testing.T) {
	resolver := newTestHmsResolver(t, HmsConfig{}, map[string]map[string]string{
		"fpv_tip_missing": {"zh": "云台%component_index异常（%alarmid）"},
	})

	result := resolver.Resolve(HmsItem{DeviceType: "0-67-0", Code: "missing"})
	if result.Message != "云台%component_index异常（%alarmid）" {
		t.Fatalf("Message = %q, want unresolved placeholders preserved", result.Message)
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
