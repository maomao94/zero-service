package netx

import "context"

var defaultClient = NewClient()

func Get(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Get(ctx, url, opts...)
}

func Post(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Post(ctx, url, opts...)
}

func Put(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Put(ctx, url, opts...)
}

func Delete(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Delete(ctx, url, opts...)
}

func Patch(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Patch(ctx, url, opts...)
}

func Head(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Head(ctx, url, opts...)
}

func Options(ctx context.Context, url string, opts ...RequestOption) (*Response, error) {
	return defaultClient.Options(ctx, url, opts...)
}

func SendRequest(ctx context.Context, req *Request, opts ...ClientOption) (*Response, error) {
	if len(opts) == 0 {
		return defaultClient.Do(ctx, req)
	}
	c := NewClient(opts...)
	return c.Do(ctx, req)
}
