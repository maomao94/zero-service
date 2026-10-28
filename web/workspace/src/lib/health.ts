export interface ProbeResult {
  ok: boolean
  /** 成功时的往返耗时（毫秒） */
  ms: number | null
  /** 失败原因，用于卡片提示 */
  error?: string
}

/**
 * 探测同源健康检查地址。
 * 同源代理下不会触发跨域，也不依赖子系统自身的 CORS 配置。
 */
export async function probe(url: string, timeoutMs = 2500): Promise<ProbeResult> {
  const started = performance.now()
  const controller = new AbortController()
  const timer = window.setTimeout(() => controller.abort(), timeoutMs)

  try {
    const response = await fetch(url, { cache: 'no-store', signal: controller.signal })
    const ms = Math.round(performance.now() - started)
    if (!response.ok) {
      return { ok: false, ms: null, error: `HTTP ${response.status}` }
    }
    return { ok: true, ms }
  } catch (error) {
    const timeout = error instanceof DOMException && error.name === 'AbortError'
    return { ok: false, ms: null, error: timeout ? '请求超时' : '连接失败' }
  } finally {
    window.clearTimeout(timer)
  }
}
