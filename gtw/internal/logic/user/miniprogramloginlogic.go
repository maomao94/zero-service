package user

import (
	"context"
	"zero-service/zerorpc/zerorpc"

	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MiniProgramLoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 小程序登录
func NewMiniProgramLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MiniProgramLoginLogic {
	return &MiniProgramLoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *MiniProgramLoginLogic) MiniProgramLogin(req *types.MiniProgramLoginRequest) (resp *types.MiniProgramLoginReply, err error) {
	res, err := l.svcCtx.ZeroRpcCli.MiniProgramLogin(l.ctx, &zerorpc.MiniProgramLoginReq{
		Code: req.Code,
	})
	if err != nil {
		return nil, err
	}
	return &types.MiniProgramLoginReply{
		OpenId:     res.OpenId,
		UnionId:    res.UnionId,
		SessionKey: res.SessionKey,
	}, nil
}
