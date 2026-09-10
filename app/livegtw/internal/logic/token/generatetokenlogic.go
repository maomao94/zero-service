// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package token

import (
	"context"
	"errors"
	"time"

	"zero-service/app/livegtw/internal/svc"
	"zero-service/app/livegtw/internal/types"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/logx"
)

var (
	errInvalidAuthType = errors.New("authType must be 'user' or 'device'")
	errUserIdRequired  = errors.New("userId is required")
)

type GenerateTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 签发 Token（无需用户鉴权，使用签发密钥验证）
func NewGenerateTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenerateTokenLogic {
	return &GenerateTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GenerateTokenLogic) GenerateToken(req *types.GenerateTokenRequest) (resp *types.GenerateTokenReply, err error) {
	// 防止服务端未配置 SignKey 时被绕过
	serverKey := l.svcCtx.Config.TokenSign.SignKey
	if len(serverKey) == 0 {
		return nil, tool.ErrSignKeyNotConfig
	}

	// 常量时间比较，防止时序攻击
	if !tool.VerifySignKey(serverKey, req.SignKey) {
		return nil, tool.ErrInvalidSignKey
	}

	// 验证认证类型
	if req.AuthType != "user" && req.AuthType != "device" {
		return nil, errInvalidAuthType
	}

	// 验证必填字段
	if len(req.UserId) == 0 {
		return nil, errUserIdRequired
	}

	// 计算过期时间
	expireSeconds := int64(req.ExpireSeconds)
	if expireSeconds <= 0 {
		expireSeconds = int64(l.svcCtx.Config.TokenSign.DefaultExpireSeconds)
		if expireSeconds <= 0 {
			expireSeconds = 3600
		}
	}

	// 根据类型签发 token（device 复用 user-id/user-name，auth-type 区分语义）
	var tokenString string
	var accessExpire, refreshAfter int64

	if req.AuthType == "user" {
		tokenString, accessExpire, refreshAfter, err = tool.GenerateUserToken(
			l.svcCtx.Config.JwtAuth.AccessSecret,
			expireSeconds,
			req.UserId,
			req.UserName,
			req.DeptCode,
		)
	} else {
		tokenString, accessExpire, refreshAfter, err = tool.GenerateDeviceToken(
			l.svcCtx.Config.JwtAuth.AccessSecret,
			expireSeconds,
			req.UserId,
			req.UserName,
		)
	}

	if err != nil {
		return nil, err
	}

	return &types.GenerateTokenReply{
		Token:        tokenString,
		AccessExpire: accessExpire,
		RefreshAfter: refreshAfter,
		ExpireTime:   time.Unix(accessExpire, 0).Format("2006-01-02 15:04:05"),
	}, nil
}
