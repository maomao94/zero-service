import { html, useCallback, useEffect, useRef, useState } from "../lib/deps.js";
import { renderMarkdown } from "../lib/markdown.js";
import { ModePicker } from "./ModePicker.js";
import { InterruptPanel } from "./InterruptPanel.js";
import { CalcPanel } from "./CalcPanel.js";
import { RagPanel } from "./RagPanel.js";
import { SkillChips } from "./SkillChips.js";

/** 工具参数/结果：尽量格式化为多行 JSON，否则原文 */
function formatToolPayload(raw) {
  if (raw == null) return "";
  const s = String(raw);
  if (!s.trim()) return "";
  try {
    const o = JSON.parse(s);
    return JSON.stringify(o, null, 2);
  } catch (_) {
    return s;
  }
}

function agentChip(name) {
  if (!name) return null;
  return html`<span class="msg-agent-chip" title="ADK Agent">${name}</span>`;
}

/** Unix 秒或毫秒 → 简短本地时间 */
function formatMsgClock(ts) {
  if (ts == null || ts === 0) return "";
  const n = Number(ts);
  if (!Number.isFinite(n)) return "";
  const ms = n < 1e12 ? n * 1000 : n;
  const d = new Date(ms);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString(undefined, {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function timeChip(ts) {
  const t = formatMsgClock(ts);
  if (!t) return null;
  return html`<span class="msg-time" title="消息时间">${t}</span>`;
}

function ToolCallBlock({ m }) {
  const isKnowledgeTool = m.tool === "search_knowledge_base";
  const hasResult = m.result != null && String(m.result).length > 0;
  const cls = [
    "tool-call",
    m.error ? "error" : hasResult ? "" : "pending",
    isKnowledgeTool ? "knowledge" : "",
  ]
    .filter(Boolean)
    .join(" ");
  let argsDisplay = m.args ? formatToolPayload(m.args) : "";
  let resultDisplay = hasResult ? formatToolPayload(m.result) : "";
  if (m.tool === "echo" && m.args) {
    try {
      const o = JSON.parse(m.args);
      if (o && typeof o.text === "string") argsDisplay = o.text;
    } catch (_) { /* keep formatted */ }
  }
  if (m.tool === "echo" && hasResult) {
    resultDisplay = String(m.result);
  }
  const done = hasResult || !!m.error;
  const [expanded, setExpanded] = useState(() => !done);
  useEffect(() => {
    if (isKnowledgeTool && done) setExpanded(false);
  }, [isKnowledgeTool, done]);
  return html`
    <div class=${cls}>
      <button
        type="button"
        class="tool-call-toggle"
        onClick=${() => setExpanded((v) => !v)}
        aria-expanded=${expanded}
      >
        <span class="chevron" aria-hidden="true">${expanded ? "▼" : "▶"}</span>
        <span class="tool-call-badge">${isKnowledgeTool ? "KB" : "tool"}</span>
        ${agentChip(m.agent_name)}
        <span class="name">${m.tool || "tool"}</span>
        ${done
          ? html`<span class="tool-call-status">完成</span>`
          : html`<span class="tool-call-status pending">进行中</span>`}
      </button>
      ${expanded && m.args &&
      html`
        <div class="tool-section-label">参数</div>
        <pre class="tool-payload">${m.tool === "echo" ? argsDisplay : argsDisplay}</pre>
      `}
      ${expanded && hasResult &&
      html`
        <div class="tool-section-label">结果</div>
        <pre class="tool-payload">${resultDisplay}</pre>
      `}
      ${m.error && html`<div class="tool-soft-error">错误: ${m.error}</div>`}
    </div>
  `;
}

function MessageItem({ m }) {
  const role = m.role || "assistant";
  if (role === "tool_call") {
    return html`<${ToolCallBlock} m=${m} />`;
  }
  if (role === "assistant") {
    return html`
      <div class="message assistant">
        <div class="role">${timeChip(m.createdAt)}${agentChip(m.agent_name)}<span class="role-label">assistant</span></div>
        <div class="body" dangerouslySetInnerHTML=${{ __html: renderMarkdown(m.content || "") }}></div>
      </div>`;
  }
  if (role === "user") {
    return html`
      <div class="message user">
        <div class="role">${timeChip(m.createdAt)}<span class="role-label">user</span></div>
        <div class="body">${m.content}</div>
      </div>`;
  }
  if (role === "system") {
    return html`
      <div class="message system">
        <div class="role">${timeChip(m.createdAt)}<span class="role-label">system</span></div>
        <div class="body">${m.content}</div>
      </div>`;
  }
  return html`
    <div class="message tool">
      <div class="role">${timeChip(m.createdAt)}<span class="role-label">${role}</span></div>
      <div class="body">${m.content}</div>
    </div>
  `;
}

export function ChatView({
  session, messages, input, setInput,
  modes, mode, onModeChange,
  skills,
  running, onSend, onStop,
  interrupt, onResume,
  onRefreshSession,
}) {
  const scrollRef = useRef(null);
  const composerRef = useRef(null);

  /** 插入文本后把光标放到底部输入框（等 Preact 提交 DOM 后再 focus）。 */
  const focusComposer = useCallback(() => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        const el = composerRef.current;
        if (!el || el.disabled) return;
        el.focus();
        const len = el.value.length;
        try {
          el.setSelectionRange(len, len);
        } catch (_) { /* ignore */ }
      });
    });
  }, []);

  useEffect(() => {
    if (scrollRef.current) scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
  }, [messages, interrupt, running]);

  const onKey = (e) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      if (!running && input.trim()) onSend();
    }
  };

  const calcDisabled = running || !session || !!interrupt;

  return html`
    <section class="chat-area">
      <div class="chat-toolbar">
        <div class="left">
          <span class="title">${session ? (session.title || "未命名会话") : "未选择会话"}</span>
          <span class="sub">
            ${session
              ? html`
                  ${session.sessionId.slice(0, 8)} · ${session.status || "idle"} · ${session.messageCount || 0} 条
                  ${session.knowledgeBaseId
                    ? html`<span class="kb-pill" title=${session.knowledgeBaseName || session.knowledgeBaseId}>知识库</span>`
                    : null}
                `
              : "在左侧选择或新建一个会话开始对话"}
          </span>
        </div>
        <div class="right">
          <span style="font-size:12px; color:var(--text-muted);">Mode</span>
          <${ModePicker}
            modes=${modes}
            value=${mode}
            onChange=${onModeChange}
            disabled=${running || !session}
          />
        </div>
      </div>

      <${SkillChips}
        skills=${skills || []}
        setInput=${setInput}
        disabled=${calcDisabled}
      />
      <${CalcPanel} disabled=${calcDisabled} setInput=${setInput} onInserted=${focusComposer} />
      <${RagPanel}
        disabled=${calcDisabled}
        setInput=${setInput}
        onInserted=${focusComposer}
        sessionId=${session?.sessionId || ""}
        boundKnowledge=${session && (session.knowledgeBaseId || session.knowledgeBaseName)
          ? {
              knowledgeBaseId: session.knowledgeBaseId || "",
              knowledgeBaseName: session.knowledgeBaseName || "",
            }
          : null}
        onKnowledgeBound=${onRefreshSession}
      />

      <div class="messages" ref=${scrollRef}>
        ${!session && html`
          <div class="empty">
            <h2>AI Solo</h2>
            <p>Mode 驱动的多智能体对话 · 支持 6 种人机中断</p>
          </div>
        `}
        ${session && messages.length === 0 && !interrupt && html`
          <div class="empty">
            <p>输入一条消息开始对话</p>
          </div>
        `}
        ${messages.map((m, i) => html`<${MessageItem} key=${m.id || i} m=${m} />`)}
        ${interrupt && html`<${InterruptPanel}
          key=${interrupt.interrupt_id || interrupt.kind || "interrupt"}
          data=${interrupt}
          onResume=${onResume}
          disabled=${running}
        />`}
      </div>

      <footer class="composer">
        <textarea
          ref=${composerRef}
          rows="2"
          placeholder=${interrupt ? "请先处理上方中断..." : "输入消息, Enter 发送, Shift+Enter 换行"}
          value=${input}
          disabled=${running || !session || !!interrupt}
          onInput=${(e) => setInput(e.target.value)}
          onKeyDown=${onKey}
        ></textarea>
        <div class="send-group">
          ${running
            ? html`<button class="btn danger" onClick=${onStop}>停止</button>`
            : html`<button class="btn primary" disabled=${!session || !input.trim() || !!interrupt} onClick=${onSend}>发送</button>`}
        </div>
      </footer>
    </section>
  `;
}
