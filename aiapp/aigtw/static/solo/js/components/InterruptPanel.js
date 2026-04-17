import { html } from "../lib/deps.js";
import { Approval } from "./interrupt/Approval.js";
import { SingleSelect } from "./interrupt/SingleSelect.js";
import { MultiSelect } from "./interrupt/MultiSelect.js";
import { FreeText } from "./interrupt/FreeText.js";
import { FormInput } from "./interrupt/FormInput.js";
import { InfoAck } from "./interrupt/InfoAck.js";

// InterruptPanel 分发 6 种 kind 到对应子面板.
export function InterruptPanel({ data, onResume, disabled }) {
  if (!data || !data.kind) return null;
  switch (data.kind) {
    case "approval":       return html`<${Approval}     data=${data} onResume=${onResume} disabled=${disabled} />`;
    case "single_select":  return html`<${SingleSelect} data=${data} onResume=${onResume} disabled=${disabled} />`;
    case "multi_select":   return html`<${MultiSelect}  data=${data} onResume=${onResume} disabled=${disabled} />`;
    case "free_text":      return html`<${FreeText}     data=${data} onResume=${onResume} disabled=${disabled} />`;
    case "form_input":     return html`<${FormInput}    data=${data} onResume=${onResume} disabled=${disabled} />`;
    case "info_ack":       return html`<${InfoAck}      data=${data} onResume=${onResume} disabled=${disabled} />`;
    default:
      return html`
        <div class="interrupt-panel">
          <span class="kind">${data.kind}</span>
          <div class="detail">未知的中断类型: ${data.kind}</div>
          <div class="actions">
            <button class="btn ghost" disabled=${disabled}
              onClick=${() => onResume({ action: "cancel" })}>取消</button>
          </div>
        </div>
      `;
  }
}
