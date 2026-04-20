package netx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Response struct {
	StatusCode    int
	Headers       http.Header
	Data          []byte
	CostMs        int64
	CostFormatted string
	Success       bool
	Error         string
}

func DecodeJSON(resp *Response, target any) error {
	if resp == nil {
		return errors.New("response is nil")
	}
	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return json.Unmarshal(resp.Data, target)
}

func FormatCostMs(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}
