package gnetx

import "context"

// Handler is the unified message handler for both Server and Client.
// conn is a Conn, satisfied by both ServerConn and ClientConn.
type Handler interface {
	Handle(ctx context.Context, conn Conn, msg any) (any, error)
}

// HandlerFunc adapts a function to Handler.
type HandlerFunc func(ctx context.Context, conn Conn, msg any) (any, error)

func (f HandlerFunc) Handle(ctx context.Context, conn Conn, msg any) (any, error) {
	return f(ctx, conn, msg)
}

// AsyncHandler marks a Handler for async offload to the gnet worker pool.
type AsyncHandler interface {
	IsAsync() bool
}

type asyncFlag struct{ Handler }

func (asyncFlag) IsAsync() bool { return true }

// Async wraps a Handler to mark it for async execution.
func Async(h Handler) Handler {
	return asyncFlag{Handler: h}
}

// AsyncFunc wraps a HandlerFunc to mark it for async execution.
func AsyncFunc(fn HandlerFunc) Handler {
	return asyncFlag{Handler: fn}
}

func isAsync(h Handler) bool {
	ah, ok := h.(AsyncHandler)
	return ok && ah.IsAsync()
}
