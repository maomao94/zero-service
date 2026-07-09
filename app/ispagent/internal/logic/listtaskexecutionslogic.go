package logic

import (
	"context"
	"time"

	"zero-service/app/ispagent/internal/svc"
	"zero-service/app/ispagent/ispagent"
	"zero-service/common/crontask"

	"github.com/dromara/carbon/v2"
	"github.com/teambition/rrule-go"
	"github.com/zeromicro/go-zero/core/logx"
)

type ListTaskExecutionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListTaskExecutionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTaskExecutionsLogic {
	return &ListTaskExecutionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListTaskExecutionsLogic) ListTaskExecutions(in *ispagent.ListTaskExecutionsReq) (*ispagent.ListTaskExecutionsRes, error) {
	if l.svcCtx.Store == nil {
		return nil, crontask.ErrNotFound
	}

	task, err := l.svcCtx.Store.GetByCode(l.ctx, in.GetTaskCode())
	if err != nil {
		return nil, err
	}

	count := int(in.GetCount())
	if count <= 0 {
		count = 20
	}

	times := l.computeExecTimes(task, count)
	return &ispagent.ListTaskExecutionsRes{
		TaskCode:  task.TaskCode,
		TaskName:  task.TaskName,
		RruleStr:  task.RRuleStr,
		ExecTimes: times,
	}, nil
}

func (l *ListTaskExecutionsLogic) computeExecTimes(task *crontask.TaskConfig, count int) []string {
	if task.RRuleStr == "" {
		return nil
	}

	set, err := rrule.StrToRRuleSet(task.RRuleStr)
	if err != nil {
		return nil
	}

	from := carbon.Now().StdTime().Add(-time.Second)
	var times []string
	for i := 0; i < count; i++ {
		next := set.After(from, false)
		if next.IsZero() {
			break
		}
		times = append(times, carbon.CreateFromStdTime(next).ToDateTimeString())
		from = next
	}
	return times
}
