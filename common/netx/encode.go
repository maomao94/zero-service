package netx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"
)

// ValidateAndFlatten 将 JSON body 扁平化为 key-value 映射，支持嵌套对象和数组。
func ValidateAndFlatten(body []byte) (map[string][]string, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	result := make(map[string][]string)
	flattenMap(raw, "", result)
	return result, nil
}

// EncodeURLEncodedIfNeeded 检测 body 是否已是合法 URL-encoded 格式，否则尝试 JSON 扁平化编码。
// 如果 JSON 扁平化也失败，原样返回 body。
func EncodeURLEncodedIfNeeded(body []byte) (io.Reader, string) {
	// 如果 body 是合法 JSON，优先尝试扁平化编码
	if json.Valid(body) {
		encoded, err := EncodeURLEncoded(body)
		if err != nil {
			return bytes.NewReader(body), "application/x-www-form-urlencoded"
		}
		return strings.NewReader(encoded), "application/x-www-form-urlencoded"
	}
	// 检测是否已是合法的 URL-encoded 格式
	if q, err := url.ParseQuery(string(body)); err == nil && len(q) > 0 {
		for k := range q {
			if k != "" {
				return bytes.NewReader(body), "application/x-www-form-urlencoded"
			}
		}
	}
	encoded, err := EncodeURLEncoded(body)
	if err != nil {
		return bytes.NewReader(body), "application/x-www-form-urlencoded"
	}
	return strings.NewReader(encoded), "application/x-www-form-urlencoded"
}

// EncodeURLEncoded 将 JSON body 转换并编码为 URL-encoded 格式字符串。
func EncodeURLEncoded(body []byte) (string, error) {
	data, err := ValidateAndFlatten(body)
	if err != nil {
		return "", err
	}
	values := make(url.Values)
	for k, vs := range data {
		for _, v := range vs {
			values.Add(k, v)
		}
	}
	return values.Encode(), nil
}

// EncodeMultipart 将字段映射编码为 multipart/form-data 格式的 reader 和 Content-Type 头。
func EncodeMultipart(fields map[string][]string) (io.Reader, string, error) {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)

	for k, vs := range fields {
		for _, v := range vs {
			fw, err := w.CreateFormField(k)
			if err != nil {
				return nil, "", fmt.Errorf("create form field %q: %w", k, err)
			}
			if _, err = fw.Write([]byte(v)); err != nil {
				return nil, "", fmt.Errorf("write form field %q: %w", k, err)
			}
		}
	}

	if err := w.Close(); err != nil {
		return nil, "", fmt.Errorf("close multipart writer: %w", err)
	}
	return buf, w.FormDataContentType(), nil
}

func flattenMap(data map[string]any, prefix string, result map[string][]string) {
	for key, val := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		flattenValue(val, fullKey, result)
	}
}

func flattenValue(val any, key string, result map[string][]string) {
	if key == "" {
		return
	}
	switch v := val.(type) {
	case nil:
		return
	case string:
		result[key] = append(result[key], v)
	case bool:
		result[key] = append(result[key], fmt.Sprintf("%v", v))
	case json.Number:
		result[key] = append(result[key], string(v))
	case float64:
		result[key] = append(result[key], formatFloat(v))
	case float32:
		result[key] = append(result[key], formatFloat(float64(v)))
	case map[string]any:
		flattenMap(v, key, result)
	case []any:
		for _, item := range v {
			flattenValue(item, key, result)
		}
	default:
		s := fmt.Sprintf("%v", v)
		if s != "" {
			result[key] = append(result[key], s)
		}
	}
}

func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return strings.TrimRight(fmt.Sprintf("%f", f), "0")
}
