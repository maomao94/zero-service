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

// Request 表示一个 HTTP 请求，支持 JSON/Form/Raw/Reader 等多种 Body 来源。
// 支持链式 Builder 方法和函数式 RequestOption 两种构建方式。
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

// FileUpload 表示一个文件上传字段。
type FileUpload struct {
	FieldName string
	FileName  string
	Content   io.Reader
}

// RequestOption 函数式请求配置选项。
type RequestOption func(*Request)

// NewRequest 创建请求对象，可选配合 RequestOption 进行配置。
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

// WithHeaders 批量设置请求头。
func WithHeaders(h http.Header) RequestOption {
	return func(r *Request) { r.Headers = h.Clone() }
}

// WithHeader 设置单个请求头 key-value。
func WithHeader(key, value string) RequestOption {
	return func(r *Request) {
		if r.Headers == nil {
			r.Headers = make(http.Header)
		}
		r.Headers.Set(key, value)
	}
}

// WithQueryParams 设置 URL 查询参数。
func WithQueryParams(q url.Values) RequestOption {
	return func(r *Request) { r.QueryParams = cloneValues(q) }
}

// WithFormData 设置表单数据，自动设置 Content-Type 为 application/x-www-form-urlencoded。
func WithFormData(f url.Values) RequestOption {
	return func(r *Request) {
		r.FormData = cloneValues(f)
		r.bodyKind = bodyKindForm
		r.ContentType = "application/x-www-form-urlencoded"
	}
}

// WithBody 设置原始 Body 字节数据。
func WithBody(b []byte) RequestOption {
	return func(r *Request) {
		r.Body = bytes.Clone(b)
		r.BodyReader = nil
		r.bodyKind = bodyKindRaw
	}
}

// WithJSONBody 将任意类型序列化为 JSON Body，自动设置 Content-Type 为 application/json。
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

// WithBodyReader 设置 Body 为 io.Reader 流式读取，适用于大文件或流式数据。
// 调用方负责关闭 reader（如 *os.File），Client.Do 不会自动关闭。
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
