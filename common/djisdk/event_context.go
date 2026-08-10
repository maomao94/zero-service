package djisdk

import "context"

type eventCorrelationContextKey struct{}

type eventCorrelation struct {
	tid string
	bid string
}

func withEventCorrelation(ctx context.Context, tid, bid string) context.Context {
	return context.WithValue(ctx, eventCorrelationContextKey{}, eventCorrelation{tid: tid, bid: bid})
}

// EventCorrelationFromContext returns the TID and BID from the outer DJI event envelope.
func EventCorrelationFromContext(ctx context.Context) (tid, bid string) {
	correlation, ok := ctx.Value(eventCorrelationContextKey{}).(eventCorrelation)
	if !ok {
		return "", ""
	}
	return correlation.tid, correlation.bid
}
