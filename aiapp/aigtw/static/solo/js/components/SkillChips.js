import { html } from "../lib/deps.js";

/** 技能快捷入口：仅把文案写入输入框；是否 launch 某技能由模型经服务端 skill 中间件（磁盘 SKILL.md）自行决定，不经 meta 传 skill_id。 */
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
