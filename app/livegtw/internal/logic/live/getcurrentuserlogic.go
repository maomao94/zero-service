package live

import (
	"context"

	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"
	"zero-service/common/authctx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCurrentUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCurrentUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCurrentUserLogic {
	return &GetCurrentUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCurrentUserLogic) GetCurrentUser(req *types.GetCurrentUserRequest) (resp *types.GetCurrentUserReply, err error) {
	return &types.GetCurrentUserReply{
		AuthType: authctx.GetAuthType(l.ctx),
		UserId:   authctx.GetUserId(l.ctx),
		UserName: authctx.GetUserName(l.ctx),
		DeptCode: authctx.GetDeptCode(l.ctx),
	}, nil
}
