package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type RecordEndLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRecordEndLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecordEndLogic {
	return &RecordEndLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 结束录制任务
func (l *RecordEndLogic) RecordEnd(in *oryxserver.RecordEndReq) (*oryxserver.RecordEndRes, error) {
	if in.Uuid == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "录制任务 UUID 不能为空")
	}
	err := l.svcCtx.OryxClient.RecordEnd(l.ctx, in.Uuid)
	if err != nil {
		l.Logger.Errorf("调用 Oryx API 失败: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 Oryx API 失败")
	}
	return &oryxserver.RecordEndRes{}, nil
}
