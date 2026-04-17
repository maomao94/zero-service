import { html, useCallback, useMemo, useState } from "../lib/deps.js";
import { evalArith } from "../lib/calcEval.js";

const STORAGE_OPEN = "solo_calc_panel_open";

function readOpenPref() {
  try {
    return globalThis.sessionStorage?.getItem(STORAGE_OPEN) === "1";
  } catch {
    return false;
  }
}

function writeOpenPref(open) {
  try {
    globalThis.sessionStorage?.setItem(STORAGE_OPEN, open ? "1" : "0");
  } catch {
    /* 隐私模式等 */
  }
}

function formatValue(v) {
  if (!Number.isFinite(v)) return String(v);
  if (Number.isInteger(v)) return String(v);
  const s = v.toFixed(8).replace(/\.?0+$/, "");
  return s || "0";
}

export function CalcPanel({ disabled, setInput, onInserted }) {
  const [open, setOpen] = useState(readOpenPref);
  const [expr, setExpr] = useState("");

  const setOpenPersist = useCallback((next) => {
    setOpen((prev) => {
      const v = typeof next === "function" ? next(prev) : next;
      writeOpenPref(v);
      return v;
    });
  }, []);

  const preview = useMemo(() => {
    const t = expr.trim();
    if (!t) return { kind: "empty" };
    const r = evalArith(expr);
    if (!r.ok) return { kind: "err", msg: r.error };
    return { kind: "ok", value: r.value };
  }, [expr]);

  const insertLine = useCallback((text) => {
    setInput((prev) => (prev && prev.trim() ? `${prev.trim()}\n${text}` : text));
    onInserted?.();
  }, [setInput, onInserted]);

  const onInsertExpr = () => {
    if (preview.kind !== "ok") return;
    insertLine(expr.trim());
  };

  const onInsertWithHint = () => {
    if (preview.kind !== "ok") return;
    const e = expr.trim();
    const line = `请用 calculator 工具计算：${e}（本地预览结果约为 ${formatValue(preview.value)}）`;
    insertLine(line);
  };

  return html`
    <div class="calc-panel-wrap">
      <button type="button" class="btn sm ghost calc-toggle" onClick=${() => setOpenPersist((v) => !v)}>
        计算器 ${open ? "▲" : "▼"}
      </button>
      ${open && html`
        <div class="calc-panel" aria-label="本地计算器">
          <div class="calc-row">
            <label class="calc-label">表达式</label>
            <input
              type="text"
              class="calc-input"
              placeholder="例如 (15+25)/4*3"
              value=${expr}
              onInput=${(e) => setExpr(e.target.value)}
              spellcheck=${false}
            />
          </div>
          <div class="calc-preview ${preview.kind === "err" ? "is-err" : preview.kind === "ok" ? "is-ok" : ""}">
            ${preview.kind === "empty" && html`<span class="muted">输入后在此预览结果（与后端 calculator 规则一致）</span>`}
            ${preview.kind === "err" && html`<span>本地校验：${preview.msg}</span>`}
            ${preview.kind === "ok" && html`<span>= ${formatValue(preview.value)}</span>`}
          </div>
          <div class="calc-actions">
            <button type="button" class="btn sm" disabled=${disabled || preview.kind !== "ok"} onClick=${onInsertExpr}>
              插入表达式
            </button>
            <button type="button" class="btn sm" disabled=${disabled || preview.kind !== "ok"} onClick=${onInsertWithHint}>
              插入并提示模型
            </button>
          </div>
          ${disabled && html`
            <p class="calc-hint">插入到底部输入框：需已选会话、非流式中、且无待处理中断。</p>
          `}
        </div>
      `}
    </div>
  `;
}
