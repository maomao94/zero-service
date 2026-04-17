import { html, useState } from "../../lib/deps.js";

export function FreeText({ data, onResume, disabled }) {
  const [text, setText] = useState("");
  const multiline = !!data.multiline;
  return html`
    <div class="interrupt-panel">
      <span class="kind">free_text</span>
      <div class="question">${data.question || "请输入文本"}</div>
      ${data.tool_name && html`<div class="tool-name">tool: ${data.tool_name}</div>`}
      ${multiline
        ? html`<textarea
            placeholder=${data.placeholder || ""}
            value=${text}
            onInput=${(e) => setText(e.target.value)}
          ></textarea>`
        : html`<input
            type="text"
            placeholder=${data.placeholder || ""}
            value=${text}
            onInput=${(e) => setText(e.target.value)}
          />`}
      <div class="actions">
        <button class="btn primary" disabled=${disabled || (data.required && !text.trim())}
          onClick=${() => onResume({ action: "text", text })}>提交</button>
        <button class="btn ghost" disabled=${disabled}
          onClick=${() => onResume({ action: "cancel" })}>取消</button>
      </div>
    </div>
  `;
}
