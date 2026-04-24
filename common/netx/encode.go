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

func ValidateAndFlatten(body []byte) (map[string][]string, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	result := make(map[string][]string)
	flattenMap(raw, "", result)
	return result, nil
}

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

func EncodeMultipart(fields map[string][]string) (io.Reader, string, error) {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)

	for k, vs := range fields {
		for _, v := range vs {
			fw, fErr := w.CreateFormField(k)
			if fErr != nil {
				continue
			}
			fw.Write([]byte(v))
		}
	}

	w.Close()
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
