package svc

import (
	"crypto/tls"
	"net/http"
	"time"

	"zero-service/app/live/internal/config"
	"zero-service/app/live/model/gormmodel"
	"zero-service/common/gormx"
	"zero-service/common/livekitx"
	"zero-service/common/tool"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpc"
)

type ServiceContext struct {
	Config config.Config
	// LiveKit SDK 客户端，业务直接调用 SDK API
	LiveKit *livekitx.Client
	// DB 会议单据与参会记录（pgsql）
	DB *gormx.DB
	// Redis 基础能力：锁（RedisLock）、webhook 幂等、业务编号（IdUtil）
	Redis *redis.Redis
	// IdUtil 会议号等业务编号生成（Redis 序号，category=live 防与其他业务撞号）
	IdUtil *tool.IdUtil
	// MeetingRepo 会议与参会记录存取
	MeetingRepo *MeetingRepo
}

func NewServiceContext(c config.Config) *ServiceContext {
	logx.Must(logx.SetUp(c.Log))
	svcCtx := &ServiceContext{Config: c}

	// LiveKit client：注入 go-zero httpc.Service（底层 transport 忽略 TLS
	// 校验，兼容自签证书的 https，观测复用 httpc）；不注入时 SDK 走内部
	// 容错传输
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // 开发/内网自签证书环境
		},
	}
	lk, err := livekitx.New(
		livekitx.WithURL(c.LiveKit.Url),
		livekitx.WithAPIKey(c.LiveKit.ApiKey, c.LiveKit.ApiSecret),
		livekitx.WithHTTPService(httpc.NewServiceWithClient("httpc-livekit", httpClient)),
	)
	if err != nil {
		logx.Must(err)
	}
	svcCtx.LiveKit = lk

	// 数据库（会议单据与参会记录）
	db := gormx.MustOpenWithConf(c.DB)
	db.MustAutoMigrate(&gormmodel.LiveMeeting{}, &gormmodel.LiveMeetingParticipant{}, &gormmodel.LiveMeetingMessage{})
	svcCtx.DB = db

	// Redis（锁 / 幂等 / 序号），与 oryxserver 一致使用 go-zero 原生 client
	redisClient := redis.MustNewRedis(c.Redis.RedisConf)
	svcCtx.Redis = redisClient
	svcCtx.IdUtil = tool.NewIdUtil(redisClient)

	svcCtx.MeetingRepo = NewMeetingRepo(db)
	return svcCtx
}
