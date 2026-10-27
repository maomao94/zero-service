package app

import (
	"context"
	"strings"

	"zero-service/app/opengtw/internal/svc"
	"zero-service/app/opengtw/internal/types"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	xerrors "github.com/zeromicro/x/errors"
)

type ListAppsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 应用列表
func NewListAppsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppsLogic {
	return &ListAppsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAppsLogic) ListApps(req *types.ListAppsRequest) (*types.ListAppsReply, error) {
	if req == nil {
		req = &types.ListAppsRequest{}
	}
	if req.Status != 0 && req.Status != 1 && req.Status != 2 {
		return nil, xerrors.New(int(extproto.Code__1_00_PARAM_INVALID), "status 只能是 0/1/2")
	}

	pageNum, pageSize := normalizePage(req.PageNum, req.PageSize)
	apps, total, err := l.svcCtx.AppStore.FindPage(l.ctx, strings.TrimSpace(req.Keyword), req.Status, pageNum, pageSize)
	if err != nil {
		l.Errorf("list open apps failed: %v", err)
		return nil, xerrors.New(int(extproto.Code__1_00_DB), "查询应用列表失败")
	}

	items := make([]types.AppItem, 0, len(apps))
	for _, app := range apps {
		items = append(items, types.AppItem{
			AppId:        app.AppId,
			Name:         app.Name,
			Remark:       app.Remark,
			Status:       app.Status,
			SignRequired: app.SignRequired,
			LastUsedAt:   formatTime(app.LastUsedAt),
			CreatedAt:    formatTime(app.CreatedAt),
		})
	}
	return &types.ListAppsReply{
		Items:    items,
		Total:    total,
		PageNum:  pageNum,
		PageSize: pageSize,
	}, nil
}
