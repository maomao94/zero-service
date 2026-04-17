import { html } from "../lib/deps.js";

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

export function SessionList({ sessions, currentId, onPick, onDelete, onRefresh, onNew }) {
  return html`
    <aside class="sidebar">
      <div class="sidebar-header">
        <h3>会话列表</h3>
        <div>
          <button class="btn primary sm" onClick=${onNew}>+ 新建</button>
          <button class="btn ghost sm" onClick=${onRefresh}>刷新</button>
        </div>
      </div>
      <ul class="session-list">
        ${sessions.length === 0 && html`<li style="color:var(--text-muted); cursor:default;">暂无会话</li>`}
        ${sessions.map((s) => html`
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
    </aside>
  `;
}
