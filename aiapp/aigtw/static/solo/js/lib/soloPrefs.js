/** 与 RagPanel / 新建会话共享：记住用户上次选中的知识库 ID */

const PREFERRED_KB_ID = "solo.preferredKnowledgeBaseId";

export function readPreferredKnowledgeBaseId() {
  try {
    const s = localStorage.getItem(PREFERRED_KB_ID);
    return s && s.trim() ? s.trim() : "";
  } catch {
    return "";
  }
}

export function writePreferredKnowledgeBaseId(id) {
  try {
    const v = id != null ? String(id).trim() : "";
    if (v) localStorage.setItem(PREFERRED_KB_ID, v);
    else localStorage.removeItem(PREFERRED_KB_ID);
  } catch {
    /* ignore */
  }
}
