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
  listModes:    ()        => request("GET",    "/modes"),
  listSkills:   ()        => request("GET",    "/skills"),
  listSessions: (p = {})  => request("GET",    `/sessions?page=${p.page || 1}&pageSize=${p.pageSize || 20}`),
  createSession:(p)       => request("POST",   "/sessions", p),
  getSession:   (id)      => request("GET",    `/sessions/${id}`),
  deleteSession:(id)      => request("DELETE", `/sessions/${id}`),
  listMessages: (id, lim = 200) => request("GET", `/sessions/${id}/messages?limit=${lim}`),
  getInterrupt: (iid)     => request("GET",    `/interrupt/${iid}`),
  // einox/rag（网关未启用 rag 时接口会报错，调用方自行 try/catch）
  ragListCollections: () => request("GET", "/rag/collections"),
  ragCreateCollection: (body) => request("POST", "/rag/collections", body),
  ragDeleteCollection: (id) => request("DELETE", `/rag/collections/${id}`),
  ragIngest: (collectionId, body) => request("POST", `/rag/collections/${collectionId}/ingest`, body),
  ragListSources: (collectionId) => request("GET", `/rag/collections/${collectionId}/sources`),
  ragDeleteSource: (collectionId, sourceId) => request("DELETE", `/rag/collections/${collectionId}/sources/${sourceId}`),
  ragQuery: (collectionId, body) => request("POST", `/rag/collections/${collectionId}/query`, body),
};

export function streamEndpoints() {
  return {
    chat:   `${BASE}/chat`,
    resume: (iid) => `${BASE}/interrupt/${iid}/resume`,
  };
}
