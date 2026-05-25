package logic

import (
	"context"
	"fmt"
	"github.com/duke-git/lancet/v2/compare"
	"github.com/duke-git/lancet/v2/random"
	"zero-service/common/tool"
	"zero-service/model"
	"zero-service/third_party/extproto"
	"zero-service/zerorpc/internal/svc"
	"zero-service/zerorpc/zerorpc"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LoginLogic) Login(in *zerorpc.LoginReq) (*zerorpc.LoginRes, error) {
	var userId int64
	if compare.Equal(in.AuthType, "miniProgram") {
		responseGetUserPhoneNumber, err := l.svcCtx.MiniCli.PhoneNumber.GetUserPhoneNumber(l.ctx, in.AuthKey)
		if err != nil {
			return nil, err
		}
		if responseGetUserPhoneNumber.ErrCode != 0 {
			l.Errorf("小程序手机号快速登录失败 %v,%v", responseGetUserPhoneNumber.ErrCode, responseGetUserPhoneNumber.ErrMsg)
			return nil, tool.NewErrorByPbCode(extproto.Code__1_06_THIRD_PARTY, "小程序手机号快速登录失败")
		}
		phoneNumber := responseGetUserPhoneNumber.PhoneInfo.PhoneNumber
		u, err := l.svcCtx.UserModel.FindOneByMobile(l.ctx, phoneNumber)
		if err != nil && err != model.ErrNotFound {
			return nil, err
		}
		if u != nil {
			userId = u.Id
		}
	} else if compare.Equal(in.AuthType, "mobile") {
		key := fmt.Sprintf("%s:%s:%s", l.svcCtx.Config.Name, in.AuthKey, "smsCode")
		val, err := l.svcCtx.RedisClient.GetCtx(l.ctx, key)
		if err != nil {
			return nil, err
		}
		if !compare.Equal(val, in.Password) {
			return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "手机号验证码错误")
		}
		_, err = l.svcCtx.RedisClient.Del(key)
		if err != nil {
			return nil, err
		}
		u, err := l.svcCtx.UserModel.FindOneByMobile(l.ctx, in.AuthKey)
		if err != nil && err != model.ErrNotFound {
			return nil, err
		}
		if u != nil {
			userId = u.Id
		}
	} else if compare.Equal(in.AuthType, "unionId") {
		responseCode2Session, err := l.svcCtx.MiniCli.Auth.Session(l.ctx, in.AuthKey)
		if err != nil {
			return nil, err
		}
		if responseCode2Session.ErrCode != 0 {
			l.Errorf("小程序unionId快速登录失败 %v,%v", responseCode2Session.ErrCode, responseCode2Session.ErrMsg)
			return nil, tool.NewErrorByPbCode(extproto.Code__1_06_THIRD_PARTY, "小程序unionId快速登录失败")
		}
		// todo
		return nil, tool.NewErrorByPbCode(extproto.Code__1_05_BIZ, "未保存 unionId")
	} else {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "未知授权类型")
	}
	if userId == 0 {
		nU, err := l.svcCtx.UserModel.Insert(l.ctx, nil, &model.User{
			Mobile:   in.AuthKey,
			Password: "",
			Nickname: random.RandNumeralOrLetter(8),
			Sex:      0,
			Avatar:   "",
			Info:     "",
		})
		if err != nil {
			return nil, err
		}
		userId, _ = nU.LastInsertId()
	}
	generateTokenLogic := NewGenerateTokenLogic(l.ctx, l.svcCtx)
	generateTokenRes, err := generateTokenLogic.GenerateToken(&zerorpc.GenerateTokenReq{
		UserId: userId,
	})
	if err != nil {
		return nil, err
	}
	return &zerorpc.LoginRes{
		AccessToken:  generateTokenRes.AccessToken,
		AccessExpire: generateTokenRes.AccessExpire,
		RefreshAfter: generateTokenRes.RefreshAfter,
	}, nil
}
