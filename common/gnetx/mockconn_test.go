package gnetx

import (
	"bytes"
	"io"
	"net"
	"time"

	"github.com/panjf2000/gnet/v2"
)

// mockConn 是仅用于 Framer 单测的最小 gnet.Conn 实现。
// 只支持 Peek/Discard/InboundBuffered/Write/RemoteAddr/LocalAddr 等分帧所需方法；
// 其余方法返回零值或 ErrUnsupported，不在测试中使用。
// 它模拟 gnet 的 inbound buffer：Peek 返回前 n 字节但不消费，Discard 消费 n 字节。
type mockConn struct {
	buf    *bytes.Buffer
	remote net.Addr
	local  net.Addr
	ctx    any
	closed bool
}

func newMockConn(data []byte) *mockConn {
	return &mockConn{
		buf:    bytes.NewBuffer(append([]byte(nil), data...)),
		remote: &net.TCPAddr{IP: net.IPv4(1, 2, 3, 4), Port: 9999},
		local:  &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9000},
	}
}

func (c *mockConn) Peek(n int) ([]byte, error) {
	if c.buf.Len() < n {
		return nil, io.ErrShortBuffer
	}
	b := c.buf.Bytes()
	out := make([]byte, n)
	copy(out, b[:n])
	return out, nil
}

func (c *mockConn) Discard(n int) (int, error) {
	if c.buf.Len() < n {
		n = c.buf.Len()
	}
	c.buf.Next(n)
	return n, nil
}

func (c *mockConn) Next(n int) ([]byte, error) {
	if n < 0 {
		b := c.buf.Bytes()
		out := make([]byte, len(b))
		copy(out, b)
		c.buf.Reset()
		return out, nil
	}
	if c.buf.Len() < n {
		return nil, io.ErrShortBuffer
	}
	out := make([]byte, n)
	_, _ = c.buf.Read(out)
	return out, nil
}

func (c *mockConn) InboundBuffered() int { return c.buf.Len() }

func (c *mockConn) Read(p []byte) (int, error)         { return c.buf.Read(p) }
func (c *mockConn) WriteTo(w io.Writer) (int64, error) { return c.buf.WriteTo(w) }
func (c *mockConn) Write(p []byte) (int, error)        { return c.buf.Write(p) }
func (c *mockConn) Writev(bs [][]byte) (int, error) {
	var n int
	for _, b := range bs {
		x, _ := c.buf.Write(b)
		n += x
	}
	return n, nil
}
func (c *mockConn) ReadFrom(r io.Reader) (int64, error)           { return c.buf.ReadFrom(r) }
func (c *mockConn) SendTo(buf []byte, addr net.Addr) (int, error) { return 0, nil }
func (c *mockConn) Flush() error                                  { return nil }
func (c *mockConn) OutboundBuffered() int                         { return 0 }
func (c *mockConn) AsyncWrite(buf []byte, cb gnet.AsyncCallback) error {
	_, _ = c.buf.Write(buf)
	if cb != nil {
		_ = cb(c, nil)
	}
	return nil
}
func (c *mockConn) AsyncWritev(bs [][]byte, cb gnet.AsyncCallback) error {
	for _, b := range bs {
		_, _ = c.buf.Write(b)
	}
	return nil
}

func (c *mockConn) Context() any           { return c.ctx }
func (c *mockConn) SafeContext() any       { return c.ctx }
func (c *mockConn) SetContext(ctx any)     { c.ctx = ctx }
func (c *mockConn) SetSafeContext(ctx any) { c.ctx = ctx }
func (c *mockConn) LocalAddr() net.Addr    { return c.local }
func (c *mockConn) RemoteAddr() net.Addr   { return c.remote }

func (c *mockConn) Fd() int                                  { return -1 }
func (c *mockConn) Dup() (int, error)                        { return -1, nil }
func (c *mockConn) SetReadBuffer(int) error                  { return nil }
func (c *mockConn) SetWriteBuffer(int) error                 { return nil }
func (c *mockConn) SetLinger(int) error                      { return nil }
func (c *mockConn) SetKeepAlivePeriod(d time.Duration) error { return nil }
func (c *mockConn) SetKeepAlive(bool, time.Duration, time.Duration, int) error {
	return nil
}
func (c *mockConn) SetNoDelay(bool) error { return nil }

func (c *mockConn) EventLoop() gnet.EventLoop { return nil }

func (c *mockConn) Wake(gnet.AsyncCallback) error { return nil }
func (c *mockConn) CloseWithCallback(gnet.AsyncCallback) error {
	c.closed = true
	return nil
}
func (c *mockConn) Close() error {
	c.closed = true
	return nil
}
func (c *mockConn) SetDeadline(time.Time) error      { return nil }
func (c *mockConn) SetReadDeadline(time.Time) error  { return nil }
func (c *mockConn) SetWriteDeadline(time.Time) error { return nil }
