export type MeetingStatus = 1 | 2 | 3

export interface MeetingInfo {
  meetingNo: string
  title: string
  status: MeetingStatus
  createUser: string
  updateUser: string
  deptCode: string
  startTime: string
  endTime: string
  createTime: string
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
}

export interface ApiPage<T> {
  total: number
  meetings: T[]
}

export interface ApiMessages {
  total: number
  messages: MeetingMessage[]
}
