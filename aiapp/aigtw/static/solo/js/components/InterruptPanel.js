import { html } from "../lib/deps.js";
import { Approval } from "./interrupt/Approval.js";
import { SingleSelect } from "./interrupt/SingleSelect.js";
import { MultiSelect } from "./interrupt/MultiSelect.js";
import { FreeText } from "./interrupt/FreeText.js";
import { FormInput } from "./interrupt/FormInput.js";
import { InfoAck } from "./interrupt/InfoAck.js";
import { t } from "../lib/i18n.js";

// InterruptPanel 分发 6 种 kind 到对应子面板.
export function InterruptPanel({ data, onResume, disabled }) {
  if (!data || !data.kind) return null;
  const agentChip = data.agent_name
    ? html`<div class="interrupt-agent-chip" title="触发该中断的 ADK Agent">${data.agent_name}</div>`
    : null;
  const wrap = (inner) => html`<div class="interrupt-wrap">${agentChip}${inner}</div>`;
  switch (data.kind) {
    case "approval":       return wrap(html`<${Approval}     data=${data} onResume=${onResume} disabled=${disabled} />`);
    case "single_select":  return wrap(html`<${SingleSelect} data=${data} onResume=${onResume} disabled=${disabled} />`);
    case "multi_select":   return wrap(html`<${MultiSelect}  data=${data} onResume=${onResume} disabled=${disabled} />`);
    case "free_text":      return wrap(html`<${FreeText}     data=${data} onResume=${onResume} disabled=${disabled} />`);
    case "form_input":     return wrap(html`<${FormInput}    data=${data} onResume=${onResume} disabled=${disabled} />`);
    case "info_ack":       return wrap(html`<${InfoAck}      data=${data} onResume=${onResume} disabled=${disabled} />`);
    default:
      return wrap(html`
        <div class="interrupt-panel">
          <span class="kind">${data.kind}</span>
          <div class="detail">${t(data, "unknownInterrupt")}: ${data.kind}</div>
          <div class="actions">
            <button class="btn ghost" disabled=${disabled}
              onClick=${() => onResume({ action: "no" })}>${t(data, "no")}</button>
          </div>
        </div>
      `);
  }
}
