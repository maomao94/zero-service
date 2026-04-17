const DICT = {
  zh: {
    yes: "确认",
    no: "取消",
    submit: "提交",
    denyReasonPlaceholder: "可选: 取消原因",
    approvalDefaultQuestion: "需要您确认以继续",
    freeTextDefaultQuestion: "请输入文本",
    singleSelectDefaultQuestion: "请选择一项",
    multiSelectDefaultQuestion: "请选择 (可多选)",
    multiSelectMin: "至少 {n} 项",
    multiSelectMax: "最多 {n} 项",
    multiSelectSep: "，",
    formDefaultQuestion: "请填写表单",
    formOtherHint: "其他（自定义）",
    formOtherPlaceholder: "填写未列出的选项…",
    selectPlaceholder: "请选择…",
    unknownInterrupt: "未知的中断类型",
  },
  en: {
    yes: "Yes",
    no: "No",
    submit: "Submit",
    denyReasonPlaceholder: "Optional: reason for no",
    approvalDefaultQuestion: "Please confirm to continue",
    freeTextDefaultQuestion: "Please enter text",
    singleSelectDefaultQuestion: "Please select one option",
    multiSelectDefaultQuestion: "Please select options",
    multiSelectMin: "At least {n}",
    multiSelectMax: "At most {n}",
    multiSelectSep: " · ",
    formDefaultQuestion: "Please fill out the form",
    formOtherHint: "Other (custom)",
    formOtherPlaceholder: "Type a value not listed…",
    selectPlaceholder: "Choose…",
    unknownInterrupt: "Unknown interrupt type",
  },
};

function normalize(lang) {
  const v = String(lang || "").trim().toLowerCase();
  if (v.startsWith("en")) return "en";
  if (v.startsWith("zh")) return "zh";
  return "zh";
}

function applyVars(s, vars) {
  if (!vars || typeof s !== "string") return s;
  let out = s;
  Object.keys(vars).forEach((k) => {
    out = out.split(`{${k}}`).join(String(vars[k]));
  });
  return out;
}

/** @param {any} data interrupt payload (expects snake_case ui_lang from protocol) */
export function t(data, key, vars) {
  const lang = normalize(data && data.ui_lang);
  const table = DICT[lang] || DICT.zh;
  const raw = table[key] || DICT.zh[key] || key;
  return applyVars(raw, vars);
}
