// Code generated by goctl. DO NOT EDIT.
// goctl 1.7.2
// Source: zerorpc.proto

package server

import (
	"context"

	"zero-service/zerorpc/internal/logic"
	"zero-service/zerorpc/internal/svc"
	"zero-service/zerorpc/zerorpc"
)

type ZerorpcServer struct {
	svcCtx *svc.ServiceContext
	zerorpc.UnimplementedZerorpcServer
}

func NewZerorpcServer(svcCtx *svc.ServiceContext) *ZerorpcServer {
	return &ZerorpcServer{
		svcCtx: svcCtx,
	}
}

func (s *ZerorpcServer) Ping(ctx context.Context, in *zerorpc.Req) (*zerorpc.Res, error) {
	l := logic.NewPingLogic(ctx, s.svcCtx)
	return l.Ping(in)
}

// 发送延迟任务
func (s *ZerorpcServer) SendDelayTask(ctx context.Context, in *zerorpc.SendDelayTaskReq) (*zerorpc.SendDelayTaskRes, error) {
	l := logic.NewSendDelayTaskLogic(ctx, s.svcCtx)
	return l.SendDelayTask(in)
}

// 转发任务
func (s *ZerorpcServer) ForwardTask(ctx context.Context, in *zerorpc.ForwardTaskReq) (*zerorpc.ForwardTaskRes, error) {
	l := logic.NewForwardTaskLogic(ctx, s.svcCtx)
	return l.ForwardTask(in)
}

// 发送手机号验证码
func (s *ZerorpcServer) SendSMSVerifyCode(ctx context.Context, in *zerorpc.SendSMSVerifyCodeReq) (*zerorpc.SendSMSVerifyCodeRes, error) {
	l := logic.NewSendSMSVerifyCodeLogic(ctx, s.svcCtx)
	return l.SendSMSVerifyCode(in)
}

// 获取区域列表
func (s *ZerorpcServer) GetRegionList(ctx context.Context, in *zerorpc.GetRegionListReq) (*zerorpc.GetRegionListRes, error) {
	l := logic.NewGetRegionListLogic(ctx, s.svcCtx)
	return l.GetRegionList(in)
}

// 生成 token
func (s *ZerorpcServer) GenerateToken(ctx context.Context, in *zerorpc.GenerateTokenReq) (*zerorpc.GenerateTokenRes, error) {
	l := logic.NewGenerateTokenLogic(ctx, s.svcCtx)
	return l.GenerateToken(in)
}

// 登录
func (s *ZerorpcServer) Login(ctx context.Context, in *zerorpc.LoginReq) (*zerorpc.LoginRes, error) {
	l := logic.NewLoginLogic(ctx, s.svcCtx)
	return l.Login(in)
}

// 小程序登录
func (s *ZerorpcServer) MiniProgramLogin(ctx context.Context, in *zerorpc.MiniProgramLoginReq) (*zerorpc.MiniProgramLoginRes, error) {
	l := logic.NewMiniProgramLoginLogic(ctx, s.svcCtx)
	return l.MiniProgramLogin(in)
}

// 用户详情
func (s *ZerorpcServer) GetUserInfo(ctx context.Context, in *zerorpc.GetUserInfoReq) (*zerorpc.GetUserInfoRes, error) {
	l := logic.NewGetUserInfoLogic(ctx, s.svcCtx)
	return l.GetUserInfo(in)
}

// 编辑用户
func (s *ZerorpcServer) EditUserInfo(ctx context.Context, in *zerorpc.EditUserInfoReq) (*zerorpc.EditUserInfoRes, error) {
	l := logic.NewEditUserInfoLogic(ctx, s.svcCtx)
	return l.EditUserInfo(in)
}

// JSAPI支付
func (s *ZerorpcServer) WxPayJsApi(ctx context.Context, in *zerorpc.WxPayJsApiReq) (*zerorpc.WxPayJsApiRes, error) {
	l := logic.NewWxPayJsApiLogic(ctx, s.svcCtx)
	return l.WxPayJsApi(in)
}
