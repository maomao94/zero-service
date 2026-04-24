package antsx

import (
	"context"
	"sync"
)

// UnboundedChan 是无容量限制的泛型阻塞队列。
// 使用 sync.Mutex + sync.Cond 实现，与 Go 原生 channel 语义对齐：
//   - Send 在已关闭时 panic（与 Go channel 一致），可使用 TrySend 避免
//   - Receive 阻塞等待数据，关闭后排空缓冲区，返回 (zero, false)
//   - Close 幂等安全，唤醒所有阻塞的 Receive
//
// 内部使用 off 偏移量 + 周期性 compact 策略管理底层数组，
// 避免 buffer[1:] 导致底层数组头部空间无法被 GC 回收。
//
// 协程安全：所有方法均可在多个 goroutine 中并发调用。
type UnboundedChan[T any] struct {
	buffer   []T
	off      int
	mu       sync.Mutex
	notEmpty *sync.Cond
	closed   bool
}

// NewUnboundedChan 创建一个新的无界通道。
func NewUnboundedChan[T any]() *UnboundedChan[T] {
	ch := &UnboundedChan[T]{}
	ch.notEmpty = sync.NewCond(&ch.mu)
	return ch
}

// Send 向通道发送一个值。如果通道已关闭则 panic（与 Go channel 语义一致）。
// 发送后唤醒一个阻塞在 Receive 的 goroutine。
func (ch *UnboundedChan[T]) Send(val T) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	if ch.closed {
		panic(ErrChanClosed)
	}
	ch.buffer = append(ch.buffer, val)
	ch.notEmpty.Signal()
}

// TrySend 尝试向通道发送一个值。与 Send 不同，已关闭时返回 false 而非 panic。
func (ch *UnboundedChan[T]) TrySend(val T) bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	if ch.closed {
		return false
	}
	ch.buffer = append(ch.buffer, val)
	ch.notEmpty.Signal()
	return true
}

// Receive 从通道接收一个值。缓冲区为空时阻塞等待。
// 返回 (val, true) 表示成功接收；(zero, false) 表示通道已关闭且缓冲区已空。
//
// 注意：此方法不支持 context 取消控制，goroutine 可能永久阻塞。
// 如需超时/取消控制，请使用 ReceiveContext。
func (ch *UnboundedChan[T]) Receive() (T, bool) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	for ch.off >= len(ch.buffer) && !ch.closed {
		ch.notEmpty.Wait()
	}
	return ch.dequeue()
}

// ReceiveContext 从通道接收一个值，支持 context 超时/取消控制。
// 当 ctx 被取消且缓冲区为空时，返回 (zero, false)，goroutine 不会泄漏。
//
// 实现原理：启动后台 goroutine 在 ctx.Done() 时通过 Broadcast 唤醒阻塞的 Wait，
// 检查 ctx 状态后退出循环。done channel 确保数据先到达时后台 goroutine 立即退出。
//
// 锁安全性：后台 goroutine 调用 ch.notEmpty.Broadcast() 内部需获取 ch.mu 关联的锁。
// 主 goroutine 在 ch.notEmpty.Wait() 时会原子地释放 ch.mu 并挂起，因此后台
// goroutine 可以获取到 ch.mu 执行 Broadcast。Wait 被唤醒后重新获取 ch.mu
// 继续循环检查条件，不构成死锁。
func (ch *UnboundedChan[T]) ReceiveContext(ctx context.Context) (T, bool) {
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			ch.notEmpty.Broadcast()
		case <-done:
		}
	}()
	defer close(done)

	ch.mu.Lock()
	defer ch.mu.Unlock()
	for ch.off >= len(ch.buffer) && !ch.closed {
		if ctx.Err() != nil {
			var zero T
			return zero, false
		}
		ch.notEmpty.Wait()
	}
	if ch.off >= len(ch.buffer) {
		var zero T
		return zero, false
	}
	return ch.dequeue()
}

// Close 关闭通道，唤醒所有阻塞在 Receive/ReceiveContext 的 goroutine。
// 幂等调用安全，多次调用不会 panic。
// 关闭后 Receive 仍可排空缓冲区中的剩余数据。
func (ch *UnboundedChan[T]) Close() {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	if !ch.closed {
		ch.closed = true
		ch.notEmpty.Broadcast()
	}
}

// Len 返回当前缓冲区中的元素数量。
func (ch *UnboundedChan[T]) Len() int {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	return len(ch.buffer) - ch.off
}

// dequeue 从缓冲区取出一个元素并执行必要的 compact。
// 调用方必须持有 ch.mu 锁。
//
// compact 策略：当已消费偏移量（off）超过总容量的一半时，
// 将剩余数据拷贝到新分配的切片，释放底层旧数组以供 GC 回收。
func (ch *UnboundedChan[T]) dequeue() (T, bool) {
	if ch.off >= len(ch.buffer) {
		var zero T
		return zero, false
	}
	val := ch.buffer[ch.off]
	var zero T
	ch.buffer[ch.off] = zero
	ch.off++

	if ch.off > 0 && ch.off >= cap(ch.buffer)/2 {
		remaining := len(ch.buffer) - ch.off
		newBuf := make([]T, remaining)
		copy(newBuf, ch.buffer[ch.off:])
		ch.buffer = newBuf
		ch.off = 0
	}

	return val, true
}
