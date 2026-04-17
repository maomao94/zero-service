import { html, useState } from "../../lib/deps.js";

export function Approval({ data, onResume, disabled }) {
  const [reason, setReason] = useState("");
  return html`
    <div class="interrupt-panel">
      <span class="kind">approval</span>
      <div class="question">${data.question || "需要您确认以继续"}</div>
      ${data.detail && html`<div class="detail">${data.detail}</div>`}
      ${data.tool_name && html`<div class="tool-name">tool: ${data.tool_name}</div>`}
      <input
        type="text"
        placeholder="可选: 拒绝时的原因"
        value=${reason}
        onInput=${(e) => setReason(e.target.value)}
      />
      <div class="actions">
        <button class="btn primary" disabled=${disabled}
          onClick=${() => onResume({ action: "approve" })}>批准</button>
        <button class="btn danger" disabled=${disabled}
          onClick=${() => onResume({ action: "deny", reason })}>拒绝</button>
        <button class="btn ghost" disabled=${disabled}
          onClick=${() => onResume({ action: "cancel" })}>取消</button>
      </div>
    </div>
  `;
}
