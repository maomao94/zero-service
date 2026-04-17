import { html } from "../lib/deps.js";

/** 技能标签：点击将 launchPrompt 写入输入框（由父组件 setInput 拼接）。 */
export function SkillChips({ skills, setInput, disabled }) {
  if (!skills || skills.length === 0) return null;
  const onChip = (s) => {
    if (disabled) return;
    const lp = (s.launchPrompt || "").trim() || `请按技能「${s.name || s.id}」协助我。`;
    setInput((prev) => {
      const p = (prev || "").trim();
      return p ? `${lp}\n\n${p}` : lp;
    });
  };
  return html`
    <div class="skill-chips" aria-label="skills">
      <span class="skill-chips-label">Skills</span>
      ${skills.map(
        (s) => html`
          <button
            type="button"
            key=${s.id}
            class="skill-chip"
            disabled=${disabled}
            title=${s.description || s.name || s.id}
            onClick=${() => onChip(s)}
          >
            ${s.name || s.id}
          </button>
        `,
      )}
    </div>
  `;
}
