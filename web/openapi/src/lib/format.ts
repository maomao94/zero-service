/** 服务端时间统一为 "YYYY-MM-DD HH:mm:ss"；Safari 对空格格式解析不稳定，先归一为本地 ISO。 */
function parseApiDate(value: string): Date {
  return new Date(value.trim().replace(' ', 'T'))
}

export function formatDateTime(value?: string): string {
  if (!value) return '—'
  const date = parseApiDate(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  })
}

export function formatClock(value: string): string {
  const date = parseApiDate(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleTimeString('zh-CN', { hour12: false })
}

export function formatDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return '—'
  if (ms < 1000) return `${Math.round(ms)} ms`
  return `${(ms / 1000).toFixed(2)} s`
}

export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`
  return `${(bytes / 1024 / 1024).toFixed(2)} MiB`
}

export function generateTid(): string {
  const now = new Date()
  const pad = (value: number) => String(value).padStart(2, '0')
  const stamp = `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`
  const random = Math.random().toString(16).slice(2, 10)
  return `req-${stamp}-${random}`
}

export interface ParsedJson {
  value: unknown | null
  error: string | null
}

/** 网关 data 必须是 JSON 对象；数组和基础类型直接拒绝。 */
export function parseJsonObject(text: string): ParsedJson {
  const source = text.trim()
  if (!source) return { value: {}, error: null }
  try {
    const value: unknown = JSON.parse(source)
    if (value === null || typeof value !== 'object' || Array.isArray(value)) {
      return { value: null, error: 'data 必须是 JSON 对象（不能是数组、字符串或 null）' }
    }
    return { value, error: null }
  } catch (error) {
    return { value: null, error: `JSON 解析失败：${error instanceof Error ? error.message : '未知错误'}` }
  }
}

export function prettyJsonText(text: string): string {
  return JSON.stringify(JSON.parse(text), null, 2)
}

export function minifyJsonText(text: string): string {
  return JSON.stringify(JSON.parse(text))
}

export function prettyValue(value: unknown): string {
  if (value === undefined) return ''
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

export function shortId(value: string, keep = 10): string {
  if (value.length <= keep) return value
  return `${value.slice(0, keep)}…`
}
