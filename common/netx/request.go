package netx

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

type Request struct {
	URL         string
	Method      string
	Headers     http.Header
	QueryParams url.Values
	FormData    url.Values
	Body        []byte
}

type FileUpload struct {
	FieldName string
	FileName  string
	Content   io.Reader
}

type RequestOption func(*Request)

func NewRequest(rawURL, method string, opts ...RequestOption) *Request {
	r := &Request{
		URL:    rawURL,
		Method: method,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func WithHeaders(h http.Header) RequestOption {
	return func(r *Request) { r.Headers = h }
}

func WithHeader(key, value string) RequestOption {
	return func(r *Request) {
		if r.Headers == nil {
			r.Headers = make(http.Header)
		}
		r.Headers.Set(key, value)
	}
}

func WithQueryParams(q url.Values) RequestOption {
	return func(r *Request) { r.QueryParams = q }
}

func WithFormData(f url.Values) RequestOption {
	return func(r *Request) { r.FormData = f }
}

func WithBody(b []byte) RequestOption {
	return func(r *Request) { r.Body = b }
}

func WithJSONBody(v any) RequestOption {
	return func(r *Request) {
		data, err := json.Marshal(v)
		if err != nil {
			return
		}
		r.Body = data
		if r.Headers == nil {
			r.Headers = make(http.Header)
		}
		r.Headers.Set("Content-Type", "application/json")
	}
}

func WithBodyReader(reader io.Reader) RequestOption {
	return func(r *Request) {
		if reader == nil {
			return
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			return
		}
		r.Body = data
	}
}
