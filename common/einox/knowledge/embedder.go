package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Embedder 将文本编码为向量。
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// ArkEmbedder 火山 ARK /v3/embeddings。
type ArkEmbedder struct {
	client  *http.Client
	baseURL string
	apiKey  string
	model   string
}

// NewArkEmbedder baseURL 为空时按 region 选北京或上海 ARK 端点。
func NewArkEmbedder(apiKey, baseURL, model, region string) *ArkEmbedder {
	b := strings.TrimSpace(baseURL)
	if b == "" {
		switch strings.ToLower(strings.TrimSpace(region)) {
		case "cn-shanghai", "shanghai":
			b = "https://ark.cn-shanghai.volces.com/api/v3"
		default:
			b = "https://ark.cn-beijing.volces.com/api/v3"
		}
	}
	return &ArkEmbedder{
		client:  &http.Client{Timeout: 120 * time.Second},
		baseURL: strings.TrimSuffix(b, "/"),
		apiKey:  strings.TrimSpace(apiKey),
		model:   strings.TrimSpace(model),
	}
}

type embeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingsResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Embed 批量调用。
func (e *ArkEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if e.apiKey == "" {
		return nil, fmt.Errorf("knowledge embedding: api key is empty")
	}
	if e.model == "" {
		return nil, fmt.Errorf("knowledge embedding: model (endpoint id) is empty")
	}
	const batch = 16
	var out [][]float32
	for i := 0; i < len(texts); i += batch {
		j := i + batch
		if j > len(texts) {
			j = len(texts)
		}
		part := texts[i:j]
		vec, err := e.embedBatch(ctx, part)
		if err != nil {
			return nil, err
		}
		out = append(out, vec...)
	}
	return out, nil
}

func (e *ArkEmbedder) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	body, err := json.Marshal(embeddingsRequest{Model: e.model, Input: texts})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("knowledge embedding: http %d: %s", resp.StatusCode, string(raw))
	}
	var er embeddingsResponse
	if err := json.Unmarshal(raw, &er); err != nil {
		return nil, fmt.Errorf("knowledge embedding: decode: %w", err)
	}
	if er.Error != nil && er.Error.Message != "" {
		return nil, fmt.Errorf("knowledge embedding: %s", er.Error.Message)
	}
	if len(er.Data) != len(texts) {
		return nil, fmt.Errorf("knowledge embedding: expect %d vectors, got %d", len(texts), len(er.Data))
	}
	out := make([][]float32, len(er.Data))
	for i := range er.Data {
		v := make([]float32, len(er.Data[i].Embedding))
		for k, x := range er.Data[i].Embedding {
			v[k] = float32(x)
		}
		out[i] = v
	}
	return out, nil
}
