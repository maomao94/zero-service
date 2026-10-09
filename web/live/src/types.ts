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

export interface MeetingRecording {
  recordId: string
  meetingNo: string
  egressId: string
  /** 录制状态（对齐 LiveKit EgressStatus：0-启动中,1-录制中,2-收尾中,3-已完成,4-失败,5-已中止,6-超限） */
  status: number
  fileName: string
  /** 播放地址（仅已完成或超限结束且已产出文件时有值） */
  fileUrl: string
  fileSize: number
  /** 录制时长（秒） */
  duration: number
  startTime: string
  endTime: string
  error: string
}

export interface ApiPage<T> {
  total: number
  meetings: T[]
}

export interface ApiRecordings {
  total: number
  recordings: MeetingRecording[]
}

export interface MeetingRecordState {
  /** 当前进行中的录制（无则为空对象） */
  recording: MeetingRecording
  /** 是否正在录制 */
  active: boolean
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
