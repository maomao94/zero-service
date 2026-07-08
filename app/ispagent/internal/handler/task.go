package handler

import (
	"context"

	"zero-service/common/isp"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/zeromicro/go-zero/core/logx"
)

// HandleTaskDispatch 处理任务下发指令 (101-1)。返回 error 表示处理失败。
func HandleTaskDispatch(ctx context.Context, msg *isp.Message) error {
	logx.WithContext(ctx).Infof("[ispagent] 任务下发 code=%s items=%d", msg.Code, len(msg.Items))
	for i, item := range msg.Items {
		deviceList := strutil.SplitAndTrim(item["device_list"], ",")
		logx.WithContext(ctx).Infof("[ispagent] 任务[%d] task_code=%s task_name=%s priority=%s device_level=%s device_list_size=%v",
			i, item["task_code"], item["task_name"], item["priority"], item["device_level"], len(deviceList))
	}
	return nil
}

// ResponseCode 根据 error 返回 ISP 状态码。
func ResponseCode(err error) string {
	if err != nil {
		return isp.StatusError
	}
	return isp.StatusSuccess
}
