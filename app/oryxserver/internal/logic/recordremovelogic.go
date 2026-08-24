package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type RecordRemoveLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRecordRemoveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecordRemoveLogic {
	return &RecordRemoveLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除录制文件
func (l *RecordRemoveLogic) RecordRemove(in *oryxserver.RecordRemoveReq) (*oryxserver.RecordRemoveRes, error) {
	if in.Uuid == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "录制任务 UUID 不能为空")
	}
	err := l.svcCtx.OryxClient.RecordRemove(l.ctx, in.Uuid)
	if err != nil {
		l.Logger.Errorf("调用 Oryx API 失败: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 Oryx API 失败")
	}
	return &oryxserver.RecordRemoveRes{}, nil
}
