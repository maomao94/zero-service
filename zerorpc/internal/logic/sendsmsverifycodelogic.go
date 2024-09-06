package logic

import (
	"context"
	"fmt"
	"github.com/duke-git/lancet/v2/random"
	"github.com/songzhibin97/gkit/errors"
	"zero-service/zerorpc/internal/svc"
	"zero-service/zerorpc/zerorpc"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendSMSVerifyCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendSMSVerifyCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendSMSVerifyCodeLogic {
	return &SendSMSVerifyCodeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 发送手机号验证码
func (l *SendSMSVerifyCodeLogic) SendSMSVerifyCode(in *zerorpc.SendSMSVerifyCodeReq) (*zerorpc.SendSMSVerifyCodeRes, error) {
	code := random.RandNumeral(6)
	if l.svcCtx.Config.Mode != "prd" {
		code = "888888"
	}
	key := fmt.Sprintf("%s:%s:%s", l.svcCtx.Config.Name, in.Mobile, "smsCode")
	b, _ := l.svcCtx.RedisClient.SetnxExCtx(l.ctx, key, code, 60*3)
	if !b {
		return nil, errors.BadRequest("9999", "验证码保存错误")
	}
	return &zerorpc.SendSMSVerifyCodeRes{
		Code: code,
	}, nil
}
