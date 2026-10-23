import type { ApiMessages, ApiPage, ApiRecordings, DialSipReply, JoinReply, MeetingInfo, MeetingRecording, MeetingRecordState, ParticipantInfo, SipProviderInfo, TicketReply } from '../types'

const API_ROOT = import.meta.env.VITE_API_ROOT || '/live/v1'

/** 认证类业务码：100401 未认证、130101 应用不存在/停用、130102 密钥无效（见 docs/error-codes.md §6.3） */
const AUTH_CODES = [100401, 130101, 130102]
const SESSION_EXPIRED_EVENT = 'live:session-expired'
/** 非信封错误只截取一小段文本，避免整页 HTML 进入 toast */
const MAX_TEXT_SUMMARY = 120

export class ApiError extends Error {
  /** 业务码（信封 code）；非信封错误（nginx/框架）统一为 -1 */
  code: number
  /** HTTP 状态；业务失败时网关仍返回 200，不能用来判断业务成败 */
  status: number
  /** 网关回显的 X-Request-Id；非信封错误可能为空 */
  requestId: string
  constructor(message: string, status = 0, options: { code?: number; requestId?: string } = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = options.code ?? -1
    this.requestId = options.requestId ?? ''
  }
}

/** 登录失效判定：只认信封 code；非信封 HTTP 401 在请求层已归一为 100401。 */
export function isAuthError(error: unknown): error is ApiError {
  return error instanceof ApiError && AUTH_CODES.includes(error.code)
}

/** 网关统一信封：code 为数字、msg 为字符串（data 可缺省） */
type Envelope = { code: number; msg: string; data?: unknown }

function isEnvelope(value: unknown): value is Envelope {
  if (typeof value !== 'object' || value === null) return false
  const record = value as Record<string, unknown>
  return typeof record.code === 'number' && typeof record.msg === 'string'
}

function textSummary(text: string): string {
  const compact = text.replace(/\s+/g, ' ').trim()
  if (!compact) return ''
  return compact.length > MAX_TEXT_SUMMARY ? `${compact.slice(0, MAX_TEXT_SUMMARY)}…` : compact
}

/** 并发请求同时失效时只广播一次，避免重复登出提示 */
let lastSessionExpiredAt = 0
function notifySessionExpired(): void {
  const now = Date.now()
  if (now - lastSessionExpiredAt < 1000) return
  lastSessionExpiredAt = now
  window.dispatchEvent(new CustomEvent(SESSION_EXPIRED_EVENT))
}

function authHeaders(): HeadersInit {
  const token = localStorage.getItem('live_jwt')
  return token ? { Authorization: `Bearer ${token}` } : {}
}

async function request<T>(path: string, init: RequestInit = {}, requiresAuth = true): Promise<T> {
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  if (requiresAuth) Object.entries(authHeaders()).forEach(([key, value]) => headers.set(key, value))
  if (init.body) headers.set('Content-Type', 'application/json')
  let response: Response
  try {
    response = await fetch(`${API_ROOT}${path}`, { ...init, headers })
  } catch {
    console.warn('[api] 请求失败', { path, code: -1, requestId: '', status: 0, message: 'network error' })
    throw new ApiError('网络连接失败，请检查网关地址', 0, { code: -1 })
  }

  const requestId = response.headers.get('X-Request-Id') || ''
  // 先读文本再尝试解析，避免非 JSON（nginx 502、框架纯文本）直接抛错丢失状态信息
  const raw = await response.text().catch(() => '')
  let payload: unknown
  if (raw) {
    try { payload = JSON.parse(raw) } catch { payload = undefined }
  }
  const envelope = isEnvelope(payload) ? payload : null
  const httpStatus = response.status

  // 认证失败：业务码优先；HTTP 401 只作为非信封兜底，且匿名接口不触发全局登出。
  const authCode = envelope && AUTH_CODES.includes(envelope.code)
  const nonEnvelope401 = !envelope && httpStatus === 401
  if (requiresAuth && (authCode || nonEnvelope401)) {
    const code = envelope?.code ?? 100401
    console.warn('[api] 认证失败', { path, code, requestId, status: httpStatus })
    // 统一会话过期处理：广播事件，由 App 监听并登出（任何调用路径都会触发）
    notifySessionExpired()
    throw new ApiError('登录已失效，请重新登录', httpStatus, { code, requestId })
  }

  if (envelope) {
    if (envelope.code === 0) {
      // void 接口的 data 缺省时返回 undefined
      return (envelope.data === undefined ? undefined : envelope.data) as T
    }
    const message = envelope.msg.trim() || `请求失败（code ${envelope.code}）`
    console.warn('[api] 业务失败', { path, code: envelope.code, requestId, status: httpStatus, message })
    throw new ApiError(message, httpStatus, { code: envelope.code, requestId })
  }

  const summary = textSummary(raw)
  // livegtw 业务响应已全部走信封，2xx 却不是信封说明链路异常（旧网关、错误代理等）
  if (response.ok) {
    console.warn('[api] 响应不是统一信封', { path, code: -1, requestId, status: httpStatus, message: summary })
    throw new ApiError('响应格式异常：网关未返回统一信封，请检查网关版本或代理配置', httpStatus, { code: -1, requestId })
  }

  const message = httpStatus === 504
    ? '网关响应超时，请稍后重试'
    : httpStatus === 502
      ? '网关不可达，请稍后重试'
      : `请求失败（HTTP ${httpStatus}）${summary ? `：${summary}` : ''}`
  console.warn('[api] HTTP 错误', { path, code: -1, requestId, status: httpStatus, message: summary })
  throw new ApiError(message, httpStatus, { code: -1, requestId })
}

const json = (body: unknown): RequestInit => ({ method: 'POST', body: JSON.stringify(body) })

export const api = {
  getCurrentUser: () => request<{ authType: string; userId: string; userName: string; deptCode: string }>('/getCurrentUser'),
  createMeeting: (title: string) => request<{ meeting: MeetingInfo }>('/createMeeting', json({ title })),
  joinMeeting: (meetingNo: string, options?: { meetingCode?: string; canPublish?: boolean; canSubscribe?: boolean; canPublishData?: boolean; canPublishSources?: string[] }) => {
    const body: Record<string, unknown> = {
      canPublish: options?.canPublish ?? true,
      canSubscribe: options?.canSubscribe ?? true,
      canPublishData: options?.canPublishData ?? true,
      canPublishSources: options?.canPublishSources ?? ['camera', 'microphone', 'screen_share']
    }
    if (options?.meetingCode) {
      body.meetingCode = options.meetingCode
    } else {
      body.meetingNo = meetingNo
    }
    return request<JoinReply>('/joinMeeting', json(body))
  },
  notifyMeetingParticipant: (meetingNo: string, identity: string) => request<{ requestId: string }>('/notifyMeetingParticipant', json({ meetingNo, identity })),
  joinByTicket: (ticket: string) => request<JoinReply>(`/joinMeetingByTicket?ticket=${encodeURIComponent(ticket)}`, {}, false),
  getMeeting: (meetingNo: string) => request<{ meeting: MeetingInfo }>(`/getMeeting?meetingNo=${encodeURIComponent(meetingNo)}`),
  listMeetings: (params: Record<string, string | number> = {}) => request<ApiPage<MeetingInfo>>(`/listMeetings?${new URLSearchParams(Object.entries(params).map(([k, v]) => [k, String(v)]))}`),
  listMyMeetings: (page = 1, pageSize = 20, options?: { status?: number; title?: string; hasRecording?: number }) => {
    const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) })
    if (options?.status) params.set('status', String(options.status))
    if (options?.title) params.set('title', options.title)
    if (options?.hasRecording) params.set('hasRecording', String(options.hasRecording))
    return request<ApiPage<MeetingInfo>>(`/myMeetings?${params.toString()}`)
  },
  endMeeting: (meetingNo: string) => request<void>('/endMeeting', json({ meetingNo })),
  kickParticipant: (meetingNo: string, identity: string) => request<void>('/kickParticipant', json({ meetingNo, identity })),
  muteParticipant: (meetingNo: string, identity: string, muted: boolean, kind: 'audio' | 'video') => request<void>('/muteParticipant', json({ meetingNo, identity, muted, kind })),
  listParticipants: (meetingNo: string, options?: { excludeKinds?: number[] }) => {
    const params = new URLSearchParams({ meetingNo })
    // excludeKinds 逐项排除参与者类型（如 2=EGRESS 录屏机器人），空则不传、返回全部
    if (options?.excludeKinds?.length) options.excludeKinds.forEach((kind) => params.append('excludeKinds', String(kind)))
    return request<{ participants: ParticipantInfo[] }>(`/listParticipants?${params.toString()}`)
  },
  sendMeetingData: (meetingNo: string, topic: string, payload: string, destinations: string[] = []) => request<void>('/sendMeetingData', json({ meetingNo, topic, payload, destinations })),
  performRpc: (meetingNo: string, identity: string, payload: string) => request<{ response: string }>('/performMeetingRpc', json({ meetingNo, identity, method: 'echo', payload, responseTimeoutMs: 5000 })),
  generateTicket: (meetingNo: string, identity: string, name: string, expireSeconds: number, canPublish = true, canSubscribe = true, canPublishData = true, canPublishSources: string[] = [], ticketType = 1) => request<TicketReply>('/generateMeetingTicket', json({ meetingNo, identity, name, expireSeconds, canPublish, canSubscribe, canPublishData, canPublishSources, ticketType })),
  reportMessage: (message: { meetingNo: string; content: string; messageType: string }) => request<{ messageId: string }>('/reportMeetingMessage', json(message)),
  listMessages: (meetingNo: string) => request<ApiMessages>(`/listMeetingMessages?meetingNo=${encodeURIComponent(meetingNo)}&page=1&pageSize=100`),
  dialSip: (params: { calleeNumber: string; meetingNo?: string; participantName?: string; providerCode: string }) => request<DialSipReply>('/sip-calls/dial', json(params)),
  listSipProviders: () => request<{ providers: SipProviderInfo[] }>('/sip-providers'),
  createSipProvider: (params: { code: string; name: string; address: string; numbers: string[]; authUsername?: string; authPassword?: string }) => request<{ provider: SipProviderInfo }>('/sip-providers', json(params)),
  updateSipProvider: (params: { id: string; name?: string; address?: string; numbers?: string[]; authUsername?: string; authPassword?: string; status?: number }) => request<{ provider: SipProviderInfo }>('/sip-providers/update', json(params)),
  deleteSipProvider: (id: string) => request<void>('/sip-providers/delete', json({ id })),
  generateToken: (params: { authType: 'user' | 'device'; userId: string; userName?: string; deptCode?: string; expireSeconds?: number; signKey: string }) => request<{ token: string; expireTime: string }>('/generateToken', json(params), false),
  startMeetingRecord: (meetingNo: string, audioOnly = false, layout = '') => request<{ recording: MeetingRecording }>('/startMeetingRecord', json({ meetingNo, audioOnly, layout })),
  stopMeetingRecord: (params: { meetingNo?: string; recordId?: string }) => request<void>('/stopMeetingRecord', json(params)),
  listMeetingRecordings: (meetingNo: string, page = 1, pageSize = 10) => request<ApiRecordings>(`/listMeetingRecordings?meetingNo=${encodeURIComponent(meetingNo)}&page=${page}&pageSize=${pageSize}`),
  getMeetingRecording: (recordId: string) => request<{ recording: MeetingRecording }>(`/getMeetingRecording?recordId=${encodeURIComponent(recordId)}`),
  getMeetingRecordState: (meetingNo: string) => request<MeetingRecordState>(`/getMeetingRecordState?meetingNo=${encodeURIComponent(meetingNo)}`),
}
