import type { HistoryEntry, Settings } from '../types'
import { DEFAULT_METHOD } from './methods'

const SETTINGS_KEY = 'zero.openapi.settings.v1'
const HISTORY_KEY = 'zero.openapi.history.v1'
const HISTORY_LIMIT = 50

const envBaseUrl = (import.meta.env.VITE_OPENAPI_BASE_URL ?? '').trim()

export const DEFAULT_SETTINGS: Settings = {
  baseUrl: envBaseUrl,
  appId: '',
  secret: '',
  signKey: '',
  adminToken: '',
  lastMethod: DEFAULT_METHOD.key,
}

function readText(key: string): string | null {
  try {
    return window.localStorage.getItem(key)
  } catch {
    return null
  }
}

export function loadSettings(): Settings {
  const raw = readText(SETTINGS_KEY)
  if (!raw) return { ...DEFAULT_SETTINGS }
  try {
    const parsed = JSON.parse(raw) as Partial<Settings>
    return {
      baseUrl: typeof parsed.baseUrl === 'string' ? parsed.baseUrl : DEFAULT_SETTINGS.baseUrl,
      appId: typeof parsed.appId === 'string' ? parsed.appId : '',
      secret: typeof parsed.secret === 'string' ? parsed.secret : '',
      signKey: typeof parsed.signKey === 'string' ? parsed.signKey : '',
      adminToken: typeof parsed.adminToken === 'string' ? parsed.adminToken : '',
      lastMethod: typeof parsed.lastMethod === 'string' ? parsed.lastMethod : DEFAULT_SETTINGS.lastMethod,
    }
  } catch {
    return { ...DEFAULT_SETTINGS }
  }
}

export function saveSettings(settings: Settings): void {
  try {
    window.localStorage.setItem(SETTINGS_KEY, JSON.stringify(settings))
  } catch {
    // localStorage 不可用时静默降级，不影响当前会话使用。
  }
}

export function loadHistory(): HistoryEntry[] {
  const raw = readText(HISTORY_KEY)
  if (!raw) return []
  try {
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter(isHistoryEntry).slice(0, HISTORY_LIMIT)
  } catch {
    return []
  }
}

export function saveHistory(entries: HistoryEntry[]): void {
  try {
    window.localStorage.setItem(HISTORY_KEY, JSON.stringify(entries.slice(0, HISTORY_LIMIT)))
  } catch {
    // 忽略配额或隐私模式错误。
  }
}

export function clearStoredCredentials(): void {
  try {
    const settings = loadSettings()
    window.localStorage.setItem(SETTINGS_KEY, JSON.stringify({
      ...settings,
      appId: '',
      secret: '',
      signKey: '',
      adminToken: '',
    } satisfies Settings))
  } catch {
    // ignore
  }
}

function isHistoryEntry(value: unknown): value is HistoryEntry {
  if (!value || typeof value !== 'object') return false
  const item = value as Partial<HistoryEntry>
  return (
    typeof item.id === 'string' &&
    typeof item.at === 'string' &&
    typeof item.method === 'string' &&
    typeof item.tid === 'string' &&
    typeof item.dataText === 'string' &&
    typeof item.httpStatus === 'number' &&
    typeof item.code === 'number' &&
    typeof item.durationMs === 'number'
  )
}
