import { html, useMemo, useState } from "../lib/deps.js";

function badge(mode) {
  if (!mode) return null;
  return html`<span class=${`badge mode-${mode}`}>${mode}</span>`;
}

function statusBadge(status) {
  if (status === "interrupted" || status === "running") {
    return html`<span class=${`badge status-${status}`}>${status}</span>`;
  }
  return null;
}

export function SessionList({
  sessions,
  sessionsTotal = 0,
  hasMoreSessions = false,
  onLoadMoreSessions,
  currentId,
  onPick,
  onDelete,
  onRefresh,
  onNew,
}) {
  const [query, setQuery] = useState("");
  const q = query.trim().toLowerCase();
  const visible = useMemo(() => {
    if (!q) return sessions;
    return sessions.filter((s) => {
      const title = (s.title || "").toLowerCase();
      const id = (s.sessionId || "").toLowerCase();
      const last = (s.lastMessage || "").toLowerCase();
      return title.includes(q) || id.includes(q) || last.includes(q);
    });
  }, [sessions, q]);

  const totalHint = sessionsTotal > 0
    ? `已加载 ${sessions.length} / 共 ${sessionsTotal} 条`
    : "";
  return html`
    <aside class="sidebar">
      <div class="sidebar-header">
        <h3>会话列表</h3>
        <div>
          <button class="btn primary sm" onClick=${onNew}>+ 新建</button>
          <button class="btn ghost sm" onClick=${onRefresh}>刷新</button>
        </div>
      </div>
      <div class="sidebar-search">
        <input
          type="search"
          placeholder="搜索标题、ID、预览…"
          value=${query}
          onInput=${(e) => setQuery(e.target.value)}
        />
      </div>
      ${totalHint && html`<div class="sidebar-meta">${totalHint}</div>`}
      <ul class="session-list">
        ${sessions.length === 0 && html`<li style="color:var(--text-muted); cursor:default;">暂无会话</li>`}
        ${sessions.length > 0 && visible.length === 0 && html`
          <li style="color:var(--text-muted); cursor:default;">无匹配会话</li>
        `}
        ${visible.map((s) => html`
          <li
            key=${s.sessionId}
            class=${s.sessionId === currentId ? "active" : ""}
            onClick=${() => onPick(s.sessionId)}
          >
            <div class="title">${s.title || "未命名"}</div>
            <div class="meta">
              <span>${badge(s.mode)} ${statusBadge(s.status)}</span>
              <span style="display:flex; gap:6px; align-items:center;">
                <span>${s.messageCount || 0} 条</span>
                <button
                  class="btn ghost sm"
                  title="删除"
                  onClick=${(ev) => { ev.stopPropagation(); onDelete(s.sessionId); }}
                >✕</button>
              </span>
            </div>
            ${s.lastMessage && html`
              <div class="meta" style="color:var(--text-muted); white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">
                ${s.lastMessage}
              </div>
            `}
          </li>
        `)}
      </ul>
      ${hasMoreSessions && html`
        <div class="sidebar-footer">
          <button class="btn ghost sm" style="width:100%;" onClick=${onLoadMoreSessions}>
            加载更多
          </button>
        </div>
      `}
    </aside>
  `;
}
