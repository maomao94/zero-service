package solo

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"zero-service/aiapp/aisolo/aisolo"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/common/ctxdata"
	"zero-service/common/ssex"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChatLogic struct {
	Logger logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	w      http.ResponseWriter
	r      *http.Request
}

func NewChatLogic(ctx context.Context, svcCtx *svc.ServiceContext, w http.ResponseWriter, r *http.Request) *ChatLogic {
	return &ChatLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		w:      w,
		r:      r,
	}
}

func (l *ChatLogic) Chat(req *types.SoloChatReq) error {
	sw, err := ssex.NewWriter(l.w)
	if err != nil {
		return err
	}

	// 获取用户ID
	userID := ctxdata.GetUserId(l.ctx)
	if userID == "" {
		userID = "anonymous"
	}

	l.Logger.Infof("solo chat stream started, sessionId: %s, userID: %s", req.SessionId, userID)

	// 启动 keep-alive goroutine
	kaStop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-kaStop:
				return
			case <-ticker.C:
				sw.WriteKeepAlive()
			}
		}
	}()
	defer close(kaStop)

	// 构建 gRPC 请求
	protoReq := &aisolo.AskRequest{
		SessionId: req.SessionId,
		UserId:    userID,
		Message:   req.Message,
		AgentMode: l.toAgentMode(req.AgentMode),
	}

	// 调用 gRPC 流
	stream, err := l.svcCtx.EinoCli.AskStream(l.ctx, protoReq)
	if err != nil {
		l.Logger.Errorf("AskStream gRPC call failed: %v", err)
		return err
	}

	// 直接透传 A2UI NDJSON 数据
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			sw.WriteDone()
			l.Logger.Infof("solo stream completed, sessionId: %s, userID: %s", req.SessionId, userID)
			return nil
		}
		if err != nil {
			l.Logger.Errorf("stream recv error: %v", err)
			return nil
		}

		// 透传 gRPC 层的 A2UI JSON
		if chunk.Data != "" {
			sw.Write([]byte(chunk.Data))
		}

		// 检查客户端断开
		if l.r.Context().Err() != nil {
			l.Logger.Infof("stream client disconnected, sessionId: %s, userID: %s", req.SessionId, userID)
			return nil
		}
	}
}

// toAgentMode 转换 Agent 模式
func (l *ChatLogic) toAgentMode(mode string) aisolo.AgentMode {
	switch mode {
	case "fast":
		return aisolo.AgentMode_AGENT_MODE_FAST
	case "deep":
		return aisolo.AgentMode_AGENT_MODE_DEEP
	default:
		return aisolo.AgentMode_AGENT_MODE_AUTO
	}
}
