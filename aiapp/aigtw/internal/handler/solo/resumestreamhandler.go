package solo

import (
	"io"

	"net/http"

	"zero-service/aiapp/aigtw/internal/svc"
	"zero-service/aiapp/aisolo/aisolo"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ResumeStreamHandler(serverCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		interruptID := r.PathValue("id")

		var in aisolo.ResumeReq
		if err := httpx.Parse(r, &in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		in.InterruptId = interruptID

		streamClient, err := serverCtx.AiSoloCli.ResumeStream(r.Context(), &in)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Transfer-Encoding", "chunked")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		for {
			resp, err := streamClient.Recv()
			if err != nil {
				if err != io.EOF {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				break
			}

			if resp.Chunk != nil {
				if resp.Chunk.Data != "" {
					data := resp.Chunk.Data
					if data != "" {
						w.Write([]byte("data: " + data + "\n\n"))
						flusher.Flush()
					}
				}

				if resp.Chunk.IsFinal {
					w.Write([]byte("data: [DONE]\n\n"))
					flusher.Flush()
					break
				}
			}
		}
	}
}
