package logic

import (
	"context"
	"net/url"
	"strings"

	"zero-service/app/oryxserver/internal/relay"
	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/duke-git/lancet/v2/cryptor"
	"github.com/zeromicro/go-zero/core/logx"
)

type StartRelayPullLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStartRelayPullLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartRelayPullLogic {
	return &StartRelayPullLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 启动 FFmpeg 中继拉流（拉取源流 → copy 推送到固定目标 Oryx/SRS；分布式模式下走 Redis 状态 + 租约）
func (l *StartRelayPullLogic) StartRelayPull(in *oryxserver.StartRelayPullReq) (*oryxserver.StartRelayPullRes, error) {
	// app 为空 → 配置默认值；stream 为空 → 自动生成
	app := in.App
	if app == "" {
		app = l.svcCtx.Config.RelayConfig.DefaultApp
	}
	stream := in.Stream
	if stream == "" {
		genStream, err := tool.SimpleUUID()
		if err != nil {
			return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "生成流名失败")
		}
		stream = genStream
	}
	rc := l.svcCtx.Config.RelayConfig
	// source_url 必须提供
	source := strings.TrimSpace(in.SourceUrl)
	if source == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "source_url 不能为空")
	}
	// 目标地址（作为状态/租约的唯一标识）
	target := strings.TrimRight(rc.SrsRtmpAddr, "/") + "/" + app + "/" + stream
	// 鉴权：业务系统自行计算，服务只拼 URL；有值就用，没有用配置默认值
	secretKey := strings.TrimSpace(in.SecretKey)
	if secretKey == "" {
		secretKey = rc.SecretKey
	}
	secretValue := strings.TrimSpace(in.SecretValue)
	if secretValue == "" {
		secretValue = rc.SecretValue
	}
	authQuery := buildAuthQuery(secretKey, secretValue)
	// ffmpeg 推流目标（含鉴权参数）
	relayURL := target
	if authQuery != "" {
		relayURL += "?" + authQuery
	}

	// Redis 状态 + 租约 + 本地进程
	// target = 基础地址（状态/租约标识）；relayURL = 含鉴权参数（ffmpeg 推流目标）
	normalizedTarget := relay.NormalizeTarget(target)
	alreadyRunning, _ := l.svcCtx.StateStore.HasLease(l.ctx, normalizedTarget)
	startedTarget, err := l.svcCtx.DistRelay.StartRelay(l.ctx, source, target, relayURL, in.MaxDurationSeconds)
	if err != nil {
		l.Logger.Errorf("启动中继拉流失败: source=%s target=%s err=%v", source, target, err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "启动中继拉流失败")
	}
	l.Logger.Infof("启动中继拉流成功: target=%s source=%s alreadyRunning=%v", startedTarget, source, alreadyRunning)
	return &oryxserver.StartRelayPullRes{
		RelayId:        cryptor.Md5String(startedTarget),
		App:            app,
		Stream:         stream,
		AlreadyRunning: alreadyRunning,
	}, nil
}

// buildAuthQuery 构建推流 URL 查询参数：key=value（业务系统自行计算，服务不参与）
func buildAuthQuery(key, value string) string {
	if key == "" || value == "" {
		return ""
	}
	return url.QueryEscape(key) + "=" + url.QueryEscape(value)
}
