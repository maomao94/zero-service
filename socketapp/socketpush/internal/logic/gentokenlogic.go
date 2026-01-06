package logic

import (
	"context"
	"time"
	"zero-service/common/ctxdata"
	"zero-service/socketapp/socketpush/internal/svc"
	"zero-service/socketapp/socketpush/socketpush"

	"github.com/golang-jwt/jwt/v4"
	"github.com/songzhibin97/gkit/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type GenTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGenTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GenTokenLogic {
	return &GenTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 生成token
func (l *GenTokenLogic) GenToken(in *socketpush.GenTokenReq) (*socketpush.GenTokenRes, error) {
	if len(in.Uid) == 0 {
		return nil, errors.BadRequest("uid is not empty", "参数错误")
	}
	now := time.Now().Unix()
	accessExpire := l.svcCtx.Config.JwtAuth.AccessExpire
	accessToken, err := l.getJwtToken(l.svcCtx.Config.JwtAuth.AccessSecret, now, accessExpire, in.Uid, in.Payload)
	if err != nil {
		return nil, err
	}
	return &socketpush.GenTokenRes{
		AccessToken:  accessToken,
		AccessExpire: now + accessExpire,
		RefreshAfter: now + accessExpire/2,
	}, nil
}

const (
	jwtAudience  = "aud"
	jwtExpire    = "exp"
	jwtId        = "jti"
	jwtIssueAt   = "iat"
	jwtIssuer    = "iss"
	jwtNotBefore = "nbf"
	jwtSubject   = "sub"
)

func (l *GenTokenLogic) getJwtToken(secretKey string, iat, seconds int64, uid string, payload map[string]string) (string, error) {
	claims := make(jwt.MapClaims)
	claims["exp"] = iat + seconds
	claims["iat"] = iat
	claims[ctxdata.CtxKeyUID] = uid
	if payload != nil && len(payload) > 0 {
		for k, v := range payload {
			if k == "" {
				continue
			}
			switch k {
			case jwtAudience, jwtExpire, jwtId, jwtIssueAt, jwtIssuer, jwtNotBefore, jwtSubject, ctxdata.CtxKeyUID:
				// ignore the standard claims
			default:
				claims[k] = v
			}
		}
	}
	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = claims
	return token.SignedString([]byte(secretKey))
}
