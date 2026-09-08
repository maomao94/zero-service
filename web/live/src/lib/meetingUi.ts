import type { ParticipantInfo } from '../types'

export type LiveParticipantView = {
  identity: string
  name?: string
}

export type MeetingMember = {
  identity: string
  name: string
  online: boolean
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
