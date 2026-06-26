package netx

import "context"

var defaultClient = NewClient()

// Get 发送 GET 请求（包级别便捷函数，使用默认 Client）。
func Get(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Get(ctx, url, opts...)
}

// Post 发送 POST 请求（包级别便捷函数，使用默认 Client）。
func Post(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Post(ctx, url, opts...)
}

// Put 发送 PUT 请求（包级别便捷函数，使用默认 Client）。
func Put(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Put(ctx, url, opts...)
}

// Delete 发送 DELETE 请求（包级别便捷函数，使用默认 Client）。
func Delete(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Delete(ctx, url, opts...)
}

// Patch 发送 PATCH 请求（包级别便捷函数，使用默认 Client）。
func Patch(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Patch(ctx, url, opts...)
}

// Head 发送 HEAD 请求（包级别便捷函数，使用默认 Client）。
func Head(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Head(ctx, url, opts...)
}

// Options 发送 OPTIONS 请求（包级别便捷函数，使用默认 Client）。
func Options(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Options(ctx, url, opts...)
}

// SendRequest 使用默认 Client（或通过 ClientOption 自定义）执行请求。
// 不带 opts 时直接复用 defaultClient（连接池复用）；
// 带 opts 时每次新建 Client（含新 Transport/连接池），高频调用应自行创建并持有 Client 实例。
func SendRequest(ctx context.Context, req *Request, opts ...ClientOption) (*Response, error) {
	if len(opts) == 0 {
		return defaultClient.Do(ctx, req)
	}
	c := NewClient(opts...)
	return c.Do(ctx, req)
}
