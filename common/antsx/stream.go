package antsx

import (
	"errors"
	"io"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
)

// streamItem 是流中传输的单个数据单元，携带数据块和可选错误。
type streamItem[T any] struct {
	chunk T
	err   error
}

// stream 是基于 Go channel 的流式传输管道，采用双 channel 架构（借鉴 eino 框架）：
//   - items: 带缓冲的数据通道，实现背压控制
//   - closed: 无缓冲的关闭信号通道，接收端关闭时广播通知发送端
//
// closedFlag 使用 atomic CAS 保证 closeRecv 的幂等性，避免多次 close(closed) 导致 panic。
// sendClosedFlag 使用 atomic CAS 保证 closeSend 的幂等性。
type stream[T any] struct {
	items          chan streamItem[T]
	closed         chan struct{}
	automaticClose bool
	closedFlag     uint32
	sendClosedFlag uint32
}

// newStream 创建指定缓冲容量的 stream 实例。
func newStream[T any](cap int) *stream[T] {
	return &stream[T]{
		items:  make(chan streamItem[T], cap),
		closed: make(chan struct{}),
	}
}

// send 向流中发送一个数据块。返回 true 表示接收端已关闭，发送方应停止发送。
// 使用两阶段 select 模式：先非阻塞快速检查关闭状态，再阻塞式发送同时监听关闭信号。
func (s *stream[T]) send(chunk T, err error) (closed bool) {
	// 阶段1：非阻塞快速检查
	select {
	case <-s.closed:
		return true
	default:
	}
	// 阶段2：阻塞发送，同时监听关闭信号
	item := streamItem[T]{chunk, err}
	select {
	case <-s.closed:
		return true
	case s.items <- item:
		return false
	}
}

// recv 从流中接收一个数据块。当 items channel 关闭时返回 io.EOF。
func (s *stream[T]) recv() (T, error) {
	item, ok := <-s.items
	if !ok {
		var zero T
		return zero, io.EOF
	}
	return item.chunk, item.err
}

// closeSend 关闭发送端（items channel），通知接收方数据已全部发送完毕。
// 使用 atomic CAS 保证幂等：多次调用不会 panic，仅第一次实际执行 close。
func (s *stream[T]) closeSend() {
	if atomic.CompareAndSwapUint32(&s.sendClosedFlag, 0, 1) {
		close(s.items)
	}
}

// closeRecv 关闭接收端（closed channel），通知发送方接收端已退出。
// 使用 atomic CAS 保证幂等：多次调用不会 panic，仅第一次实际执行 close。
func (s *stream[T]) closeRecv() {
	if atomic.CompareAndSwapUint32(&s.closedFlag, 0, 1) {
		close(s.closed)
	}
}

// Pipe 创建一对关联的 StreamReader 和 StreamWriter，构成流式管道。
// cap 指定内部 channel 缓冲容量，控制背压行为：
//   - cap > 0: 生产者可以超前消费者 cap 个元素
//   - cap = 0: 完全同步，每次 Send 都等待 Recv
//
// 用法:
//
//	sr, sw := antsx.Pipe[string](10)
//	go func() {
//	    defer sw.Close()
//	    sw.Send("chunk", nil)
//	}()
//	defer sr.Close()
//	val, err := sr.Recv() // "chunk", nil
func Pipe[T any](cap int) (*StreamReader[T], *StreamWriter[T]) {
	s := newStream[T](cap)
	return &StreamReader[T]{typ: readerStream, st: s}, &StreamWriter[T]{st: s}
}

// StreamWriter 是流管道的写入端，通过 Send 发送数据，通过 Close 通知读取端数据结束。
type StreamWriter[T any] struct {
	st *stream[T]
}

// Send 向流中发送一个值和可选错误。返回 true 表示读取端已关闭，应停止发送。
func (sw *StreamWriter[T]) Send(val T, err error) (closed bool) {
	return sw.st.send(val, err)
}

// Close 关闭写入端，读取端后续 Recv 将收到 io.EOF。
func (sw *StreamWriter[T]) Close() {
	sw.st.closeSend()
}

// readerType 标识 StreamReader 内部的具体读取器类型，用于多态分发。
type readerType int

const (
	readerStream  readerType = iota // 基于 channel 的标准流
	readerArray                     // 基于数组的伪流
	readerMulti                     // 多流合并
	readerConvert                   // 带类型转换的装饰器流
	readerChild                     // Copy 产生的子流
)

// reader 是内部读取器接口，统一不同类型读取器的调用方式。
type reader[T any] interface {
	recv() (T, error)
	close()
}

// StreamReader 是流管道的读取端，支持多种底层实现（stream/array/multi/convert/child）。
// 通过 Recv 逐个读取数据，通过 Close 释放资源。
//
// 协程安全说明：单个 StreamReader 不应在多个 goroutine 中并发调用 Recv。
// 如需多消费者，请使用 Copy 创建独立的子流。
type StreamReader[T any] struct {
	typ readerType
	st  *stream[T]
	ar  *arrayReader[T]
	mr  *multiReader[T]
	cr  *convertReader[T]
	csr *childReader[T]
}

// Recv 从流中读取下一个元素。到达末尾时返回 io.EOF。
func (sr *StreamReader[T]) Recv() (T, error) {
	switch sr.typ {
	case readerStream:
		return sr.st.recv()
	case readerArray:
		return sr.ar.recv()
	case readerMulti:
		return sr.mr.recv()
	case readerConvert:
		return sr.cr.recv()
	case readerChild:
		return sr.csr.recv()
	default:
		panic("antsx: unknown reader type")
	}
}

// Close 关闭读取端并释放底层资源。
// 对于 stream 类型，关闭后发送端的 Send 将立即返回 closed=true。
func (sr *StreamReader[T]) Close() {
	switch sr.typ {
	case readerStream:
		sr.st.closeRecv()
	case readerArray:
	case readerMulti:
		sr.mr.close()
	case readerConvert:
		sr.cr.close()
	case readerChild:
		sr.csr.close()
	}
}

// SetAutomaticClose 通过 runtime.SetFinalizer 注册 GC 回收时自动关闭。
// 作为兜底的资源清理机制，防止调用方忘记调用 Close 导致 goroutine 泄漏。
// 应在 StreamReader 创建后立即调用，不应在多个 goroutine 中并发调用。
func (sr *StreamReader[T]) SetAutomaticClose() {
	switch sr.typ {
	case readerStream:
		sr.st.automaticClose = true
		runtime.SetFinalizer(sr, func(r *StreamReader[T]) {
			r.Close()
		})
	case readerMulti:
		for _, idx := range sr.mr.nonClosed {
			st := sr.mr.sts[idx]
			st.automaticClose = true
		}
		runtime.SetFinalizer(sr, func(r *StreamReader[T]) {
			r.Close()
		})
	case readerConvert:
		sr.cr.sr.SetAutomaticClose()
	case readerChild:
	case readerArray:
	}
}

// Copy 将一个 StreamReader 复制为 n 个独立的子流（fan-out）。
// 每个子流独立消费完整的数据序列，底层使用链表 + sync.Once 实现零拷贝广播。
// 所有子流关闭后，原始流自动关闭。
// n < 2 时直接返回原始 reader。
func (sr *StreamReader[T]) Copy(n int) []*StreamReader[T] {
	if n < 2 {
		return []*StreamReader[T]{sr}
	}
	if sr.typ == readerArray {
		ret := make([]*StreamReader[T], n)
		for i, ar := range sr.ar.copy(n) {
			ret[i] = &StreamReader[T]{typ: readerArray, ar: ar}
		}
		return ret
	}
	return copyReaders[T](sr, n)
}

// iStreamReader 是类型擦除的 StreamReader 接口，用于跨泛型边界传递流。
type iStreamReader interface {
	recvAny() (any, error)
	copyAny(int) []iStreamReader
	Close()
	SetAutomaticClose()
}

// streamReaderAny 将泛型 StreamReader[T] 包装为 iStreamReader 接口，实现类型擦除。
type streamReaderAny[T any] struct {
	sr *StreamReader[T]
}

func (s *streamReaderAny[T]) recvAny() (any, error) {
	return s.sr.Recv()
}

func (s *streamReaderAny[T]) Close() {
	s.sr.Close()
}

func (s *streamReaderAny[T]) SetAutomaticClose() {
	s.sr.SetAutomaticClose()
}

func (s *streamReaderAny[T]) copyAny(n int) []iStreamReader {
	copies := s.sr.Copy(n)
	ret := make([]iStreamReader, n)
	for i, c := range copies {
		ret[i] = &streamReaderAny[T]{sr: c}
	}
	return ret
}

// StreamReaderFromArray 将切片包装为 StreamReader，按顺序逐个返回元素。
// 到达末尾时 Recv 返回 io.EOF。适用于将已有数据转换为流式接口。
func StreamReaderFromArray[T any](arr []T) *StreamReader[T] {
	return &StreamReader[T]{typ: readerArray, ar: &arrayReader[T]{arr: arr}}
}

// arrayReader 基于切片实现的流读取器，通过 index 游标逐个返回元素。
type arrayReader[T any] struct {
	arr   []T
	index int
}

func (ar *arrayReader[T]) recv() (T, error) {
	if ar.index < len(ar.arr) {
		val := ar.arr[ar.index]
		ar.index++
		return val, nil
	}
	var zero T
	return zero, io.EOF
}

func (ar *arrayReader[T]) close() {}

// copy 创建 n 个共享底层数组但各自独立游标的 arrayReader 副本。
func (ar *arrayReader[T]) copy(n int) []*arrayReader[T] {
	ret := make([]*arrayReader[T], n)
	for i := 0; i < n; i++ {
		ret[i] = &arrayReader[T]{arr: ar.arr, index: ar.index}
	}
	return ret
}

// toStream 将 arrayReader 剩余元素转换为 stream，用于 Merge 等操作。
func (ar *arrayReader[T]) toStream() *stream[T] {
	s := newStream[T](len(ar.arr) - ar.index)
	for i := ar.index; i < len(ar.arr); i++ {
		s.send(ar.arr[i], nil)
	}
	s.closeSend()
	return s
}

// remaining 返回尚未读取的剩余元素切片。
func (ar *arrayReader[T]) remaining() []T {
	if ar.index >= len(ar.arr) {
		return nil
	}
	return ar.arr[ar.index:]
}

// ConvertOption 是 StreamReaderWithConvert 的函数式选项。
type ConvertOption func(*convertOpts)

type convertOpts struct {
	errWrapper func(error) error
}

// WithErrWrapper 设置上游错误的包装函数。
// 当 errWrapper 返回 nil 时，该错误将被跳过；返回非 nil 时，该错误传递给调用方。
func WithErrWrapper(fn func(error) error) ConvertOption {
	return func(o *convertOpts) { o.errWrapper = fn }
}

// StreamReaderWithConvert 创建一个带类型转换和过滤功能的 StreamReader。
// fn 对每个元素执行转换：
//   - 返回 (val, nil): 输出 val
//   - 返回 (_, ErrNoValue): 跳过当前元素，继续读取下一个
//   - 返回 (_, otherErr): 将 otherErr 传递给调用方
func StreamReaderWithConvert[T, D any](sr *StreamReader[T], fn func(T) (D, error), opts ...ConvertOption) *StreamReader[D] {
	opt := &convertOpts{}
	for _, o := range opts {
		o(opt)
	}
	return &StreamReader[D]{
		typ: readerConvert,
		cr: &convertReader[D]{
			sr:         &streamReaderAny[T]{sr: sr},
			convert:    func(a any) (D, error) { return fn(a.(T)) },
			errWrapper: opt.errWrapper,
		},
	}
}

// convertReader 带类型转换功能的流读取器，支持元素过滤（ErrNoValue 跳过）。
type convertReader[T any] struct {
	sr         iStreamReader
	convert    func(any) (T, error)
	errWrapper func(error) error
}

// recv 读取上游数据并执行转换，ErrNoValue 的元素会被自动跳过。
func (cr *convertReader[T]) recv() (T, error) {
	for {
		out, err := cr.sr.recvAny()
		if err != nil {
			var zero T
			if err == io.EOF {
				return zero, err
			}
			if cr.errWrapper != nil {
				err = cr.errWrapper(err)
				if err != nil {
					return zero, err
				}
				continue
			}
			return zero, err
		}
		val, err := cr.convert(out)
		if err == nil {
			return val, nil
		}
		if !isErrNoValue(err) {
			return val, err
		}
	}
}

func isErrNoValue(err error) bool {
	return errors.Is(err, ErrNoValue)
}

func (cr *convertReader[T]) close() {
	cr.sr.Close()
}

// toStream 将任意类型的 StreamReader 转换为底层 stream，在后台 goroutine 中读取转发。
// panic 会被捕获并通过流传递给消费端，goroutine 退出时自动关闭源 reader。
func toStream[T any](sr *StreamReader[T]) *stream[T] {
	ret := newStream[T](5)
	go func() {
		defer func() {
			if p := recover(); p != nil {
				ret.send(*new(T), newPanicErr(p))
			}
			ret.closeSend()
			sr.Close()
		}()
		for {
			val, err := sr.Recv()
			if err == io.EOF {
				break
			}
			if ret.send(val, err) {
				break
			}
		}
	}()
	return ret
}

// toStream 将 StreamReader 转换为底层 stream 实例。
// stream 类型直接返回内部 stream；array 类型转换为带缓冲的 stream；
// 其他类型启动后台 goroutine 转发。
func (sr *StreamReader[T]) toStream() *stream[T] {
	switch sr.typ {
	case readerStream:
		return sr.st
	case readerArray:
		return sr.ar.toStream()
	default:
		return toStream[T](sr)
	}
}

// MergeStreamReaders 将多个 StreamReader 合并为一个（fan-in），按数据到达顺序交错读取。
// 优化策略：
//   - arrayReader 直接拼接为数组，避免不必要的 goroutine
//   - multiReader 自动展平，避免嵌套
//   - ≤5 路使用静态 select（零 reflect 开销），>5 路降级为 reflect.Select
//
// 返回 nil 表示输入为空。单个输入直接返回原始 reader。
func MergeStreamReaders[T any](srs []*StreamReader[T]) *StreamReader[T] {
	if len(srs) == 0 {
		return nil
	}
	if len(srs) == 1 {
		return srs[0]
	}

	var arrays []T
	var nonArrays []*stream[T]

	for _, sr := range srs {
		switch sr.typ {
		case readerArray:
			remaining := sr.ar.remaining()
			if len(remaining) > 0 {
				arrays = append(arrays, remaining...)
			}
		case readerMulti:
			for _, idx := range sr.mr.nonClosed {
				nonArrays = append(nonArrays, sr.mr.sts[idx])
			}
		default:
			nonArrays = append(nonArrays, sr.toStream())
		}
	}

	if len(arrays) > 0 {
		arrStream := newStream[T](len(arrays))
		for _, v := range arrays {
			arrStream.send(v, nil)
		}
		arrStream.closeSend()
		nonArrays = append(nonArrays, arrStream)
	}

	if len(nonArrays) == 0 {
		return nil
	}
	if len(nonArrays) == 1 {
		return &StreamReader[T]{typ: readerStream, st: nonArrays[0]}
	}

	return &StreamReader[T]{typ: readerMulti, mr: newMultiReader(nonArrays, nil)}
}

// MergeNamedStreamReaders 将多个具名 StreamReader 合并为一个。
// 当某条源流结束时，Recv 会返回 SourceEOF 错误，调用方可通过 GetSourceName 获取源名称。
// 所有源流结束后 Recv 返回 io.EOF。
func MergeNamedStreamReaders[T any](srs map[string]*StreamReader[T]) *StreamReader[T] {
	if len(srs) == 0 {
		return nil
	}

	ss := make([]*stream[T], 0, len(srs))
	names := make([]string, 0, len(srs))

	for name, sr := range srs {
		ss = append(ss, sr.toStream())
		names = append(names, name)
	}

	return &StreamReader[T]{typ: readerMulti, mr: newMultiReader(ss, names)}
}

// multiReader 多流合并读取器，从多个 stream 中按到达顺序交错读取。
// ≤maxSelectNum 路使用静态 select（receiveN），超过时使用 reflect.Select。
// reflect 路径下关闭的 case 通过置零 Chan 标记（参考 eino），避免每次重建切片的 GC 开销。
type multiReader[T any] struct {
	sts               []*stream[T]
	cases             []reflect.SelectCase // reflect 路径的全量 case 数组，关闭后 Chan 置零
	nonClosed         []int
	sourceReaderNames []string
}

// newMultiReader 创建多流合并读取器。names 可选，非 nil 时启用按来源追踪 EOF。
func newMultiReader[T any](sts []*stream[T], names []string) *multiReader[T] {
	nonClosed := make([]int, len(sts))
	for i := range sts {
		nonClosed[i] = i
	}

	mr := &multiReader[T]{
		sts:               sts,
		nonClosed:         nonClosed,
		sourceReaderNames: names,
	}

	if len(sts) > maxSelectNum {
		cases := make([]reflect.SelectCase, len(sts))
		for i, st := range sts {
			cases[i] = reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(st.items),
			}
		}
		mr.cases = cases
	}

	return mr
}

// recv 从多个源流中读取下一个到达的数据。
// 在同一个循环中根据 nonClosed 长度动态选择静态 select 或 reflect.Select 路径。
// 当某条流关闭后立即移除并（如果有 names）返回 SourceEOF，然后继续循环尝试下一条。
func (mr *multiReader[T]) recv() (T, error) {
	for len(mr.nonClosed) > 0 {
		var (
			realIdx int
			ok      bool
		)

		if len(mr.nonClosed) > maxSelectNum {
			// reflect.Select 路径：直接对全量 cases 做 select（已关闭的 Chan 为零值不会被选中）
			chosen, recv, recvOk := reflect.Select(mr.cases)
			realIdx = chosen
			ok = recvOk
			if ok {
				item := recv.Interface().(streamItem[T])
				return item.chunk, item.err
			}
			// 流关闭：将对应 case 的 Chan 置零，下次 reflect.Select 不再选中
			mr.cases[chosen].Chan = reflect.Value{}
		} else {
			// 静态 select 路径
			var item *streamItem[T]
			realIdx, item, ok = receiveN(mr.nonClosed, mr.sts)
			if ok {
				return item.chunk, item.err
			}
		}

		// 流关闭：从活跃列表中移除
		mr.removeFromNonClosed(realIdx)
		if mr.sourceReaderNames != nil {
			var zero T
			return zero, &SourceEOF{source: mr.sourceReaderNames[realIdx]}
		}
	}
	var zero T
	return zero, io.EOF
}

// removeFromNonClosed 从活跃流索引列表中移除已关闭的流。
func (mr *multiReader[T]) removeFromNonClosed(realIdx int) {
	for i, idx := range mr.nonClosed {
		if idx == realIdx {
			mr.nonClosed = append(mr.nonClosed[:i], mr.nonClosed[i+1:]...)
			return
		}
	}
}

// close 关闭所有底层 stream 的接收端。
func (mr *multiReader[T]) close() {
	for _, st := range mr.sts {
		st.closeRecv()
	}
}

// cpElement 是流 Copy 操作中的链表节点。
// 每个节点通过 sync.Once 保证数据只从源流读取一次，
// 后续到达同一节点的子流直接读取缓存值，实现零拷贝广播。
type cpElement[T any] struct {
	once sync.Once
	next *cpElement[T]
	item streamItem[T]
}

// parentReader 管理 Copy 产生的所有子流，维护每个子流在链表中的读取位置。
// 通过 atomic 计数跟踪已关闭子流数量，所有子流关闭后自动关闭源流。
type parentReader[T any] struct {
	sr        *StreamReader[T]
	cursors   []*cpElement[T]
	closedNum uint32
}

// copyReaders 将一个 StreamReader 复制为 n 个共享源流的子 reader。
// 使用链表 + sync.Once 模式：第一个到达某节点的子流执行实际 Recv，
// 后续子流直接读取缓存数据，天然并发安全。
func copyReaders[T any](sr *StreamReader[T], n int) []*StreamReader[T] {
	elem := &cpElement[T]{}
	parent := &parentReader[T]{
		sr:      sr,
		cursors: make([]*cpElement[T], n),
	}
	for i := range parent.cursors {
		parent.cursors[i] = elem
	}
	ret := make([]*StreamReader[T], n)
	for i := range ret {
		ret[i] = &StreamReader[T]{
			typ: readerChild,
			csr: &childReader[T]{parent: parent, index: i},
		}
	}
	return ret
}

// peek 读取指定子流的下一个元素。
// sync.Once 保证每个节点的数据只从源流读取一次，多个子流可安全并发读取同一节点。
func (p *parentReader[T]) peek(idx int) (T, error) {
	elem := p.cursors[idx]
	if elem == nil {
		var zero T
		return zero, ErrRecvAfterClosed
	}
	elem.once.Do(func() {
		val, err := p.sr.Recv()
		elem.item = streamItem[T]{chunk: val, err: err}
		if err != io.EOF {
			elem.next = &cpElement[T]{}
		}
	})
	t, err := elem.item.chunk, elem.item.err
	if err != io.EOF {
		p.cursors[idx] = elem.next
	}
	return t, err
}

// closeChild 关闭指定子流。所有子流关闭后自动关闭源流。
// 使用 atomic 计数保证并发安全。
func (p *parentReader[T]) closeChild(idx int) {
	if p.cursors[idx] == nil {
		return
	}
	p.cursors[idx] = nil
	cur := atomic.AddUint32(&p.closedNum, 1)
	if int(cur) == len(p.cursors) {
		p.sr.Close()
	}
}

// childReader 是 Copy 产生的子流读取器，通过 parentReader 共享源流数据。
type childReader[T any] struct {
	parent *parentReader[T]
	index  int
}

func (cr *childReader[T]) recv() (T, error) {
	return cr.parent.peek(cr.index)
}

func (cr *childReader[T]) close() {
	cr.parent.closeChild(cr.index)
}
