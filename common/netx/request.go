package netx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type bodyKind int

const (
	bodyKindNone bodyKind = iota
	bodyKindRaw
	bodyKindJSON
	bodyKindForm
	bodyKindReader
)

type Request struct {
	URL         string
	Method      string
	Headers     http.Header
	QueryParams url.Values
	FormData    url.Values
	Body        []byte
	BodyReader  io.Reader
	ContentType string
	bodyKind    bodyKind
	OptionError error
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

func (r *Request) Header(key, value string) *Request {
	WithHeader(key, value)(r)
	return r
}

func (r *Request) HeadersMap(h http.Header) *Request {
	WithHeaders(h)(r)
	return r
}

func (r *Request) Query(key, value string) *Request {
	if r.QueryParams == nil {
		r.QueryParams = make(url.Values)
	}
	r.QueryParams.Add(key, value)
	return r
}

func (r *Request) Queries(q url.Values) *Request {
	WithQueryParams(q)(r)
	return r
}

func (r *Request) JSON(v any) *Request {
	WithJSONBody(v)(r)
	return r
}

func (r *Request) Form(v url.Values) *Request {
	WithFormData(v)(r)
	return r
}

func (r *Request) Raw(b []byte) *Request {
	WithBody(b)(r)
	return r
}

func (r *Request) Reader(reader io.Reader) *Request {
	WithBodyReader(reader)(r)
	return r
}

func WithHeaders(h http.Header) RequestOption {
	return func(r *Request) { r.Headers = h.Clone() }
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
	return func(r *Request) { r.QueryParams = cloneValues(q) }
}

func WithFormData(f url.Values) RequestOption {
	return func(r *Request) {
		r.FormData = cloneValues(f)
		r.bodyKind = bodyKindForm
		r.ContentType = "application/x-www-form-urlencoded"
	}
}

func WithBody(b []byte) RequestOption {
	return func(r *Request) {
		r.Body = bytes.Clone(b)
		r.BodyReader = nil
		r.bodyKind = bodyKindRaw
	}
}

func WithJSONBody(v any) RequestOption {
	return func(r *Request) {
		data, err := json.Marshal(v)
		if err != nil {
			r.OptionError = fmt.Errorf("marshal json body: %w", err)
			return
		}
		r.Body = data
		r.BodyReader = nil
		r.bodyKind = bodyKindJSON
		r.ContentType = "application/json"
	}
}

func WithBodyReader(reader io.Reader) RequestOption {
	return func(r *Request) {
		if reader == nil {
			return
		}
		r.BodyReader = reader
		r.Body = nil
		r.bodyKind = bodyKindReader
	}
}

func cloneValues(values url.Values) url.Values {
	if values == nil {
		return nil
	}
	cloned := make(url.Values, len(values))
	for k, vs := range values {
		cloned[k] = append([]string(nil), vs...)
	}
	return cloned
}
