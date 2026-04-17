import { html, useState } from "../../lib/deps.js";

export function MultiSelect({ data, onResume, disabled }) {
  const [picked, setPicked] = useState(new Set());
  const min = data.min_select || 0;
  const max = data.max_select || 0;
  const toggle = (id) => {
    const next = new Set(picked);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setPicked(next);
  };
  const arr = [...picked];
  const ok = arr.length >= min && (max === 0 || arr.length <= max);
  return html`
    <div class="interrupt-panel">
      <span class="kind">multi_select</span>
      <div class="question">${data.question || "请选择 (可多选)"}</div>
      ${(min > 0 || max > 0) && html`
        <div class="detail">
          ${min > 0 ? `至少 ${min} 项` : ""}${min > 0 && max > 0 ? "，" : ""}${max > 0 ? `最多 ${max} 项` : ""}
        </div>`}
      ${data.tool_name && html`<div class="tool-name">tool: ${data.tool_name}</div>`}
      <div class="options">
        ${(data.options || []).map((opt) => html`
          <label key=${opt.id}>
            <input
              type="checkbox"
              checked=${picked.has(opt.id)}
              onChange=${() => toggle(opt.id)}
            />
            <span>
              <div class="option-label">${opt.label || opt.id}</div>
              ${opt.desc && html`<div class="option-desc">${opt.desc}</div>`}
            </span>
          </label>
        `)}
      </div>
      <div class="actions">
        <button class="btn primary" disabled=${disabled || !ok}
          onClick=${() => onResume({ action: "select", selectedIds: arr })}>提交</button>
        <button class="btn ghost" disabled=${disabled}
          onClick=${() => onResume({ action: "cancel" })}>取消</button>
      </div>
    </div>
  `;
}
