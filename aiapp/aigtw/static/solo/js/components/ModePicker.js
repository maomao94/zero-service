import { html } from "../lib/deps.js";

// ModePicker: 用 Chip 选择 Mode, 不再暴露具体 Agent.
export function ModePicker({ modes, value, onChange, disabled }) {
  if (!modes || modes.length === 0) {
    return html`<span style="color:var(--text-muted); font-size:12px;">无可用 Mode</span>`;
  }
  return html`
    <div class="mode-picker">
      ${modes.map((m) => html`
        <button
          type="button"
          key=${m.mode}
          class=${"chip" + (m.mode === value ? " active" : "")}
          disabled=${!!disabled}
          title=${m.description}
          onClick=${() => onChange(m.mode)}
        >${m.name || m.mode}${m.default ? " ·默认" : ""}</button>
      `)}
    </div>
  `;
}
