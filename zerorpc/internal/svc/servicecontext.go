package svc

import (
	"fmt"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/miniProgram"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/payment"
	"github.com/hibiken/asynq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/rest/httpc"
	"github.com/zeromicro/go-zero/zrpc"
	"zero-service/common/powerwechatx"
	"zero-service/model"
	"zero-service/zeroalarm/zeroalarm"
	"zero-service/zerorpc/internal/config"
)

type ServiceContext struct {
	Config       config.Config
	AsynqClient  *asynq.Client
	AsynqServer  *asynq.Server
	Scheduler    *asynq.Scheduler
	Httpc        httpc.Service
	RedisClient  *redis.Redis
	ZeroAlarmCli zeroalarm.ZeroalarmClient
	MiniCli      *miniProgram.MiniProgram
	WxPayCli     *payment.Payment

	UserModel     model.UserModel
	RegionModel   model.RegionModel
	OrderTxnModel model.OrderTxnModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	redisClient := redis.MustNewRedis(c.Redis.RedisConf)
	// 小程序配置
	miniCli, err := miniProgram.NewMiniProgram(&miniProgram.UserConfig{
		AppID:     c.MiniProgram.AppId,  // 小程序appid
		Secret:    c.MiniProgram.Secret, // 小程序app secret
		HttpDebug: false,
		Log: miniProgram.Log{
			Driver: &powerwechatx.PowerWechatLogDriver{},
		},
		// 可选，不传默认走程序内存
		//Cache: kernel.NewRedisClient(&kernel.UniversalOptions{
		//	Addrs:    []string{"127.0.0.1:6379"},
		//	Password: "",
		//	DB:       0,
		//}),
	})
	if err != nil {
		panic(fmt.Errorf("微信小程序初始化错误,%v", err))
	}
	paymentService, err := payment.NewPayment(&payment.UserConfig{
		AppID:       "wx59cf6fd0bfe7b63a",               // 小程序、公众号或者企业微信的appid
		MchID:       "1652357329",                       // 商户号 appID
		MchApiV3Key: "30FF88C53AD6D47B608571445F5C2JWY", // 微信V3接口调用必填
		//Key:         "",         // 微信V2接口调用必填
		CertPath: "zerorpc/etc/wechat/apiclient_cert.pem",    // 商户后台支付的Cert证书路径
		KeyPath:  "zerorpc/etc/wechat/apiclient_key.pem",     // 商户后台支付的Key证书路径
		SerialNo: "58EB98EF7FEE884F0129BA63C5FD9F51549FEAA7", // 商户支付证书序列号
		//CertificateKeyPath: "[certificate_key_path]",                   // 微信支付平台证书的Key证书路径,m微信V3,[选填]
		//WechatPaySerial:    "[wechat_pay_serial]",                                                        // 微信支付平台证书序列号,微信V3，[选填]
		//RSAPublicKeyPath:   "[wx_rsa_public_key_path]",                                                   // 商户支付证书序列号,微信V2，[选填]
		//SubMchID:           "[sub_mch_id]",                                                               // 服务商平台下的子商户号Id，[选填]
		//SubAppID:           "[syb_appid]",                                                                // 服务商平台下的子AppId，[选填]
		NotifyURL: "http://zero-service/gtw/v1/pay/wechat/notify",
		HttpDebug: true,
		Log: payment.Log{
			Driver: &powerwechatx.PowerWechatLogDriver{},
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
		Config:        c,
		AsynqClient:   newAsynqClient(c),
		AsynqServer:   newAsynqServer(c),
		Scheduler:     newScheduler(c),
		Httpc:         httpc.NewService("httpc"),
		RedisClient:   redisClient,
		ZeroAlarmCli:  zeroalarm.NewZeroalarmClient(zrpc.MustNewClient(c.ZeroAlarmConf).Conn()),
		MiniCli:       miniCli,
		WxPayCli:      paymentService,
		UserModel:     model.NewUserModel(sqlx.NewMysql(c.DB.DataSource)),
		RegionModel:   model.NewRegionModel(sqlx.NewMysql(c.DB.DataSource)),
		OrderTxnModel: model.NewOrderTxnModel(sqlx.NewMysql(c.DB.DataSource)),
	}
}
