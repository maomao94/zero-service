package logic

import (
	"fmt"
	"sort"

	"zero-service/common/socketiox"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

// emitDropError 表示有 session 因并发池已满未被投递，消息实际被丢弃。
// 会向上返回给 socketpush，使其能把本网关标记为失败，而不是静默成功。
type emitDropError struct {
	Dropped []string
}

func (e *emitDropError) Error() string {
	return fmt.Sprintf("emit dropped on %d session(s) because concurrency pool is full: %v", len(e.Dropped), e.Dropped)
}

// emitSessions 将消息并发投递给本地 session。
//
//   - 每个 session 只投递一次，不同连接之间并发，不存在同一连接的写序问题；
//   - 并发度由共享 TaskRunner 限制，池满时立即失败而不是阻塞 RPC；
//   - emit 自身失败（连接可能刚断开）只记录日志；调度失败会汇总返回，避免静默丢弃。
func emitSessions(
	runner *threading.TaskRunner,
	logger logx.Logger,
	sessions []*socketiox.Session,
	event string,
	payload any,
	reqId string,
) error {
	if len(sessions) == 0 {
		return nil
	}
	var dropped []string
	for _, session := range sessions {
		s := session
		err := runner.ScheduleImmediately(func() {
			if emitErr := s.EmitDown(event, payload, reqId); emitErr != nil {
				logger.Errorf("emit failed: session=%s reqId=%s err=%v", s.ID(), reqId, emitErr)
			}
		})
		if err != nil {
			dropped = append(dropped, s.ID())
		}
	}
	if len(dropped) == 0 {
		return nil
	}
	sort.Strings(dropped)
	return &emitDropError{Dropped: dropped}
}
