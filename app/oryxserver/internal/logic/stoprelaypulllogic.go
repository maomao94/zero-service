package logic

import (
	"context"
	"strings"
	"time"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/encoding/protojson"
)

type StopRelayPullLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStopRelayPullLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopRelayPullLogic {
	return &StopRelayPullLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// StopRelayPull 停止中继拉流（gRPC 入口）：
// 广播失败时入队补停任务（Asynq 异步重试）。
func (l *StopRelayPullLogic) StopRelayPull(in *oryxserver.StopRelayPullReq) (*oryxserver.StopRelayPullRes, error) {
	res, err := l.stopRelayPullOnce(in)
	if err != nil {
		// 停止失败 → 入队补停（Asynq 重试调用 StopRelayPullFromAsynq）
		l.Logger.Infof("停止失败，入队补停: app=%s stream=%s err=%v", in.App, in.Stream, err)
		// 读取当前 UUID 用于补停校验
		st, _ := l.svcCtx.StateStore.GetState(l.ctx, in.App, in.Stream)
		uuid := ""
		if st != nil {
			uuid = st.UUID
		}
		if enqueueErr := l.svcCtx.RelayRegistry.EnqueueStop(l.ctx, in.App, in.Stream, uuid); enqueueErr != nil {
			l.Logger.Errorf("入队补停失败: app=%s stream=%s err=%v", in.App, in.Stream, enqueueErr)
		}
		return &oryxserver.StopRelayPullRes{}, nil
	}
	return res, nil
}

// StopRelayPullFromAsynq 从 Asynq 补停任务调用（不入队，失败由 Asynq 重试）。
func (l *StopRelayPullLogic) StopRelayPullFromAsynq(app, stream string) error {
	_, err := l.stopRelayPullOnce(&oryxserver.StopRelayPullReq{App: app, Stream: stream})
	return err
}

// stopRelayPullOnce 停止中继拉流（核心逻辑，不入队）：
// 1. 确认是否存在（本地 + Redis）
// 2. 分布式停止（锁 → 清理 Redis → 停本地进程）
// 3. 本地未命中 + 集群模式 → 广播其他节点停止
// 广播失败返回 error，由调用方决定重试策略。
func (l *StopRelayPullLogic) stopRelayPullOnce(in *oryxserver.StopRelayPullReq) (*oryxserver.StopRelayPullRes, error) {
	if strings.TrimSpace(in.App) == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "app 不能为空")
	}
	if strings.TrimSpace(in.Stream) == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "stream 不能为空")
	}

	app := strings.TrimSpace(in.App)
	stream := strings.TrimSpace(in.Stream)

	// 1. 确认是否存在（本地进程 + Redis 状态）
	localExists := l.svcCtx.RelayRegistry.HasTarget(app, stream)
	st, stateErr := l.svcCtx.StateStore.GetState(l.ctx, app, stream)
	if stateErr != nil {
		l.Logger.Errorf("获取 Redis 状态失败（按不存在处理）: app=%s stream=%s err=%v", app, stream, stateErr)
	}
	redisExists := st != nil

	if !localExists && !redisExists {
		l.Logger.Infof("中继不存在，跳过停止: app=%s stream=%s", app, stream)
		return &oryxserver.StopRelayPullRes{}, nil
	}

	l.Logger.Infof("确认中继存在: app=%s stream=%s local=%v redis=%v", app, stream, localExists, redisExists)

	// 2. 分布式停止（锁 → 清理 Redis state+lease → 停本地进程）
	if l.svcCtx.RelayRegistry.StopRelay(l.ctx, app, stream) {
		l.Logger.Infof("本地进程已停止: app=%s stream=%s", app, stream)
		return &oryxserver.StopRelayPullRes{}, nil
	}

	// 3. 本地未命中 + 集群模式 → 广播其他节点停止
	if !l.svcCtx.IsBroadcast() || l.svcCtx.MqttClient == nil {
		return &oryxserver.StopRelayPullRes{}, nil
	}

	payload, err := protojson.Marshal(&oryxserver.StopRelayPullReq{App: in.App, Stream: in.Stream})
	if err != nil {
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "编码停止请求失败")
	}

	_, broadcastErr := l.svcCtx.Broadcaster.BroadcastReply(l.ctx, oryxserver.OryxServer_StopRelayPull_FullMethodName,
		payload, 10*time.Second)

	if broadcastErr != nil {
		return nil, broadcastErr
	}

	l.Logger.Infof("集群广播停止成功: app=%s stream=%s", app, stream)
	return &oryxserver.StopRelayPullRes{}, nil
}
