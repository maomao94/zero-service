package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type RecordGlobsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRecordGlobsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecordGlobsLogic {
	return &RecordGlobsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新录制 glob 过滤器
func (l *RecordGlobsLogic) RecordGlobs(in *oryxserver.RecordGlobsReq) (*oryxserver.RecordGlobsRes, error) {
	err := l.svcCtx.OryxClient.RecordGlobs(l.ctx, in.Globs)
	if err != nil {
		l.Logger.Errorf("调用 Oryx API 失败: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 Oryx API 失败")
	}
	return &oryxserver.RecordGlobsRes{}, nil
}
