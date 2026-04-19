import { html, useCallback, useEffect, useState } from "../lib/deps.js";
import { api } from "../api/client.js";
import { readPreferredKnowledgeBaseId, writePreferredKnowledgeBaseId } from "../lib/soloPrefs.js";

const STORAGE_OPEN = "solo_rag_panel_open";

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
    /* ignore */
  }
}

export function RagPanel({
  disabled,
  setInput,
  onInserted,
  sessionId = "",
  boundKnowledge = null,
  onKnowledgeBound,
}) {
  const [open, setOpen] = useState(readOpenPref);
  const [busy, setBusy] = useState(false);
  const [banner, setBanner] = useState("");
  const [tip, setTip] = useState("");

  const [bases, setBases] = useState([]);
  const [selectedId, setSelectedId] = useState("");
  const [newName, setNewName] = useState("");

  const [sources, setSources] = useState([]);
  const [ingestFilename, setIngestFilename] = useState("");
  const [ingestContent, setIngestContent] = useState("");

  const [queryText, setQueryText] = useState("");
  const [queryTopK, setQueryTopK] = useState(5);
  const [queryHits, setQueryHits] = useState([]);
  const [queryContext, setQueryContext] = useState("");
  const [gwMeta, setGwMeta] = useState(null);
  const setOpenPersist = useCallback((next) => {
    setOpen((prev) => {
      const v = typeof next === "function" ? next(prev) : next;
      writeOpenPref(v);
      return v;
    });
  }, []);

  const setErr = useCallback((msg) => {
    setTip("");
    setBanner(msg ? String(msg) : "");
  }, []);

  const refreshBases = useCallback(async () => {
    setBusy(true);
    setErr("");
    try {
      const r = await api.knowledgeListBases();
      const list = r.bases || [];
      setBases(list);
      setSelectedId((prev) => {
        const stored = readPreferredKnowledgeBaseId();
        if (stored && !list.some((c) => c.id === stored)) {
          writePreferredKnowledgeBaseId("");
        }
        if (stored && list.some((c) => c.id === stored)) return stored;
        if (prev && list.some((c) => c.id === prev)) return prev;
        return list[0]?.id || "";
      });
    } catch (e) {
      setBases([]);
      setSelectedId("");
      setErr(e.message || "加载知识库失败（网关可能未启用 Knowledge）");
    } finally {
      setBusy(false);
    }
  }, [setErr]);

  const refreshSources = useCallback(async (cid) => {
    if (!cid) {
      setSources([]);
      return;
    }
    setBusy(true);
    setErr("");
    try {
      const r = await api.knowledgeListDocuments(cid);
      setSources(r.documents || []);
    } catch (e) {
      setSources([]);
      setErr(e.message || "加载来源失败");
    } finally {
      setBusy(false);
    }
  }, [setErr]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const m = await api.gatewayMeta();
        if (!cancelled) setGwMeta(m);
      } catch {
        if (!cancelled) setGwMeta(null);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!open) return;
    refreshBases();
  }, [open, refreshBases]);

  useEffect(() => {
    if (!open || !selectedId) {
      setSources([]);
      return;
    }
    refreshSources(selectedId);
  }, [open, selectedId, refreshSources]);

  const onCreate = async () => {
    const name = newName.trim();
    if (!name || disabled || busy) return;
    setBusy(true);
    setErr("");
    try {
      const r = await api.knowledgeCreateBase({ name });
      setNewName("");
      await refreshBases();
      if (r && r.id) {
        setSelectedId(r.id);
        writePreferredKnowledgeBaseId(r.id);
      }
    } catch (e) {
      setErr(e.message || "创建失败");
    } finally {
      setBusy(false);
    }
  };

  const onIngest = async () => {
    if (!selectedId || disabled || busy) return;
    const content = ingestContent.trim();
    if (!content) {
      setErr("请先填写要入库的正文");
      return;
    }
    setBusy(true);
    setErr("");
    try {
      await api.knowledgeIngest(selectedId, {
        filename: ingestFilename.trim() || undefined,
        content,
      });
      setIngestContent("");
      await refreshSources(selectedId);
    } catch (e) {
      setErr(e.message || "入库失败");
    } finally {
      setBusy(false);
    }
  };

  const onBatchFiles = async (e) => {
    const files = e.target.files;
    if (!selectedId || disabled || busy || !files?.length) return;
    setBusy(true);
    setErr("");
    try {
      const items = [];
      for (const f of files) {
        const content = await f.text();
        items.push({ filename: f.name, content });
      }
      await api.knowledgeIngestBatch(selectedId, { items });
      e.target.value = "";
      await refreshSources(selectedId);
    } catch (err) {
      setErr(err.message || "批量入库失败");
    } finally {
      setBusy(false);
    }
  };

  const onDeleteSource = async (sid) => {
    if (!selectedId || disabled || busy) return;
    if (!confirm("删除该来源及其分块？")) return;
    setBusy(true);
    setErr("");
    try {
      await api.knowledgeDeleteDocument(selectedId, sid);
      await refreshSources(selectedId);
    } catch (e) {
      setErr(e.message || "删除失败");
    } finally {
      setBusy(false);
    }
  };

  const onQuery = async () => {
    if (!selectedId || disabled || busy) return;
    const q = queryText.trim();
    if (!q) {
      setErr("检索问题不能为空");
      return;
    }
    setBusy(true);
    setErr("");
    try {
      const body = { query: q, topK: queryTopK > 0 ? queryTopK : 0 };
      const r = await api.knowledgeQuery(selectedId, body);
      setQueryHits(r.hits || []);
      setQueryContext(r.context || "");
    } catch (e) {
      setQueryHits([]);
      setQueryContext("");
      setErr(e.message || "检索失败");
    } finally {
      setBusy(false);
    }
  };

  const insertCollectionHint = () => {
    if (!selectedId) return;
    const line = `（上下文）当前网关知识库 ID：${selectedId}`;
    setInput((prev) => (prev && prev.trim() ? `${prev.trim()}\n${line}` : line));
    onInserted?.();
  };

  const copyCollectionId = useCallback(async () => {
    if (!selectedId) return;
    setErr("");
    try {
      await navigator.clipboard.writeText(selectedId);
      setTip("已复制知识库 ID");
    } catch {
      try {
        const ta = document.createElement("textarea");
        ta.value = selectedId;
        document.body.appendChild(ta);
        ta.select();
        document.execCommand("copy");
        document.body.removeChild(ta);
        setTip("已复制知识库 ID");
      } catch {
        setErr("无法访问剪贴板，请手动复制下拉框对应值");
      }
    }
  }, [selectedId, setErr]);

  const onBindToSession = async () => {
    if (!sessionId || !selectedId || disabled || busy) return;
    const meta = bases.find((b) => b.id === selectedId);
    setBusy(true);
    setErr("");
    setTip("");
    try {
      await api.bindSessionKnowledge(sessionId, {
        knowledgeBaseId: selectedId,
        knowledgeBaseName: meta?.name || "",
      });
      setTip("已绑定到当前会话，模型可使用 search_knowledge_base");
      writePreferredKnowledgeBaseId(selectedId);
      onKnowledgeBound?.();
    } catch (e) {
      setErr(e.message || "绑定失败");
    } finally {
      setBusy(false);
    }
  };

  const onDeleteCollection = async () => {
    if (!selectedId || disabled || busy) return;
    if (!confirm(`确认删除整个知识库及其全部向量与文档？\n${selectedId}`)) return;
    setBusy(true);
    setErr("");
    setTip("");
    try {
      const gone = selectedId;
      await api.knowledgeDeleteBase(selectedId);
      if (readPreferredKnowledgeBaseId() === gone) writePreferredKnowledgeBaseId("");
      setQueryHits([]);
      setQueryContext("");
      setIngestContent("");
      setSources([]);
      await refreshBases();
      setTip("已删除该知识库");
    } catch (e) {
      setErr(e.message || "删除知识库失败");
    } finally {
      setBusy(false);
    }
  };

  return html`
    <div class="rag-panel-wrap">
      <button type="button" class="btn sm ghost rag-toggle" onClick=${() => setOpenPersist((v) => !v)}>
        知识库 ${open ? "▲" : "▼"}
      </button>
      ${open && html`
        <div class="rag-panel" aria-label="知识库管理">
          ${gwMeta && html`<div class="rag-meta subtle">
            当前网关知识库后端：<strong>${gwMeta.knowledgeBackend || "—"}</strong>
            （${gwMeta.knowledge || "—"}${gwMeta.knowledge_error ? ` · ${gwMeta.knowledge_error}` : ""}）
          </div>`}
          ${banner && html`<div class="rag-banner">${banner}</div>`}
          ${tip && html`<div class="rag-tip">${tip}</div>`}
          <div class="rag-row">
            <button type="button" class="btn sm" disabled=${disabled || busy} onClick=${refreshBases}>刷新列表</button>
          </div>
          ${sessionId && html`
            <div class="rag-row rag-session-bind">
              <div class="rag-session-bind-text">
                <span class="muted">当前会话</span>
                ${boundKnowledge && boundKnowledge.knowledgeBaseId
                  ? html`
                      <code class="rag-bound-id" title=${boundKnowledge.knowledgeBaseId}>${boundKnowledge.knowledgeBaseId.slice(0, 10)}…</code>
                      <span class="rag-bound-name">${boundKnowledge.knowledgeBaseName || ""}</span>
                    `
                  : html`<span class="muted">未绑定 · Agent 工具 search_knowledge_base 需先绑定</span>`}
              </div>
              <button
                type="button"
                class="btn sm primary"
                disabled=${disabled || busy || !selectedId}
                onClick=${onBindToSession}
              >
                绑定选中库
              </button>
            </div>
          `}
          <div class="rag-row">
            <label class="rag-label">知识库</label>
            <select
              class="rag-input"
              value=${selectedId}
              disabled=${disabled || busy || bases.length === 0}
              onChange=${(e) => {
                const v = e.target.value;
                setSelectedId(v);
                writePreferredKnowledgeBaseId(v);
              }}
            >
              ${bases.length === 0
                ? html`<option value="">（无）</option>`
                : bases.map((c) => html`<option value=${c.id}>${c.name || c.id}</option>`)}
            </select>
          </div>
          ${selectedId && html`
            <div class="rag-id-row">
              <code class="rag-id-code" title="知识库 ID">${selectedId}</code>
              <button type="button" class="btn sm" disabled=${disabled || busy} onClick=${copyCollectionId}>复制 ID</button>
              <button type="button" class="btn sm danger" disabled=${disabled || busy} onClick=${onDeleteCollection}>删除知识库</button>
            </div>
          `}
          <div class="rag-row rag-split">
            <input
              type="text"
              class="rag-input"
              placeholder="新建知识库名称"
              value=${newName}
              disabled=${disabled || busy}
              onInput=${(e) => setNewName(e.target.value)}
            />
            <button type="button" class="btn sm" disabled=${disabled || busy || !newName.trim()} onClick=${onCreate}>
              创建
            </button>
          </div>
          <div class="rag-row">
            <button type="button" class="btn sm ghost" disabled=${disabled || busy || !selectedId} onClick=${insertCollectionHint}>
              插入知识库 ID 到输入框
            </button>
          </div>

          <div class="rag-section-title">入库</div>
          <div class="rag-row">
            <label class="rag-label">文件名</label>
            <input
              type="text"
              class="rag-input"
              placeholder="可选，如 notes.md"
              value=${ingestFilename}
              disabled=${disabled || busy || !selectedId}
              onInput=${(e) => setIngestFilename(e.target.value)}
            />
          </div>
          <textarea
            class="rag-textarea"
            rows="4"
            placeholder="粘贴要分块入库的正文…"
            value=${ingestContent}
            disabled=${disabled || busy || !selectedId}
            onInput=${(e) => setIngestContent(e.target.value)}
          ></textarea>
          <div class="rag-row">
            <button type="button" class="btn sm primary" disabled=${disabled || busy || !selectedId} onClick=${onIngest}>
              入库
            </button>
          </div>
          <div class="rag-row rag-batch">
            <input
              type="file"
              class="rag-file"
              multiple
              disabled=${disabled || busy || !selectedId}
              onChange=${onBatchFiles}
            />
            <span class="muted">可选：多文件 UTF-8 文本批量入库</span>
          </div>

          <div class="rag-section-title">文档 (${sources.length})</div>
          <ul class="rag-sources">
            ${sources.length === 0 && html`<li class="muted">暂无来源</li>`}
            ${sources.map(
              (s) => html`
                <li>
                  <span class="rag-src-name">${s.filename || s.id}</span>
                  <span class="rag-src-meta">${s.chunks != null ? `${s.chunks} 块` : ""}</span>
                  <button
                    type="button"
                    class="btn sm ghost"
                    disabled=${disabled || busy}
                    onClick=${() => onDeleteSource(s.id)}
                  >删</button>
                </li>
              `,
            )}
          </ul>

          <div class="rag-section-title">试检索</div>
          <div class="rag-row rag-split">
            <input
              type="text"
              class="rag-input"
              placeholder="问题 / 关键词"
              value=${queryText}
              disabled=${disabled || busy || !selectedId}
              onInput=${(e) => setQueryText(e.target.value)}
            />
            <input
              type="number"
              min="0"
              class="rag-input rag-topk"
              title="TopK，0 表示用服务端默认"
              value=${queryTopK}
              disabled=${disabled || busy || !selectedId}
              onInput=${(e) => setQueryTopK(Number(e.target.value) || 0)}
            />
          </div>
          <div class="rag-row">
            <button type="button" class="btn sm" disabled=${disabled || busy || !selectedId} onClick=${onQuery}>检索</button>
          </div>
          ${queryHits.length > 0 && html`
            <div class="rag-hits">
              ${queryHits.map(
                (h, i) => html`
                  <div class="rag-hit" key=${h.sourceId || h.text || `hit-${i}`}>
                    <div class="rag-hit-score">${typeof h.score === "number" ? h.score.toFixed(4) : ""}</div>
                    <pre class="rag-hit-text">${h.text || ""}</pre>
                  </div>
                `,
              )}
            </div>
          `}
          ${queryContext && html`
            <div class="rag-ctx-label">拼接上下文</div>
            <pre class="rag-ctx">${queryContext}</pre>
          `}
          <p class="rag-hint">绑定会话后，aisolo 可调用 search_knowledge_base。下拉所选知识库会记住；新建会话时自动带上该库（无需每次再点绑定）。网关侧入库与试检索亦在此完成。</p>
        </div>
      `}
    </div>
  `;
}
