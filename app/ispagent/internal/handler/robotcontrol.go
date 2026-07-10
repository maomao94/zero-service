package handler

import (
	"context"
	"fmt"

	"zero-service/common/isp"

	"github.com/zeromicro/go-zero/core/logx"
)

// HandleRobotControl 处理机器人控制指令（server→client，Types 1~4, 21~23）。
// 记录指令日志后返回成功；实际硬件控制在后续阶段接入。
func HandleRobotControl(ctx context.Context, msg *isp.Message) error {
	cmdName := ""
	if names, ok := robotControlNameByType[msg.Type]; ok {
		if n, ok := names[msg.Command]; ok {
			cmdName = n
		}
	}
	if cmdName == "" {
		cmdName = fmt.Sprintf("未知命令(%d)", msg.Command)
	}
	logx.WithContext(ctx).Infof("[ispagent] 机器人控制 type=%d command=%d(%s) code=%s items=%d",
		msg.Type, msg.Command, cmdName, msg.Code, len(msg.Items))
	return nil
}
