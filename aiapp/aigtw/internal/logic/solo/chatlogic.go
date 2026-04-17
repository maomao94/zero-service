package solo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aigtw/internal/types"
	"zero-service/aiapp/aisolo/aisolo"
	"zero-service/common/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChatLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChatLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatLogic {
	return &ChatLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Chat 把 aisolo AskStream 的每一帧 (已经是完整的 JSON Event) 直接作为 SSE data: 帧
// 写入 HTTP 响应。这里不经过 channel, 也不再 json.Marshal, 保证 NDJSON over SSE 的完整性。
func (l *ChatLogic) Chat(req *types.SoloChatRequest, w io.Writer) error {
	userID := ctxdata.GetUserId(l.ctx)
	if userID == "" {
		return errors.New("missing user id in context")
	}
	if strings.TrimSpace(req.SessionId) == "" {
		return errors.New("sessionId is required")
	}

	stream, err := l.svcCtx.AiSoloCli.AskStream(l.ctx, &aisolo.AskReq{
		SessionId: req.SessionId,
		UserId:    userID,
		Message:   req.Message,
		Mode:      parseMode(req.Mode),
		Meta:      req.Meta,
	})
	if err != nil {
		return err
	}

	flusher, _ := w.(http.Flusher)
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		chunk := resp.GetChunk()
		if chunk == nil {
			continue
		}
		// aisolo 端用 protocol.Encode 输出, 每帧末尾带了 '\n' 用于 NDJSON。
		// 这里要套 SSE `data: <json>\n\n` 格式, 先把这个尾部换行掉, 避免变成
		// 三个换行 (不符合 SSE 规范, 部分代理会切帧错误)。
		data := strings.TrimRight(chunk.GetData(), "\r\n")
		if data == "" {
			continue
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
		if chunk.GetIsFinal() {
			return nil
		}
	}
}
