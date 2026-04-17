import { html, useState } from "../../lib/deps.js";

function seed(fields) {
  const out = {};
  (fields || []).forEach((f) => { out[f.name] = f.default || ""; });
  return out;
}

export function FormInput({ data, onResume, disabled }) {
  const [values, setValues] = useState(() => seed(data.fields));
  const fields = data.fields || [];
  const setField = (name, v) => setValues((s) => ({ ...s, [name]: v }));
  const missing = fields.filter((f) => f.required && !String(values[f.name] || "").trim());
  return html`
    <div class="interrupt-panel">
      <span class="kind">form_input</span>
      <div class="question">${data.question || "请填写表单"}</div>
      ${data.tool_name && html`<div class="tool-name">tool: ${data.tool_name}</div>`}
      <div>
        ${fields.map((f) => {
          const type = f.type || "string";
          const value = values[f.name] ?? "";
          let input;
          if (type === "boolean") {
            input = html`<input type="checkbox"
              checked=${value === "true" || value === true}
              onChange=${(e) => setField(f.name, e.target.checked ? "true" : "false")} />`;
          } else if (type === "number") {
            input = html`<input type="number"
              placeholder=${f.placeholder || ""}
              value=${value}
              onInput=${(e) => setField(f.name, e.target.value)} />`;
          } else {
            input = html`<input type="text"
              placeholder=${f.placeholder || ""}
              value=${value}
              onInput=${(e) => setField(f.name, e.target.value)} />`;
          }
          return html`
            <div class="field-row" key=${f.name}>
              <label>${f.label || f.name}${f.required ? " *" : ""}</label>
              ${input}
            </div>
          `;
        })}
      </div>
      <div class="actions">
        <button class="btn primary" disabled=${disabled || missing.length > 0}
          onClick=${() => onResume({ action: "form", formValues: values })}>提交</button>
        <button class="btn ghost" disabled=${disabled}
          onClick=${() => onResume({ action: "cancel" })}>取消</button>
      </div>
    </div>
  `;
}
