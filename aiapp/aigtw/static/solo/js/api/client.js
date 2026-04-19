// 轻量 HTTP 客户端. 附加 JWT, 统一 error 语义.
const TOKEN_KEY = "solo.jwt";

export function getToken() {
  return localStorage.getItem(TOKEN_KEY) || "";
}
export function setToken(t) {
  if (t) localStorage.setItem(TOKEN_KEY, t);
  else localStorage.removeItem(TOKEN_KEY);
}

const BASE = "/solo/v1";

async function request(method, path, body) {
  const headers = { "Content-Type": "application/json" };
  const tok = getToken();
  if (tok) headers["Authorization"] = `Bearer ${tok}`;
  const res = await fetch(BASE + path, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try {
      const err = await res.json();
      msg = err.msg || err.message || msg;
    } catch (_) { /* ignore */ }
    throw new Error(msg);
  }
  if (res.status === 204) return null;
  return res.json();
}

export const api = {
  gatewayMeta:  ()        => request("GET",    "/meta"),
  listModes:    ()        => request("GET",    "/modes"),
  listSkills:   ()        => request("GET",    "/skills"),
  listSessions: (p = {})  => request("GET",    `/sessions?page=${p.page || 1}&pageSize=${p.pageSize || 20}`),
  createSession:(p)       => request("POST",   "/sessions", p),
  bindSessionKnowledge: (sessionId, body) =>
    request("POST", `/sessions/${sessionId}/knowledge`, body),
  getSession:   (id)      => request("GET",    `/sessions/${id}`),
  deleteSession:(id)      => request("DELETE", `/sessions/${id}`),
  listMessages: (id, lim = 200) => request("GET", `/sessions/${id}/messages?limit=${lim}`),
  getInterrupt: (iid)     => request("GET",    `/interrupt/${iid}`),
  // einox/knowledge（网关未启用时接口会报错，调用方自行 try/catch）
  knowledgeListBases: () => request("GET", "/knowledge/bases"),
  knowledgeCreateBase: (body) => request("POST", "/knowledge/bases", body),
  knowledgeDeleteBase: (id) => request("DELETE", `/knowledge/bases/${id}`),
  knowledgeIngest: (baseId, body) => request("POST", `/knowledge/bases/${baseId}/ingest`, body),
  knowledgeIngestBatch: (baseId, body) => request("POST", `/knowledge/bases/${baseId}/ingest-batch`, body),
  knowledgeListDocuments: (baseId) => request("GET", `/knowledge/bases/${baseId}/documents`),
  knowledgeDeleteDocument: (baseId, sourceId) => request("DELETE", `/knowledge/bases/${baseId}/documents/${sourceId}`),
  knowledgeQuery: (baseId, body) => request("POST", `/knowledge/bases/${baseId}/query`, body),
};

export function streamEndpoints() {
  return {
    chat:   `${BASE}/chat`,
    resume: (iid) => `${BASE}/interrupt/${iid}/resume`,
  };
}
