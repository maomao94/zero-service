package nacosx

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc/resolver"
)

func init() {
	resolver.Register(&builder{})
}

// schemeName for the urls
// All target URLs like 'nacos://.../...' will be resolved by this resolver
const schemeName = "nacos"

// builder implements resolver.Builder and use for constructing all nacos resolvers
type builder struct{}

func (b *builder) Build(url resolver.Target, conn resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	tgt, err := parseURL(url.URL)
	if err != nil {
		return nil, errors.Wrap(err, "Wrong nacos URL")
	}

	cli, err := newWatchClient(tgt)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	pipe := make(chan []string)
	watch := newWatcher(ctx, cancel, pipe)

	go subscribeNacos(ctx, cli, tgt, watch.CallBackHandle)

	go populateEndpoints(ctx, conn, pipe)

	return &resolvr{
		cancelFunc: cancel,
		client:     cli,
	}, nil
}

// Scheme returns the scheme supported by this resolver.
// Scheme is defined at https://github.com/grpc/grpc/blob/master/doc/naming.md.
func (b *builder) Scheme() string {
	return schemeName
}
