import { html, useState } from "../../lib/deps.js";
import { t } from "../../lib/i18n.js";

function optionId(opt) {
  if (opt == null) return "";
  const a = opt.id != null && String(opt.id).trim() !== "" ? String(opt.id) : "";
  if (a) return a;
  const b = opt.value != null && String(opt.value).trim() !== "" ? String(opt.value) : "";
  return b;
}

export function MultiSelect({ data, onResume, disabled }) {
  const [picked, setPicked] = useState(() => new Set());
  const min = data.min_select ?? data.minSelect ?? 0;
  const max = data.max_select ?? data.maxSelect ?? 0;
  const toggle = (id) => {
    if (disabled) return;
    setPicked((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
        return next;
      }
      if (max > 0 && next.size >= max) return prev;
      next.add(id);
      return next;
    });
  };
  const arr = [...picked];
  const ok = arr.length >= min && (max === 0 || arr.length <= max);
  const boundsParts = [];
  if (min > 0) boundsParts.push(t(data, "multiSelectMin", { n: min }));
  if (max > 0) boundsParts.push(t(data, "multiSelectMax", { n: max }));
  const bounds = boundsParts.join(t(data, "multiSelectSep"));
  const options = (data.options || []).filter((opt) => optionId(opt));
  return html`
    <div class="interrupt-panel">
      <span class="kind">multi_select</span>
      <div class="question">${data.question || t(data, "multiSelectDefaultQuestion")}</div>
      ${(min > 0 || max > 0) && html`<div class="detail">${bounds}</div>`}
      ${data.tool_name && html`<div class="tool-name">tool: ${data.tool_name}</div>`}
      <div class="options">
        ${options.map((opt) => {
          const oid = optionId(opt);
          const lab = opt.label || oid;
          const on = picked.has(oid);
          return html`
            <label key=${oid}>
              <input
                type="checkbox"
                checked=${on}
                disabled=${disabled}
                onChange=${() => toggle(oid)}
              />
              <span>
                <div class="option-label">${lab}</div>
                ${opt.desc && html`<div class="option-desc">${opt.desc}</div>`}
              </span>
            </label>
          `;
        })}
      </div>
      <div class="actions">
        <button class="btn primary" disabled=${disabled || !ok}
          onClick=${() => onResume({ action: "yes", selectedIds: arr })}>${t(data, "yes")}</button>
        <button class="btn ghost" disabled=${disabled}
          onClick=${() => onResume({ action: "no" })}>${t(data, "no")}</button>
      </div>
    </div>
  `;
}
