package user

import (
	"context"
	"zero-service/zerorpc/zerorpc"

	"zero-service/gtw/internal/svc"
	"zero-service/gtw/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendSMSVerifyCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 发送手机号验证码
func NewSendSMSVerifyCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendSMSVerifyCodeLogic {
	return &SendSMSVerifyCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendSMSVerifyCodeLogic) SendSMSVerifyCode(req *types.SendSMSVerifyCodeRequest) (resp *types.SendSMSVerifyCodeReply, err error) {
	res, err := l.svcCtx.ZeroRpcCli.SendSMSVerifyCode(l.ctx, &zerorpc.SendSMSVerifyCodeReq{
		Mobile: req.Mobile,
	})
	if err != nil {
		return nil, err
	}
	return &types.SendSMSVerifyCodeReply{
		Code: res.Code,
	}, nil
}
