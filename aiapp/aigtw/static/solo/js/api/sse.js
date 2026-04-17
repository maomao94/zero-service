// 基于 fetch + ReadableStream 的 SSE 客户端.
// 为什么不直接用 EventSource?
//   1. EventSource 不支持自定义 header, 我们需要带 Authorization.
//   2. EventSource 不支持 POST body.
// 协议: 每个 data: 帧后面都是一个完整 JSON 对象 (NDJSON over SSE).
// 调用方提供 onEvent(ev), 框架负责拼分割帧和 JSON 解码.

import { getToken } from "./client.js";

export function streamPost(url, body, { onEvent, onError, onClose, signal }) {
  const headers = { "Content-Type": "application/json", "Accept": "text/event-stream" };
  const tok = getToken();
  if (tok) headers["Authorization"] = `Bearer ${tok}`;

  const ctrl = new AbortController();
  // 若外部传入 signal, 也串进去
  if (signal) {
    if (signal.aborted) ctrl.abort();
    else signal.addEventListener("abort", () => ctrl.abort(), { once: true });
  }

  (async () => {
    try {
      const res = await fetch(url, {
        method: "POST",
        headers,
        body: body ? JSON.stringify(body) : undefined,
        signal: ctrl.signal,
      });
      if (!res.ok) {
        let msg = `HTTP ${res.status}`;
        try { const j = await res.json(); msg = j.msg || j.message || msg; } catch (_) {}
        throw new Error(msg);
      }
      if (!res.body) throw new Error("no response body");

      const reader = res.body.getReader();
      const decoder = new TextDecoder("utf-8");
      let buffer = "";

      for (;;) {
        const { value, done } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });

        let sepIdx;
        while ((sepIdx = buffer.indexOf("\n\n")) >= 0) {
          const frame = buffer.slice(0, sepIdx);
          buffer = buffer.slice(sepIdx + 2);
          const line = frame.split("\n").find((l) => l.startsWith("data:"));
          if (!line) continue;
          const payload = line.slice(5).trim();
          if (!payload || payload === "[DONE]") continue;
          try {
            const ev = JSON.parse(payload);
            onEvent && onEvent(ev);
          } catch (err) {
            console.warn("[sse] bad frame", err, payload);
          }
        }
      }
      onClose && onClose();
    } catch (err) {
      if (err.name === "AbortError") { onClose && onClose(); return; }
      onError && onError(err);
    }
  })();

  return {
    abort: () => ctrl.abort(),
  };
}
