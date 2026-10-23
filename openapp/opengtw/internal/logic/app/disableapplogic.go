package app

import (
	"context"
	"strings"

	"zero-service/openapp/opengtw/internal/svc"
	"zero-service/openapp/opengtw/internal/types"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	xerrors "github.com/zeromicro/x/errors"
)

type DisableAppLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 启用或停用应用
func NewDisableAppLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DisableAppLogic {
	return &DisableAppLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DisableAppLogic) DisableApp(req *types.DisableAppRequest) (*types.DisableAppReply, error) {
	if req == nil {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_MISSING), "请求不能为空")
	}
	appID := strings.TrimSpace(req.AppId)
	if appID == "" {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_MISSING), "appId 不能为空")
	}
	if req.Status != 1 && req.Status != 2 {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_INVALID), "status 只能是 1（启用）或 2（停用）")
	}

	rows, err := l.svcCtx.AppStore.UpdateStatus(l.ctx, appID, req.Status)
	if err != nil {
		l.Errorf("update open app status failed, app_id=%s: %v", appID, err)
		return nil, xerrors.New(int(extproto.Code__1_00_DB), "更新应用状态失败")
	}
	if rows == 0 {
		return nil, xerrors.New(int(extproto.Code__1_00_RECORD_NOT_EXIST), "应用不存在")
	}

	l.svcCtx.RefreshCredentials(l.ctx)
	return &types.DisableAppReply{AppId: appID, Status: req.Status}, nil
}
