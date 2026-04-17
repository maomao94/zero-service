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

type ResumeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewResumeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResumeLogic {
	return &ResumeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Resume 把 aisolo ResumeStream 的每一帧直接写到 HTTP 响应, 语义同 Chat。
// 请求体的 Action 决定恢复类型, 具体字段 (reason / selectedIds / text / formValues) 与 kind 对应。
func (l *ResumeLogic) Resume(req *types.SoloInterruptRequest, w io.Writer) error {
	userID := ctxdata.GetUserId(l.ctx)
	if userID == "" {
		return errors.New("missing user id in context")
	}
	if strings.TrimSpace(req.InterruptId) == "" {
		return errors.New("interruptId is required")
	}

	rpcReq := &aisolo.ResumeReq{
		SessionId:   req.SessionId,
		UserId:      userID,
		InterruptId: req.InterruptId,
		Action:      parseResumeAction(req.Action),
		Reason:      req.Reason,
		SelectedIds: req.SelectedIds,
		Text:        req.Text,
		FormValues:  req.FormValues,
	}

	stream, err := l.svcCtx.AiSoloCli.ResumeStream(l.ctx, rpcReq)
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
		// aisolo 侧 protocol.Encode 尾部自带 '\n' (NDJSON), 这里要套 SSE 帧格式,
		// 先 TrimRight 掉换行符, 否则会变成三个换行 (不符合 SSE 规范)。
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
