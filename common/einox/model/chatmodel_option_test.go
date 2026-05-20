package model

import "testing"

func TestChatModelOptionsToConfig(t *testing.T) {
	temp := float32(0.25)
	cfg := chatModelOptions{}
	WithAPIKey("key")(&cfg)
	WithBaseURL("http://example.test")(&cfg)
	WithModel("model-1")(&cfg)
	WithTemperature(temp)(&cfg)
	WithMaxTokens(123)(&cfg)
	WithArkRegion("cn-shanghai")(&cfg)

	got := cfg.toConfig(ProviderArk)
	if got.Provider != ProviderArk || got.APIKey != "key" || got.BaseURL != "http://example.test" || got.Model != "model-1" {
		t.Fatalf("toConfig basic fields = %+v", got)
	}
	if got.Temperature != float64(temp) || !got.TemperatureSet || got.MaxTokens != 123 || got.ArkRegion != "cn-shanghai" {
		t.Fatalf("toConfig option fields = %+v", got)
	}
	if got.OllamaURL != "http://example.test" {
		t.Fatalf("toConfig OllamaURL = %q, want base URL", got.OllamaURL)
	}
}

func TestNewChatModelByOptionUsesCanonicalUnsupportedProviderError(t *testing.T) {
	_, err := NewChatModelByOption(Provider("unknown"), WithModel("m"))
	if err == nil {
		t.Fatal("NewChatModelByOption() error = nil, want unsupported provider error")
	}
}
