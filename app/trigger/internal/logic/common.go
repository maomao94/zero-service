package logic

import (
	"context"
	"time"
	"zero-service/app/trigger/internal/svc"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/dromara/carbon/v2"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/jsonx"
)

// prepareEnqueue 公共入队准备逻辑，提取 sendtriggerlogic 和 sendprototriggerlogic 重复代码
func prepareEnqueue(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
	msgId string,
	maxRetry int64,
	triggerTime string,
	processIn uint64,
	msg any,
) ([]asynq.Option, []byte, error) {
	opts := []asynq.Option{}

	if len(msgId) == 0 {
		var err error
		msgId, err = tool.SimpleUUID()
		if err != nil {
			return nil, nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_03_MQ, err, "生成消息ID失败")
		}
		// 由于 msg 是接口，我们不能直接设置 MsgId，需要调用方自己设置
	}
	opts = append(opts, asynq.TaskID(msgId))

	payload, err := jsonx.Marshal(msg)
	if err != nil {
		return nil, nil, tool.NewErrorByPbCode(extproto.Code__1_03_MQ, "序列化消息失败")
	}

	err = svcCtx.Validate.Struct(msg)
	if err != nil {
		return nil, nil, err
	}

	if maxRetry > 0 {
		opts = append(opts, asynq.MaxRetry(int(maxRetry)))
	}
	opts = append(opts, asynq.Queue("critical"), asynq.Retention(7*24*time.Hour))

	var d time.Duration
	if len(triggerTime) > 0 {
		t := carbon.Parse(triggerTime)
		if t.Error != nil {
			return nil, nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "triggerTime格式错误")
		}
		internal := carbon.Now().DiffInSeconds(t)
		if internal < 0 {
			return nil, nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM, "triggerTime is invalid")
		}
		d = time.Duration(internal) * time.Second
	} else {
		d = time.Duration(processIn) * time.Second
	}
	opts = append(opts, asynq.ProcessIn(d))

	return opts, payload, nil
}
