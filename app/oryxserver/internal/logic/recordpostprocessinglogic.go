package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type RecordPostProcessingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRecordPostProcessingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecordPostProcessingLogic {
	return &RecordPostProcessingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新录制后处理配置
func (l *RecordPostProcessingLogic) RecordPostProcessing(in *oryxserver.RecordPostProcessingReq) (*oryxserver.RecordPostProcessingRes, error) {
	err := l.svcCtx.OryxClient.RecordPostProcessing(l.ctx, in.PostProcess, in.PostCpDir)
	if err != nil {
		l.Logger.Errorf("调用 Oryx API 失败: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 Oryx API 失败")
	}
	return &oryxserver.RecordPostProcessingRes{}, nil
}
