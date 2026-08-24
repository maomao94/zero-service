package logic

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/url"
	"strings"

	"zero-service/app/oryxserver/internal/svc"
	"zero-service/app/oryxserver/oryxserver"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/zeromicro/go-zero/core/logx"
)

type StreamRelayLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStreamRelayLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StreamRelayLogic {
	return &StreamRelayLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 启动 FFmpeg 转推（拉取源流 → copy 转推 SRS；任务按节点本地内存管理）
func (l *StreamRelayLogic) StreamRelay(in *oryxserver.StreamRelayReq) (*oryxserver.StreamRelayRes, error) {
	if strings.TrimSpace(in.SourceUrl) == "" {
		return nil, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "source_url 不能为空")
	}
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
	target := strings.TrimRight(rc.SrsRtmpAddr, "/") + "/" + app + "/" + stream
	authQuery, err := buildAuthQuery(rc.AuthStyle, rc.Secret, rc.PushKey)
	if err != nil {
		return nil, err
	}
	if authQuery != "" {
		target += "?" + authQuery
	}

	taskID, err := l.svcCtx.RelayManager.Start(in.SourceUrl, target)
	if err != nil {
		l.Logger.Errorf("启动转推失败: source=%s target=%s err=%v", in.SourceUrl, target, err)
		return nil, tool.NewErrorByPbCodeWrap(extproto.Code__1_06_THIRD_PARTY, err, "启动转推失败")
	}
	l.Logger.Infof("启动转推成功: task_id=%s source=%s target=%s", taskID, in.SourceUrl, target)
	return &oryxserver.StreamRelayRes{
		TaskId: taskID,
		App:    app,
		Stream: stream,
	}, nil
}

// buildAuthQuery 按鉴权模式构建推流 URL 查询参数：
//   - secret：SRS/Oryx 风格，追加 ?secret=xxx
//   - sign：WVP/ZLM 风格，追加 ?sign=md5(pushkey)（小写 32 位）
//   - none：不追加
func buildAuthQuery(authStyle, secret, pushKey string) (string, error) {
	q := url.Values{}
	switch authStyle {
	case "sign":
		if strings.TrimSpace(pushKey) == "" {
			return "", tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_MISSING, "pushkey 不能为空")
		}
		q.Set("sign", fmt.Sprintf("%x", md5.Sum([]byte(pushKey))))
	case "secret":
		if secret != "" {
			q.Set("secret", secret)
		}
	}
	return q.Encode(), nil
}
