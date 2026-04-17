// useSSE: 简化组件中的 SSE 流式对接.
// 返回 { running, start, stop } — 每次 start 会关闭上一次流.
import { useCallback, useRef, useState, useEffect } from "../lib/deps.js";
import { streamPost } from "../api/sse.js";

export function useSSE() {
  const [running, setRunning] = useState(false);
  const ctrlRef = useRef(null);

  const stop = useCallback(() => {
    if (ctrlRef.current) { ctrlRef.current.abort(); ctrlRef.current = null; }
    setRunning(false);
  }, []);

  const start = useCallback((url, body, handlers = {}) => {
    stop();
    setRunning(true);
    const ctrl = streamPost(url, body, {
      onEvent: handlers.onEvent,
      onError: (err) => {
        setRunning(false);
        handlers.onError && handlers.onError(err);
      },
      onClose: () => {
        setRunning(false);
        ctrlRef.current = null;
        handlers.onClose && handlers.onClose();
      },
    });
    ctrlRef.current = ctrl;
  }, [stop]);

  useEffect(() => () => stop(), [stop]);

  return { running, start, stop };
}
