package svc

import (
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

	// LiveKit client（复用 go-zero httpc.Service 的传输与观测配置）
	httpClient := &http.Client{Timeout: 10 * time.Second}
	httpService := httpc.NewServiceWithClient("httpc-livekit", httpClient)
	lk, err := livekitx.New(
		livekitx.WithURL(c.LiveKit.Url),
		livekitx.WithAPIKey(c.LiveKit.ApiKey, c.LiveKit.ApiSecret),
		livekitx.WithHTTPService(httpService),
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