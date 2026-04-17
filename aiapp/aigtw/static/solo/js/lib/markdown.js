// 一层 marked + highlight.js 薄封装, 渲染 assistant / info_ack 的 markdown 文本.
// marked 和 highlight.js 在 index.html 里通过 <script> 全局加载.

function renderMarkdown(text) {
  if (!text) return "";
  if (typeof window === "undefined" || !window.marked) {
    return escapeHTML(text);
  }
  try {
    const marked = window.marked;
    if (marked.setOptions) {
      marked.setOptions({
        gfm: true,
        breaks: true,
        highlight(code, lang) {
          if (window.hljs) {
            try {
              if (lang && window.hljs.getLanguage(lang)) {
                return window.hljs.highlight(code, { language: lang }).value;
              }
              return window.hljs.highlightAuto(code).value;
            } catch (_) { /* ignore */ }
          }
          return code;
        },
      });
    }
    return typeof marked.parse === "function" ? marked.parse(text) : marked(text);
  } catch (err) {
    console.warn("[markdown] fallback", err);
    return escapeHTML(text);
  }
}

function escapeHTML(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

export { renderMarkdown, escapeHTML };
