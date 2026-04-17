import { html } from "../../lib/deps.js";
import { renderMarkdown } from "../../lib/markdown.js";

export function InfoAck({ data, onResume, disabled }) {
  return html`
    <div class="interrupt-panel">
      <span class="kind">info_ack</span>
      ${data.title && html`<div class="question">${data.title}</div>`}
      <div
        class="body"
        dangerouslySetInnerHTML=${{ __html: renderMarkdown(data.body || data.detail || "") }}
      ></div>
      ${data.tool_name && html`<div class="tool-name">tool: ${data.tool_name}</div>`}
      <div class="actions">
        <button class="btn primary" disabled=${disabled}
          onClick=${() => onResume({ action: "ack" })}>已确认</button>
        <button class="btn ghost" disabled=${disabled}
          onClick=${() => onResume({ action: "cancel" })}>取消</button>
      </div>
    </div>
  `;
}
