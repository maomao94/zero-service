import { io, type Socket } from 'socket.io-client'

export const MEETING_INVITE_EVENT = 'live:meeting-invite'

function meetingInviteRoom(identity: string): string {
  return `${MEETING_INVITE_EVENT}:${identity}`
}

export type MeetingInvitation = {
  id: string
  meetingNo: string
  meetingCode: string
  meetingTitle: string
  identity: string
  userId: string
  userName: string
  invitedAt: string
  receivedAt: string
  read: boolean
}

type SocketDown = {
  event?: unknown
  payload?: unknown
  reqId?: unknown
}

function parseJson(value: unknown): unknown {
  if (typeof value !== 'string') return value
  try {
    return JSON.parse(value)
  } catch {
    return null
  }
}

function asRecord(value: unknown): Record<string, unknown> | null {
  const parsed = parseJson(value)
  return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed as Record<string, unknown> : null
}

export function parseMeetingInvitation(raw: unknown): MeetingInvitation | null {
  const envelope = asRecord(raw) as SocketDown | null
  if (!envelope || (envelope.event !== undefined && envelope.event !== MEETING_INVITE_EVENT)) return null
  const payload = asRecord(envelope.payload)
  if (!payload) return null

  const meetingNo = typeof payload.meetingNo === 'string' ? payload.meetingNo.trim() : ''
  const identity = typeof payload.identity === 'string' ? payload.identity.trim() : ''
  if (!meetingNo || !identity) return null

  const requestId = typeof envelope.reqId === 'string' ? envelope.reqId.trim() : ''
  const invitedAt = typeof payload.invitedAt === 'string' ? payload.invitedAt : ''
  return {
    id: requestId || `${meetingNo}:${identity}:${invitedAt || Date.now()}`,
    meetingNo,
    meetingCode: typeof payload.meetingCode === 'string' ? payload.meetingCode : '',
    meetingTitle: typeof payload.meetingTitle === 'string' ? payload.meetingTitle : '',
    identity,
    userId: typeof payload.userId === 'string' ? payload.userId.trim() : '',
    userName: typeof payload.userName === 'string' ? payload.userName.trim() : '',
    invitedAt,
    receivedAt: new Date().toISOString(),
    read: false,
  }
}

function requestId(): string {
  return crypto.randomUUID?.() || `socket-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export function connectMeetingNotifications(token: string, identity: string, onInvite: (invitation: MeetingInvitation) => void): Socket {
  const socket = io(import.meta.env.VITE_SOCKET_URL || location.origin, {
    auth: { token },
    transports: ['websocket', 'polling'],
    reconnection: true,
  })

  socket.on('connect', () => {
    socket.emit('__join_room_up__', { reqId: requestId(), room: meetingInviteRoom(identity) })
  })
  socket.on(MEETING_INVITE_EVENT, (raw: unknown) => {
    const invitation = parseMeetingInvitation(raw)
    if (invitation?.identity === identity) onInvite(invitation)
  })
  return socket
}
