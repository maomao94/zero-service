// Code generated by goctl. DO NOT EDIT.
// goctl 1.7.3

package handler

import (
	"net/http"

	common "zero-service/gtw/internal/handler/common"
	gtw "zero-service/gtw/internal/handler/gtw"
	pay "zero-service/gtw/internal/handler/pay"
	user "zero-service/gtw/internal/handler/user"
	"zero-service/gtw/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			{
				// 获取区域列表
				Method:  http.MethodPost,
				Path:    "/getRegionList",
				Handler: common.GetRegionListHandler(serverCtx),
			},
			{
				// 上传文件
				Method:  http.MethodPost,
				Path:    "/mfs/uploadFile",
				Handler: common.MfsUploadFileHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.JwtAuth.AccessSecret),
		rest.WithPrefix("/app/common/v1"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// forward
				Method:  http.MethodPost,
				Path:    "/forward",
				Handler: gtw.ForwardHandler(serverCtx),
			},
			{
				// 下载文件
				Method:  http.MethodGet,
				Path:    "/mfs/downloadFile",
				Handler: gtw.MfsDownloadFileHandler(serverCtx),
			},
			{
				// ping
				Method:  http.MethodGet,
				Path:    "/ping",
				Handler: gtw.PingHandler(serverCtx),
			},
			{
				// pingJava
				Method:  http.MethodGet,
				Path:    "/pingJava",
				Handler: gtw.PingJavaHandler(serverCtx),
			},
		},
		rest.WithPrefix("/gtw/v1"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 微信支付通知
				Method:  http.MethodPost,
				Path:    "/wechat/paidNotify",
				Handler: pay.PaidNotifyHandler(serverCtx),
			},
			{
				// 微信退款通知
				Method:  http.MethodPost,
				Path:    "/wechat/refundedNotify",
				Handler: pay.RefundedNotifyHandler(serverCtx),
			},
		},
		rest.WithPrefix("/gtw/v1/pay"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 登录
				Method:  http.MethodPost,
				Path:    "/login",
				Handler: user.LoginHandler(serverCtx),
			},
			{
				// 小程序登录
				Method:  http.MethodPost,
				Path:    "/miniProgramLogin",
				Handler: user.MiniProgramLoginHandler(serverCtx),
			},
			{
				// 发送手机号验证码
				Method:  http.MethodPost,
				Path:    "/sendSMSVerifyCode",
				Handler: user.SendSMSVerifyCodeHandler(serverCtx),
			},
		},
		rest.WithPrefix("/app/user/v1"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				// 修改当前用户信息
				Method:  http.MethodPost,
				Path:    "/editCurrentUser",
				Handler: user.EditCurrentUserHandler(serverCtx),
			},
			{
				// 获取用户信息
				Method:  http.MethodGet,
				Path:    "/getCurrentUser",
				Handler: user.GetCurrentUserHandler(serverCtx),
			},
		},
		rest.WithJwt(serverCtx.Config.JwtAuth.AccessSecret),
		rest.WithPrefix("/app/user/v1"),
	)
}
