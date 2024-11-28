package svc

import (
	"fmt"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/payment"
	"github.com/go-playground/validator/v10"
	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/admin/guns"
	"zero-service/app/file/file"
	"zero-service/common"
	interceptor "zero-service/common/Interceptor/rpcclient"
	"zero-service/gtw/internal/config"
	"zero-service/zerorpc/zerorpc"
)

type ServiceContext struct {
	Config      config.Config
	Validate    *validator.Validate
	ZeroRpcCli  zerorpc.ZerorpcClient
	FileRpcCLi  file.FileRpcClient
	AdminRpcCli guns.AdminClient
	WxPayCli    *payment.Payment
}

func NewServiceContext(c config.Config) *ServiceContext {
	paymentService, err := payment.NewPayment(&payment.UserConfig{
		AppID:       "wx59cf6fd0bfe7b63a",               // 小程序、公众号或者企业微信的appid
		MchID:       "1652357329",                       // 商户号 appID
		MchApiV3Key: "30FF88C53AD6D47B608571445F5C2JWY", // 微信V3接口调用必填
		//Key:         "",         // 微信V2接口调用必填
		CertPath: "gtw/etc/wechat/apiclient_cert.pem",        // 商户后台支付的Cert证书路径
		KeyPath:  "gtw/etc/wechat/apiclient_key.pem",         // 商户后台支付的Key证书路径
		SerialNo: "58EB98EF7FEE884F0129BA63C5FD9F51549FEAA7", // 商户支付证书序列号
		//CertificateKeyPath: "[certificate_key_path]",                   // 微信支付平台证书的Key证书路径,m微信V3,[选填]
		//WechatPaySerial:    "[wechat_pay_serial]",                                                        // 微信支付平台证书序列号,微信V3，[选填]
		//RSAPublicKeyPath:   "[wx_rsa_public_key_path]",                                                   // 商户支付证书序列号,微信V2，[选填]
		//SubMchID:           "[sub_mch_id]",                                                               // 服务商平台下的子商户号Id，[选填]
		//SubAppID:           "[syb_appid]",                                                                // 服务商平台下的子AppId，[选填]
		NotifyURL: "http://zero-service/gtw/v1/pay/wechat/notify",
		HttpDebug: true,
		Log: payment.Log{
			Driver: &common.PowerWechatLogDriver{},
		},
		Http: payment.Http{
			Timeout: 30.0,
			BaseURI: "https://api.mch.weixin.qq.com",
		},
		// 可选，不传默认走程序内存
		//Cache: kernel.NewRedisClient(&kernel.UniversalOptions{
		//	Addrs:    []string{"127.0.0.1:6379"},
		//	Password: "",
		//	DB:       0,
		//}),
	})
	if err != nil {
		panic(fmt.Errorf("微信支付初始化错误,%v", err))
	}
	return &ServiceContext{
		Config:   c,
		Validate: validator.New(),
		ZeroRpcCli: zerorpc.NewZerorpcClient(zrpc.MustNewClient(c.ZeroRpcConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor)).Conn()),
		FileRpcCLi: file.NewFileRpcClient(zrpc.MustNewClient(c.FileRpcConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor)).Conn()),
		AdminRpcCli: guns.NewAdminClient(zrpc.MustNewClient(c.AdminRpcConf,
			zrpc.WithUnaryClientInterceptor(interceptor.UnaryMetadataInterceptor)).Conn()),
		WxPayCli: paymentService,
	}
}
