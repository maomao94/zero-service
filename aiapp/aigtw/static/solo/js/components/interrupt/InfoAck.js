import { html } from "../../lib/deps.js";
import { renderMarkdown } from "../../lib/markdown.js";
import { t } from "../../lib/i18n.js";

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
          onClick=${() => onResume({ action: "yes" })}>${t(data, "yes")}</button>
        <button class="btn ghost" disabled=${disabled}
          onClick=${() => onResume({ action: "no" })}>${t(data, "no")}</button>
      </div>
    </div>
  `;
}
