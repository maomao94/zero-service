import type { ApiMessages, ApiPage, DialSipReply, JoinReply, MeetingInfo, ParticipantInfo, SipProviderInfo, TicketReply } from '../types'

const API_ROOT = import.meta.env.VITE_API_ROOT || '/live/v1'

export class ApiError extends Error {
  status: number
  constructor(message: string, status = 0) {
    super(message)
    this.status = status
  }
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
    throw new ApiError('网络连接失败，请检查网关地址')
  }
  const payload = await response.json().catch(() => ({}))
  if (!response.ok || (payload.code !== undefined && payload.code !== 0)) {
    const message = payload.msg || payload.message || `请求失败（HTTP ${response.status}）`
    throw new ApiError(response.status === 401 || payload.code === 104101 ? '登录已失效，请重新登录' : message, response.status)
  }
  return (payload.data === undefined ? payload : payload.data) as T
}

const json = (body: unknown): RequestInit => ({ method: 'POST', body: JSON.stringify(body) })

export const api = {
  getCurrentUser: () => request<{ userId: string; userName: string; deptCode: string }>('/getCurrentUser'),
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
  listMyMeetings: (page = 1, pageSize = 20) => request<ApiPage<MeetingInfo>>(`/myMeetings?page=${page}&pageSize=${pageSize}`),
  endMeeting: (meetingNo: string) => request<void>('/endMeeting', json({ meetingNo })),
  kickParticipant: (meetingNo: string, identity: string) => request<void>('/kickParticipant', json({ meetingNo, identity })),
  muteParticipant: (meetingNo: string, identity: string, muted: boolean, kind: 'audio' | 'video') => request<void>('/muteParticipant', json({ meetingNo, identity, muted, kind })),
  listParticipants: (meetingNo: string) => request<{ participants: ParticipantInfo[] }>(`/listParticipants?meetingNo=${encodeURIComponent(meetingNo)}`),
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
}
