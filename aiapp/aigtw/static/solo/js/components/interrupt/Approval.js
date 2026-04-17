import { html, useState } from "../../lib/deps.js";
import { t } from "../../lib/i18n.js";

export function Approval({ data, onResume, disabled }) {
  const [reason, setReason] = useState("");
  return html`
    <div class="interrupt-panel">
      <span class="kind">approval</span>
      <div class="question">${data.question || t(data, "approvalDefaultQuestion")}</div>
      ${data.detail && html`<div class="detail">${data.detail}</div>`}
      ${data.tool_name && html`<div class="tool-name">tool: ${data.tool_name}</div>`}
      <input
        type="text"
        placeholder=${t(data, "denyReasonPlaceholder")}
        value=${reason}
        onInput=${(e) => setReason(e.target.value)}
      />
      <div class="actions">
        <button class="btn primary" disabled=${disabled}
          onClick=${() => onResume({ action: "yes" })}>${t(data, "yes")}</button>
        <button class="btn danger" disabled=${disabled}
          onClick=${() => onResume({ action: "no", reason })}>${t(data, "no")}</button>
      </div>
    </div>
  `;
}
