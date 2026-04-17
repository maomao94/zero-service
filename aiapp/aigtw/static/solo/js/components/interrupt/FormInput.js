import { html, useState } from "../../lib/deps.js";
import { t } from "../../lib/i18n.js";

/** 与后端 builtin.normalizeFormWidget 对齐, 避免模型写 multiselect 时落到默认单行文本框。 */
function normalizeWidget(raw) {
  if (!raw || typeof raw !== "string") return "";
  const s = raw.trim().toLowerCase().replace(/-/g, "_").replace(/\s+/g, "_");
  if (s === "multiselect" || s === "multi_select") return "multi_select";
  return raw.trim();
}

function optionId(opt) {
  if (opt == null) return "";
  const a = opt.id != null && String(opt.id).trim() !== "" ? String(opt.id) : "";
  if (a) return a;
  const b = opt.value != null && String(opt.value).trim() !== "" ? String(opt.value) : "";
  return b;
}

/** SSE 用 snake_case，GetInterrupt 用 camelCase */
function allowCustomFlag(f) {
  return !!(f && (f.allow_custom === true || f.allowCustom === true));
}

function seed(fields) {
  const out = {};
  (fields || []).forEach((f) => {
    const w = normalizeWidget(f.widget || "");
    if (w === "multi_select") {
      out[f.name] = f.default || "[]";
      return;
    }
    out[f.name] = f.default || "";
  });
  return out;
}

function asArray(v) {
  if (Array.isArray(v)) return v.map((x) => String(x));
  if (typeof v !== "string") return [];
  try {
    const arr = JSON.parse(v);
    return Array.isArray(arr) ? arr.map((x) => String(x)) : [];
  } catch (_) {
    return [];
  }
}

function mergeSubmitValues(values, customExtra, fields) {
  const out = { ...values };
  (fields || []).forEach((f) => {
    const w = normalizeWidget(f.widget || "");
    const ac = allowCustomFlag(f);
    const extra = String((customExtra && customExtra[f.name]) || "").trim();
    if (w === "multi_select") {
      let arr = asArray(out[f.name]);
      if (ac && extra && !arr.includes(extra)) {
        arr = [...arr, extra];
      }
      out[f.name] = JSON.stringify(arr);
      return;
    }
    if ((w === "select" || w === "radio") && ac && extra) {
      out[f.name] = extra;
    }
  });
  return out;
}

function fieldFilled(f, values, customExtra) {
  if (!f.required) return true;
  const w = normalizeWidget(f.widget || "");
  const custom = String((customExtra && customExtra[f.name]) || "").trim();
  if (w === "multi_select") {
    const n = asArray(values[f.name]).length;
    const fMin = f.min_select ?? f.minSelect ?? 0;
    const fMax = f.max_select ?? f.maxSelect ?? 0;
    if (fMin > 0 && n < fMin) return false;
    if (fMax > 0 && n > fMax) return false;
    if (n > 0) return true;
    return allowCustomFlag(f) && custom.length > 0;
  }
  if (w === "select" || w === "radio") {
    if (String(values[f.name] || "").trim()) return true;
    return allowCustomFlag(f) && custom.length > 0;
  }
  if (w === "switch" || (f.type || "") === "boolean") {
    return values[f.name] === "true" || values[f.name] === true;
  }
  return String(values[f.name] || "").trim().length > 0;
}

export function FormInput({ data, onResume, disabled }) {
  const [values, setValues] = useState(() => seed(data.fields));
  const [customExtra, setCustomExtra] = useState({});
  const fields = data.fields || [];
  const setField = (name, v) => setValues((s) => ({ ...s, [name]: v }));
  const setCustom = (name, v) => setCustomExtra((s) => ({ ...s, [name]: v }));

  const missing = fields.filter((f) => !fieldFilled(f, values, customExtra));

  const renderSingleChoices = (f, options, value, useRadioInput) => {
    const withIds = (options || []).filter((opt) => optionId(opt));
    const ac = allowCustomFlag(f);
    const customVal = customExtra[f.name] || "";
    return html`
      <div class="form-choice-stack">
        ${withIds.map((opt) => {
          const oid = optionId(opt);
          const lab = opt.label || oid;
          const active = !String(customVal).trim() && String(value) === oid;
          if (useRadioInput) {
            return html`
              <label key=${oid} class=${`form-choice-item form-choice-item--radio ${active ? "is-active" : ""}`}>
                <input
                  type="radio"
                  name=${`form-radio-${f.name}`}
                  checked=${active}
                  onChange=${() => {
                    setField(f.name, oid);
                    setCustom(f.name, "");
                  }}
                />
                <span class="form-choice-ui" aria-hidden="true"></span>
                <span class="form-choice-text">
                  <span class="option-label">${lab}</span>
                  ${opt.desc && html`<span class="option-desc">${opt.desc}</span>`}
                </span>
              </label>
            `;
          }
          return html`
            <button
              type="button"
              key=${oid}
              class=${`form-choice-item form-choice-item--btn ${active ? "is-active" : ""}`}
              onClick=${() => {
                setField(f.name, oid);
                setCustom(f.name, "");
              }}
            >
              <span class="form-choice-ui form-choice-ui--dot" aria-hidden="true"></span>
              <span class="form-choice-text">
                <span class="option-label">${lab}</span>
                ${opt.desc && html`<span class="option-desc">${opt.desc}</span>`}
              </span>
            </button>
          `;
        })}
        ${ac &&
        html`
          <div class="form-custom-block">
            <div class="form-custom-label">${t(data, "formOtherHint")}</div>
            <input
              type="text"
              class="form-custom-input"
              placeholder=${f.placeholder || t(data, "formOtherPlaceholder")}
              value=${customVal}
              onInput=${(e) => {
                const v = e.target.value;
                setCustom(f.name, v);
                if (String(v).trim()) setField(f.name, "");
              }}
            />
          </div>
        `}
      </div>
    `;
  };

  return html`
    <div class="interrupt-panel">
      <span class="kind">form_input</span>
      <div class="question">${data.question || t(data, "formDefaultQuestion")}</div>
      ${data.tool_name && html`<div class="tool-name">tool: ${data.tool_name}</div>`}
      <div class="form-fields-block">
        ${fields.map((f) => {
          const type = f.type || "string";
          const widget = normalizeWidget(f.widget || "");
          const options = f.options || [];
          const value = values[f.name] ?? "";
          let input;

          if (widget === "textarea") {
            input = html`<textarea
              placeholder=${f.placeholder || ""}
              value=${value}
              onInput=${(e) => setField(f.name, e.target.value)}
            ></textarea>`;
          } else if (widget === "select") {
            const ac = allowCustomFlag(f);
            if (ac) {
              input = renderSingleChoices(f, options, value, false);
            } else {
              input = html`<select
                value=${value}
                onChange=${(e) => setField(f.name, e.target.value)}
              >
                <option value="">${t(data, "selectPlaceholder")}</option>
                ${options.map((opt) => html`<option key=${opt.id} value=${opt.id}>${opt.label || opt.id}</option>`)}
              </select>`;
            }
          } else if (widget === "radio") {
            input = renderSingleChoices(f, options, value, true);
          } else if (widget === "multi_select") {
            const selected = asArray(value);
            const fMin = f.min_select ?? f.minSelect ?? 0;
            const fMax = f.max_select ?? f.maxSelect ?? 0;
            const boundsParts = [];
            if (fMin > 0) boundsParts.push(t(data, "multiSelectMin", { n: fMin }));
            if (fMax > 0) boundsParts.push(t(data, "multiSelectMax", { n: fMax }));
            const bounds = boundsParts.join(t(data, "multiSelectSep"));
            const withIds = (options || []).filter((opt) => optionId(opt));
            const ac = allowCustomFlag(f);
            const customVal = customExtra[f.name] || "";
            input = html`
              <div class="form-check-stack">
                ${(fMin > 0 || fMax > 0) && html`<div class="detail" style="margin-bottom:4px;">${bounds}</div>`}
                ${withIds.map((opt) => {
                  const oid = optionId(opt);
                  const lab = opt.label || oid;
                  const on = selected.includes(oid);
                  return html`
                    <label key=${oid} class=${`form-check-item ${on ? "is-on" : ""}`}>
                      <input
                        type="checkbox"
                        checked=${on}
                        disabled=${disabled}
                        onChange=${() => {
                          setValues((s) => {
                            const cur = asArray(s[f.name]);
                            let next;
                            if (on) {
                              next = cur.filter((x) => x !== oid);
                            } else {
                              if (fMax > 0 && cur.length >= fMax) return s;
                              next = cur.includes(oid) ? cur : [...cur, oid];
                            }
                            return { ...s, [f.name]: JSON.stringify(next) };
                          });
                        }}
                      />
                      <span class="form-check-ui" aria-hidden="true"></span>
                      <span class="form-check-caption">${lab}</span>
                    </label>
                  `;
                })}
                ${ac &&
                html`
                  <div class="form-custom-block form-custom-block--tight">
                    <div class="form-custom-label">${t(data, "formOtherHint")}</div>
                    <input
                      type="text"
                      class="form-custom-input"
                      placeholder=${f.placeholder || t(data, "formOtherPlaceholder")}
                      value=${customVal}
                      onInput=${(e) => setCustom(f.name, e.target.value)}
                    />
                  </div>
                `}
              </div>
            `;
          } else if (widget === "switch" || type === "boolean") {
            input = html`<input type="checkbox"
              checked=${value === "true" || value === true}
              onChange=${(e) => setField(f.name, e.target.checked ? "true" : "false")} />`;
          } else if (widget === "number" || type === "number") {
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
              <div class="field-control">${input}</div>
            </div>
          `;
        })}
      </div>
      <div class="actions">
        <button class="btn primary" disabled=${disabled || missing.length > 0}
          onClick=${() => onResume({ action: "yes", formValues: mergeSubmitValues(values, customExtra, fields) })}>${t(data, "yes")}</button>
        <button class="btn ghost" disabled=${disabled}
          onClick=${() => onResume({ action: "no" })}>${t(data, "no")}</button>
      </div>
    </div>
  `;
}
