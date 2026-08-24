package logic

import (
	"context"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type RecordQueryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRecordQueryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecordQueryLogic {
	return &RecordQueryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 查询录制配置
func (l *RecordQueryLogic) RecordQuery(in *oryxserver.RecordQueryReq) (*oryxserver.RecordQueryRes, error) {
	data, err := l.svcCtx.OryxClient.RecordQuery(l.ctx)
	if err != nil {
		l.Logger.Errorf("调用 Oryx API 失败: %v", err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "调用 Oryx API 失败")
	}
	return &oryxserver.RecordQueryRes{
		All:          data.All,
		Home:         data.Home,
		Globs:        data.Globs,
		ProcessCpDir: data.ProcessCpDir,
	}, nil
}
