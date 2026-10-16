import type { ParticipantInfo } from '../types'

/** 参与者类型（对齐 LiveKit ParticipantInfo_Kind）。 */
export const PARTICIPANT_KIND_EGRESS = 2

export type LiveParticipantView = {
  identity: string
  name?: string
  /** 参与者类型（LiveKit Participant.kind）；2=EGRESS（录屏机器人） */
  kind?: number
}

export type MeetingMember = {
  identity: string
  name: string
  online: boolean
}

/** 是否为 Egress（录屏机器人）参与者，UI 需屏蔽。 */
export function isEgressParticipant(participant: { kind?: number }): boolean {
  return participant.kind === PARTICIPANT_KIND_EGRESS
}

/** 过滤掉 Egress（录屏机器人）参与者，返回真实业务参与者。 */
export function excludeEgressParticipants<T extends { kind?: number }>(participants: T[]): T[] {
  return participants.filter((participant) => !isEgressParticipant(participant))
}

export function resolveLiveKitUrl(protocol: string, host: string, configuredUrl?: string): string {
  const browserProtocol = protocol === 'https:' ? 'wss:' : 'ws:'
  const configured = configuredUrl?.trim() || '/livekit'
  const url = new URL(configured, `${protocol}//${host}`)
  if (url.protocol === 'http:') url.protocol = 'ws:'
  if (url.protocol === 'https:') url.protocol = 'wss:'
  if (url.protocol !== 'ws:' && url.protocol !== 'wss:') url.protocol = browserProtocol
  return url.toString().replace(/\/$/, '')
}

export function normalizePublishSources(sources?: string[]): string[] | null {
  const normalized = sources?.map((source) => source.trim()).filter(Boolean) || []
  return normalized.length > 0 ? [...new Set(normalized)] : null
}

export function mergeMeetingParticipants(history: ParticipantInfo[], liveParticipants: LiveParticipantView[]): MeetingMember[] {
  const members = new Map<string, MeetingMember>()
  history.forEach((participant) => {
    const identity = participant.identity.trim()
    if (identity) members.set(identity, { identity, name: participant.name || identity, online: false })
  })
  liveParticipants.forEach((participant) => {
    const identity = participant.identity.trim()
    if (!identity) return
    const existing = members.get(identity)
    members.set(identity, { identity, name: participant.name || existing?.name || identity, online: true })
  })
  return [...members.values()]
}

export function countLiveParticipants(participants: LiveParticipantView[]): number {
  return new Set(participants.map((participant) => participant.identity.trim()).filter(Boolean)).size
}
