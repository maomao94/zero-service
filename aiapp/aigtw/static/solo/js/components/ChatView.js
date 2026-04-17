import { html, useCallback, useEffect, useRef } from "../lib/deps.js";
import { renderMarkdown } from "../lib/markdown.js";
import { ModePicker } from "./ModePicker.js";
import { InterruptPanel } from "./InterruptPanel.js";
import { CalcPanel } from "./CalcPanel.js";

function MessageItem({ m }) {
  const role = m.role || "assistant";
  if (role === "tool_call") {
    const hasResult = m.result != null && String(m.result).length > 0;
    const cls = m.error ? "tool-call error" : hasResult ? "tool-call" : "tool-call pending";
    return html`
      <div class=${cls}>
        <div class="name">→ ${m.tool || "tool"}</div>
        ${m.args && html`<div class="args">args: ${m.args}</div>`}
        ${hasResult && html`<div class="result">result: ${m.result}</div>`}
        ${m.error && html`<div class="result tool-soft-error">error: ${m.error}</div>`}
      </div>
    `;
  }
  if (role === "assistant") {
    return html`
      <div class="message assistant">
        <div class="role">assistant</div>
        <div class="body" dangerouslySetInnerHTML=${{ __html: renderMarkdown(m.content || "") }}></div>
      </div>`;
  }
  if (role === "user") {
    return html`
      <div class="message user">
        <div class="role">user</div>
        <div class="body">${m.content}</div>
      </div>`;
  }
  if (role === "system") {
    return html`<div class="message system">${m.content}</div>`;
  }
  return html`
    <div class="message tool">
      <div class="role">${role}</div>
      <div class="body">${m.content}</div>
    </div>
  `;
}

export function ChatView({
  session, messages, input, setInput,
  modes, mode, onModeChange,
  running, onSend, onStop,
  interrupt, onResume,
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
              ? `${session.sessionId.slice(0, 8)} · ${session.status || "idle"} · ${session.messageCount || 0} 条`
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

      <${CalcPanel} disabled=${calcDisabled} setInput=${setInput} onInserted=${focusComposer} />

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
        ${interrupt && html`<${InterruptPanel} data=${interrupt} onResume=${onResume} disabled=${running} />`}
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
