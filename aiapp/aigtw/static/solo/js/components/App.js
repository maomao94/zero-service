import { html, useCallback, useEffect, useRef, useState } from "../lib/deps.js";
import { api, getToken, setToken, streamEndpoints } from "../api/client.js";
import { useSSE } from "../hooks/useSSE.js";
import { useToasts } from "../hooks/useToast.js";
import { SessionList } from "./SessionList.js";
import { ChatView } from "./ChatView.js";

// =============================================================================
// Event 接收状态机: 把 protocol.Event 序列拼成可渲染的 messages + 当前 interrupt
// =============================================================================

function applyEvent(state, ev) {
  const msgs = state.messages.slice();
  switch (ev.type) {
    case "turn.start":
      return { ...state, interrupt: null };

    case "message.start": {
      const d = ev.data || {};
      msgs.push({
        id: d.message_id,
        role: d.role || "assistant",
        content: "",
        agent_name: d.agent_name || "",
      });
      return { ...state, messages: msgs };
    }
    case "message.delta": {
      const d = ev.data || {};
      for (let i = msgs.length - 1; i >= 0; i--) {
        if (msgs[i].id === d.message_id) {
          msgs[i] = {
            ...msgs[i],
            content: (msgs[i].content || "") + (d.text || ""),
            agent_name: msgs[i].agent_name || d.agent_name || "",
          };
          return { ...state, messages: msgs };
        }
      }
      msgs.push({
        id: d.message_id,
        role: "assistant",
        content: d.text || "",
        agent_name: d.agent_name || "",
      });
      return { ...state, messages: msgs };
    }
    case "message.end": {
      const d = ev.data || {};
      for (let i = msgs.length - 1; i >= 0; i--) {
        if (msgs[i].id === d.message_id) {
          msgs[i] = {
            ...msgs[i],
            content: d.text || msgs[i].content || "",
            role: d.role || msgs[i].role,
            agent_name: msgs[i].agent_name || d.agent_name || "",
          };
          return { ...state, messages: msgs };
        }
      }
      msgs.push({
        id: d.message_id,
        role: d.role || "assistant",
        content: d.text || "",
        agent_name: d.agent_name || "",
      });
      return { ...state, messages: msgs };
    }

    case "tool.call.start": {
      const d = ev.data || {};
      msgs.push({
        id: `tc:${d.call_id}`,
        role: "tool_call",
        tool: d.tool,
        args: d.args_json,
        agent_name: d.agent_name || "",
      });
      return { ...state, messages: msgs };
    }
    case "tool.call.end": {
      const d = ev.data || {};
      for (let i = msgs.length - 1; i >= 0; i--) {
        if (msgs[i].id === `tc:${d.call_id}`) {
          msgs[i] = {
            ...msgs[i],
            result: d.result,
            error: d.error,
            agent_name: msgs[i].agent_name || d.agent_name || "",
          };
          return { ...state, messages: msgs };
        }
      }
      msgs.push({
        id: `tc:${d.call_id}`,
        role: "tool_call",
        tool: d.tool,
        result: d.result,
        error: d.error,
        agent_name: d.agent_name || "",
      });
      return { ...state, messages: msgs };
    }

    case "interrupt":
      return { ...state, interrupt: ev.data || null };

    case "turn.end":
      return state;

    case "error": {
      const d = ev.data || {};
      msgs.push({ role: "system", content: `错误: ${d.code || ""} ${d.message || ""}` });
      return { ...state, messages: msgs };
    }

    default:
      return state;
  }
}

const UI_LANG_KEY = "solo.uiLang";

function readStoredUILang() {
  try {
    const s = localStorage.getItem(UI_LANG_KEY);
    if (s === "zh" || s === "en") return s;
  } catch (_) { /* ignore */ }
  const n = (typeof navigator !== "undefined" && navigator.language) || "";
  if (String(n).toLowerCase().startsWith("en")) return "en";
  return "zh";
}

// =============================================================================
// 顶层 App 组件
// =============================================================================

export function App() {
  const toasts = useToasts();
  const sse = useSSE();

  const [token, setTokenState] = useState(() => getToken());
  const [modes, setModes] = useState([]);
  const [skills, setSkills] = useState([]);
  const [sessions, setSessions] = useState([]);
  const [currentId, setCurrentId] = useState("");
  const [currentSession, setCurrentSession] = useState(null);
  const [mode, setMode] = useState("agent");
  const [messages, setMessages] = useState([]);
  const [interrupt, setInterrupt] = useState(null);
  const [input, setInput] = useState("");
  const [uiLang, setUiLangState] = useState(() => readStoredUILang());

  // 防止快速切换会话时，较慢的 getSession/listMessages 晚到覆盖当前选中项。
  const pickedSessionRef = useRef("");

  const setUiLang = useCallback((v) => {
    const x = v === "en" ? "en" : "zh";
    try {
      localStorage.setItem(UI_LANG_KEY, x);
    } catch (_) { /* ignore */ }
    setUiLangState(x);
  }, []);

  // --------------- 初始化: modes + sessions ---------------
  // 只依赖稳定的 toasts.push, 避免 render 循环.
  const pushToast = toasts.push;
  const loadModes = useCallback(async () => {
    try {
      const r = await api.listModes();
      const list = r.modes || [];
      setModes(list);
      if (list.length > 0) {
        const def = list.find((m) => m.default) || list[0];
        setMode((prev) => prev || def.mode);
      }
    } catch (err) { pushToast(`加载 Mode 失败: ${err.message}`, "error"); }
  }, [pushToast]);

  const loadSkills = useCallback(async () => {
    try {
      const r = await api.listSkills();
      setSkills(r.skills || []);
    } catch (err) { pushToast(`加载 Skills 失败: ${err.message}`, "error"); }
  }, [pushToast]);

  const loadSessions = useCallback(async () => {
    try {
      const r = await api.listSessions({ page: 1, pageSize: 50 });
      setSessions(r.sessions || []);
    } catch (err) { pushToast(`加载会话失败: ${err.message}`, "error"); }
  }, [pushToast]);

  // 只在 token 变化时加载一次; loadModes/loadSessions 本身稳定不再列入 deps.
  useEffect(() => {
    if (!token) return;
    loadModes();
    loadSkills();
    loadSessions();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  // --------------- 选中 session / 加载消息 ---------------
  // 若会话处于 interrupted 状态, 额外拉一次 GetInterrupt 回填中断面板,
  // 实现 "页面刷新后继续审批/确认" 的体验。
  const pickSession = useCallback(async (id) => {
    if (!id) return;
    pickedSessionRef.current = id;
    sse.stop();
    setCurrentId(id);
    setMessages([]);
    setInterrupt(null);
    try {
      const [sess, msgs] = await Promise.all([
        api.getSession(id),
        api.listMessages(id, 200),
      ]);
      if (pickedSessionRef.current !== id) return;
      const s = sess.session;
      setCurrentSession(s);
      if (s && s.mode) setMode(s.mode);
      const normalized = (msgs.messages || []).map((m) => ({
        id: m.id, role: m.role, content: m.content,
      }));
      setMessages(normalized);

      if (s && s.status === "interrupted" && s.interruptId) {
        try {
          const r = await api.getInterrupt(s.interruptId);
          if (pickedSessionRef.current !== id) return;
          if (r && r.info) {
            // 后端字段是 interruptId / minSelect / maxSelect 等 camelCase,
            // 转换成 protocol.Event 里的 snake_case, 复用 InterruptPanel 渲染逻辑。
            setInterrupt({
              interrupt_id: r.info.interruptId,
              kind:         r.info.kind,
              tool_name:    r.info.toolName,
              required:     r.info.required,
              ui_lang:      r.info.uiLang,
              agent_name:   r.info.agentName || "",
              question:     r.info.question,
              detail:       r.info.detail,
              options:      r.info.options || [],
              min_select:   r.info.minSelect,
              max_select:   r.info.maxSelect,
              placeholder:  r.info.placeholder,
              multiline:    r.info.multiline,
              fields:       r.info.fields || [],
              title:        r.info.title,
              body:         r.info.body,
            });
          }
        } catch (err) {
          if (pickedSessionRef.current === id) {
            pushToast(`加载中断详情失败: ${err.message}`, "error");
          }
        }
      }
    } catch (err) {
      if (pickedSessionRef.current === id) {
        pushToast(`加载会话失败: ${err.message}`, "error");
      }
    }
  }, [pushToast, sse]);

  // --------------- 新建 / 删除 ---------------
  const newSession = useCallback(async () => {
    try {
      const r = await api.createSession({ title: "新会话", mode, uiLang });
      const s = r.session;
      setSessions((list) => [s, ...list]);
      await pickSession(s.sessionId);
    } catch (err) { pushToast(`创建会话失败: ${err.message}`, "error"); }
  }, [mode, uiLang, pickSession, pushToast]);

  const deleteSession = useCallback(async (id) => {
    if (!confirm("确认删除该会话?")) return;
    try {
      await api.deleteSession(id);
      setSessions((list) => list.filter((s) => s.sessionId !== id));
      if (id === currentId) {
        setCurrentId(""); setCurrentSession(null);
        setMessages([]); setInterrupt(null);
      }
      pushToast("已删除", "success");
    } catch (err) { pushToast(`删除失败: ${err.message}`, "error"); }
  }, [currentId, pushToast]);

  // --------------- SSE 事件处理 ---------------
  const onEvent = useCallback((ev) => {
    setMessages((prevMsgs) => {
      setInterrupt((prevIt) => {
        const next = applyEvent({ messages: prevMsgs, interrupt: prevIt }, ev);
        prevMsgs = next.messages;
        return next.interrupt;
      });
      return prevMsgs;
    });
  }, []);

  const refreshCurrent = useCallback(async () => {
    if (!currentId) return;
    try {
      const sess = await api.getSession(currentId);
      setCurrentSession(sess.session);
      setSessions((list) => list.map((s) => (s.sessionId === currentId ? sess.session : s)));
    } catch (_) { /* 非关键 */ }
  }, [currentId]);

  // --------------- 发送消息 ---------------
  // onClose 只刷新当前会话 (状态 / 最后一条消息 / message_count),
  // refreshCurrent 内部已经把 sidebar 里该 session 条目原位更新了,
  // 没必要再整体拉 /sessions 把左栏翻一遍 (老逻辑做这件事纯粹是 UX 冗余)。
  const send = useCallback(() => {
    if (!currentId || !input.trim()) return;
    const streamSessionId = currentId;
    const userText = input;
    setInput("");
    setMessages((list) => [...list, { role: "user", content: userText, id: `u:${Date.now()}` }]);
    const { chat } = streamEndpoints();
    sse.start(
      chat,
      { sessionId: currentId, message: userText, mode, meta: { ui_lang: uiLang } },
      {
        onEvent: (ev) => {
          if (pickedSessionRef.current !== streamSessionId) return;
          onEvent(ev);
        },
        onError: (err) => pushToast(`对话中断: ${err.message}`, "error"),
        onClose: () => { refreshCurrent(); },
      },
    );
  }, [currentId, input, mode, uiLang, sse, onEvent, pushToast, refreshCurrent]);

  // --------------- 中断恢复 ---------------
  const resume = useCallback((payload) => {
    if (!interrupt || !interrupt.interrupt_id) return;
    const streamSessionId = currentId;
    const body = {
      sessionId: currentId,
      action: payload.action,
      reason: payload.reason || "",
      selectedIds: payload.selectedIds || [],
      text: payload.text || "",
      formValues: payload.formValues || {},
    };
    const iid = interrupt.interrupt_id;
    setInterrupt(null);
    const { resume: resumeURL } = streamEndpoints();
    sse.start(
      resumeURL(iid),
      body,
      {
        onEvent: (ev) => {
          if (pickedSessionRef.current !== streamSessionId) return;
          onEvent(ev);
        },
        onError: (err) => pushToast(`恢复失败: ${err.message}`, "error"),
        onClose: () => { refreshCurrent(); },
      },
    );
  }, [interrupt, currentId, sse, onEvent, pushToast, refreshCurrent]);

  // --------------- Mode 切换 ---------------
  const onModeChange = useCallback(async (next) => {
    setMode(next);
    // 如果已有会话, 不做服务器端切换 (后端目前按请求体 mode 派发 Agent).
    // 当前 session.mode 会在下一轮 AskStream 里刷新.
  }, []);

  // --------------- Token 保存 ---------------
  const saveToken = useCallback(() => {
    setToken(token);
    pushToast("JWT 已保存", "success");
    loadModes(); loadSkills(); loadSessions();
  }, [token, pushToast, loadModes, loadSkills, loadSessions]);

  const connStatus = sse.running ? "run" : token ? "ok" : "err";

  return html`
    <div class="app-shell">
      <header class="app-header">
        <div class="brand">
          AI Solo
          <span class="sub">Mode 驱动 · Eino ADK</span>
        </div>
        <div class="actions">
          <span class=${`status-dot ${connStatus}`} title=${sse.running ? "运行中" : "空闲"}></span>
          <select
            class="btn sm"
            title="界面语言 (写入会话, 每轮对话随 meta 下发)"
            value=${uiLang}
            onChange=${(e) => setUiLang(e.target.value)}
          >
            <option value="zh">中文 UI</option>
            <option value="en">English UI</option>
          </select>
          <input
            type="password"
            placeholder="粘贴 JWT access token"
            value=${token}
            onInput=${(e) => setTokenState(e.target.value)}
          />
          <button class="btn primary sm" onClick=${saveToken}>保存</button>
        </div>
      </header>

      <main class="app-main">
        <${SessionList}
          sessions=${sessions}
          currentId=${currentId}
          onPick=${pickSession}
          onDelete=${deleteSession}
          onRefresh=${loadSessions}
          onNew=${newSession}
        />
        <${ChatView}
          session=${currentSession}
          messages=${messages}
          input=${input}
          setInput=${setInput}
          modes=${modes}
          mode=${mode}
          onModeChange=${onModeChange}
          skills=${skills}
          running=${sse.running}
          onSend=${send}
          onStop=${sse.stop}
          interrupt=${interrupt}
          onResume=${resume}
        />
      </main>

      <div class="toasts">
        ${toasts.items.map((t) => html`
          <div key=${t.id} class=${`toast ${t.kind}`} onClick=${() => toasts.remove(t.id)}>${t.msg}</div>
        `)}
      </div>
    </div>
  `;
}
