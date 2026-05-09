package netx

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"time"
)

var (
	ErrResponseTooLarge = errors.New("response body too large")
	ErrUploadTooLarge   = errors.New("upload body too large")
)

type Response struct {
	StatusCode    int
	Headers       http.Header
	Data          []byte
	CostMs        int64
	CostFormatted string
	Success       bool
	Err           error
}

func (r *Response) JSON(target any) error {
	if err := r.ensureDecodable(); err != nil {
		return err
	}
	return json.Unmarshal(r.Data, target)
}

func (r *Response) XML(target any) error {
	if err := r.ensureDecodable(); err != nil {
		return err
	}
	return xml.Unmarshal(r.Data, target)
}

func (r *Response) Text() (string, error) {
	if err := r.ensureDecodable(); err != nil {
		return "", err
	}
	return string(r.Data), nil
}

func (r *Response) Decode(target any) error {
	if err := r.ensureDecodable(); err != nil {
		return err
	}
	mediaType := ""
	if r.Headers != nil {
		mediaType = r.Headers.Get("Content-Type")
	}
	if parsed, _, err := mime.ParseMediaType(mediaType); err == nil {
		mediaType = parsed
	}
	mediaType = strings.ToLower(mediaType)
	sniffed := strings.TrimSpace(string(r.Data))
	if strings.Contains(mediaType, "json") || strings.HasPrefix(sniffed, "{") || strings.HasPrefix(sniffed, "[") {
		return json.Unmarshal(r.Data, target)
	}
	if strings.Contains(mediaType, "xml") || strings.HasPrefix(sniffed, "<") {
		return xml.Unmarshal(r.Data, target)
	}
	if strings.HasPrefix(mediaType, "text/") {
		if s, ok := target.(*string); ok {
			*s = string(r.Data)
			return nil
		}
		return fmt.Errorf("decode text response requires *string target")
	}
	return fmt.Errorf("unsupported response content type: %s", mediaType)
}

func (r *Response) ensureDecodable() error {
	if r == nil {
		return errors.New("response is nil")
	}
	if r.Err != nil {
		return r.Err
	}
	if !r.Success {
		return fmt.Errorf("request failed: status %d", r.StatusCode)
	}
	return nil
}

func DecodeJSON(resp *Response, target any) error {
	return resp.JSON(target)
}

func FormatCostMs(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

func elapsedSince(start time.Time) (int64, string) {
	costMs := time.Since(start).Milliseconds()
	return costMs, FormatCostMs(costMs)
}
