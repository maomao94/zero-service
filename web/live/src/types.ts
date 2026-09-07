export type MeetingStatus = 1 | 2 | 3

export interface MeetingInfo {
  meetingNo: string
  meetingCode: string
  title: string
  status: MeetingStatus
  createUser: string
  updateUser: string
  deptCode: string
  startTime: string
  endTime: string
  createTime: string
  emptyTimeout: number
  departureTimeout: number
  maxParticipants: number
  roomSid: string
}

export interface ParticipantInfo {
  identity: string
  name: string
  status: number
  joinTime: string
  leftTime: string
}

export interface MeetingMessage {
  messageId: string
  senderId: string
  senderName: string
  content: string
  messageType: string
  createTime: string
}

export interface JoinReply {
  token: string
  wsUrl: string
  meeting: MeetingInfo
  canPublish: boolean
  canSubscribe: boolean
  canPublishData: boolean
  canPublishSources: string[]
}

export interface TicketReply {
  ticket: string
  expireTime: string
  canPublish: boolean
  canSubscribe: boolean
  canPublishData: boolean
  canPublishSources: string[]
  ticketType: number
}

export interface ApiPage<T> {
  total: number
  meetings: T[]
}

export interface ApiMessages {
  total: number
  messages: MeetingMessage[]
}

export interface SipProviderInfo {
  id: string
  code: string
  name: string
  address: string
  numbers: string[]
  status: number
  createTime: string
  sipTrunkId: string
}

export interface DialSipReply {
  meeting?: MeetingInfo
  sipCallId: string
}
