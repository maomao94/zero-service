package handler

import (
	"context"

	"zero-service/common/isp"

	"github.com/zeromicro/go-zero/core/logx"
)

func HandleModelUpdateReport(ctx context.Context, msg *isp.Message) error {
	logx.WithContext(ctx).Infof("[ispagent] 模型更新上报 code=%s items=%d", msg.Code, len(msg.Items))
	for i, item := range msg.Items {
		typ := item["type"]
		fp := item["file_path"]
		logx.WithContext(ctx).Infof("[ispagent] 模型[%d] type=%s(%s) file_path=%s",
			i, typ, modelTypeName[typ], fp)
	}
	return nil
}
