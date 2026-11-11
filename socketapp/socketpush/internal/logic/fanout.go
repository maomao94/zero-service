package logic

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
	"zero-service/socketapp/socketgtw/socketgtw"

	"github.com/zeromicro/go-zero/core/logx"
	"golang.org/x/sync/errgroup"
)

// defaultPushTimeout 是对全部网关节点的推送总预算，Config.PushTimeout 未配置时生效。
const defaultPushTimeout = 2 * time.Second

// PushFanoutError 表示一次网关扇出推送中有节点失败。
// 调用方可用 errors.As 取出失败节点，决定重试或告警；重试可能造成重复推送。
type PushFanoutError struct {
	Total  int
	Failed []string
	Err    error
}

func (e *PushFanoutError) Error() string {
	return fmt.Sprintf("push failed on %d/%d gateway(s): %v, first error: %v", len(e.Failed), e.Total, e.Failed, e.Err)
}

func (e *PushFanoutError) Unwrap() error { return e.Err }

// fanoutPush 向已发现的全部网关节点并发推送，并汇总失败节点。
//
// 与 threading.GoSafe 逐个丢弃错误不同：单个节点失败既不影响其他节点，也不会被静默忽略，
// 调用方能拿到具体失败节点。超过 timeout 未完成的调用按失败计。
func fanoutPush(
	ctx context.Context,
	logger logx.Logger,
	clients map[string]socketgtw.SocketGtwClient,
	timeout time.Duration,
	call func(ctx context.Context, client socketgtw.SocketGtwClient) error,
) error {
	if len(clients) == 0 {
		return nil
	}
	if timeout <= 0 {
		timeout = defaultPushTimeout
	}
	// 推送不应因调用方上下文取消而中断，但仍需有界，避免节点异常时长期占用 goroutine。
	baseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()

	var (
		mu       sync.Mutex
		failed   []string
		firstErr error
	)
	recordFailure := func(gateway string, err error) {
		mu.Lock()
		defer mu.Unlock()
		failed = append(failed, gateway)
		if firstErr == nil {
			firstErr = err
		}
	}

	g, gctx := errgroup.WithContext(baseCtx)
	for gateway, client := range clients {
		g.Go(func() error {
			defer func() {
				// 保持与原 GoSafe 行为一致：单个节点的 panic 不拖垮整批推送。
				if r := recover(); r != nil {
					logger.Errorf("socketpush fanout panic: gateway=%s panic=%v", gateway, r)
					recordFailure(gateway, fmt.Errorf("panic: %v", r))
				}
			}()
			if err := call(gctx, client); err != nil {
				logger.Errorf("socketpush fanout failed: gateway=%s err=%v", gateway, err)
				recordFailure(gateway, err)
			}
			return nil
		})
	}
	_ = g.Wait()

	if len(failed) == 0 {
		return nil
	}
	sort.Strings(failed)
	return &PushFanoutError{
		Total:  len(clients),
		Failed: failed,
		Err:    firstErr,
	}
}
