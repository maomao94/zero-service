import { html, useState } from "../../lib/deps.js";
import { t } from "../../lib/i18n.js";

export function SingleSelect({ data, onResume, disabled }) {
  const [picked, setPicked] = useState((data.options && data.options[0] && data.options[0].id) || "");
  return html`
    <div class="interrupt-panel">
      <span class="kind">single_select</span>
      <div class="question">${data.question || t(data, "singleSelectDefaultQuestion")}</div>
      ${data.tool_name && html`<div class="tool-name">tool: ${data.tool_name}</div>`}
      <div class="options">
        ${(data.options || []).map((opt) => html`
          <label key=${opt.id}>
            <input
              type="radio"
              name="ss"
              value=${opt.id}
              checked=${picked === opt.id}
              onChange=${() => setPicked(opt.id)}
            />
            <span>
              <div class="option-label">${opt.label || opt.id}</div>
              ${opt.desc && html`<div class="option-desc">${opt.desc}</div>`}
            </span>
          </label>
        `)}
      </div>
      <div class="actions">
        <button class="btn primary" disabled=${disabled || !picked}
          onClick=${() => onResume({ action: "yes", selectedIds: [picked] })}>${t(data, "yes")}</button>
        <button class="btn ghost" disabled=${disabled}
          onClick=${() => onResume({ action: "no" })}>${t(data, "no")}</button>
      </div>
    </div>
  `;
}
