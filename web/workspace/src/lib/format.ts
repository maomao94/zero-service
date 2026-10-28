/** 时间与地址的展示格式，集中处理避免各组件各写一套 */

export function clockTime(at: number | null | undefined): string {
  if (!at) return '—'
  return new Date(at).toLocaleTimeString('zh-CN', { hour12: false })
}

export function dateTime(at: number | null | undefined): string {
  if (!at) return '—'
  const date = new Date(at)
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

/** 构建时间由 ISO 字符串注入，统一按本地时区展示到分钟 */
export function isoToDateTime(iso: string): string {
  const parsed = Date.parse(iso)
  return Number.isNaN(parsed) ? '—' : dateTime(parsed)
}

export function hostOf(url: string): string {
  try {
    return new URL(url).host
  } catch {
    return url
  }
}

export function latencyText(ms: number | null): string {
  return ms === null ? '—' : `${ms} ms`
}

/** 站点标识：当前协议 + 主机（含端口），用于环境面板与端点拼接 */
export function siteOrigin(): string {
  return location.origin
}
