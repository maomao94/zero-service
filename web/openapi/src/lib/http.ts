import type {
  AdminResult,
  GatewayCallResult,
  GatewayEnvelope,
} from '../types'

const DEFAULT_TIMEOUT = 120_000

export function normalizeBaseUrl(value: string): string {
  return value.trim().replace(/\/+$/, '')
}

export function buildUrl(baseUrl: string, path: string): string {
  const base = normalizeBaseUrl(baseUrl)
  return `${base}${path.startsWith('/') ? path : `/${path}`}`
}

export function gatewayUrl(baseUrl: string): string {
  return buildUrl(baseUrl, '/v1/gateway')
}

/** Basic 凭据中的 appId/secret 均为 ASCII，这里用 TextEncoder 兼容意外字符。 */
function basicAuthHeader(appId: string, secret: string): string {
  const bytes = new TextEncoder().encode(`${appId}:${secret}`)
  let binary = ''
  bytes.forEach((byte) => {
    binary += String.fromCharCode(byte)
  })
  return `Basic ${window.btoa(binary)}`
}

function isEnvelope(value: unknown): value is GatewayEnvelope {
  if (!value || typeof value !== 'object') return false
  const candidate = value as Partial<GatewayEnvelope>
  return typeof candidate.code === 'number' && typeof candidate.msg === 'string'
}

function parseBody(text: string): unknown {
  if (!text) return undefined
  try {
    return JSON.parse(text)
  } catch {
    return undefined
  }
}

/**
 * 非信封响应（nginx 502/504、go-zero 413/500 等）没有业务码，只能按 HTTP 状态给出文案。
 * 成功判定仍以信封 code===0 为准，这里只负责可读提示。
 */
function httpFailureMessage(status: number, statusText: string): string {
  switch (status) {
    case 401:
      return 'HTTP 401：缺少或无效的凭据'
    case 413:
      return 'HTTP 413：请求体超过网关上限'
    case 502:
      return 'HTTP 502：网关不可达，后端服务未启动或 nginx 无法连接上游'
    case 504:
      return 'HTTP 504：网关超时，后端在超时时间内没有响应'
    default: {
      const suffix = statusText ? ` ${statusText}` : ''
      return `HTTP ${status}${suffix}：响应不是有效的 {code,msg,data} 信封`
    }
  }
}

export interface GatewayCallInput {
  baseUrl: string
  appId: string
  secret: string
  signKey?: string
  method: string
  data: unknown
  tid: string
  timeoutMs?: number
}

async function hmacSha256(data: string, key: string): Promise<string> {
  const cryptoKey = await crypto.subtle.importKey('raw', new TextEncoder().encode(key), { name: 'HMAC', hash: 'SHA-256' }, false, ['sign'])
  const signature = await crypto.subtle.sign('HMAC', cryptoKey, new TextEncoder().encode(data))
  return Array.from(new Uint8Array(signature), (byte) => byte.toString(16).padStart(2, '0')).join('')
}

async function sha256(data: string): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(data))
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, '0')).join('')
}

/**
 * 调用 POST /v1/gateway。
 * 网络错误与业务错误都以结果对象返回，避免 UI 到处 try/catch。
 */
export async function callGateway(input: GatewayCallInput): Promise<GatewayCallResult> {
  const started = performance.now()
  const controller = new AbortController()
  const timeout = window.setTimeout(() => controller.abort(), input.timeoutMs ?? DEFAULT_TIMEOUT)

  try {
    // body 只序列化一次：签名对这段字节算 bodyHash，发送时也必须是同一段字节。
    // timestamp/nonce 放在 body 里，由 bodyHash 覆盖；签名头只放 X-Sign: v1:<hex>。
    const timestamp = Date.now()
    const nonce = crypto.randomUUID()
    const body: Record<string, unknown> = {
      method: input.method,
      data: input.data,
      tid: input.tid,
    }
    if (input.signKey) {
      body.timestamp = timestamp
      body.nonce = nonce
    }
    const bodyText = JSON.stringify(body)
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      Authorization: basicAuthHeader(input.appId, input.secret),
    }
    if (input.signKey) {
      const bodyHash = await sha256(bodyText)
      const canonical = ['v1', input.appId, '/v1/gateway', bodyHash].join('\n')
      headers['X-Sign'] = `v1:${await hmacSha256(canonical, input.signKey)}`
    }
    const response = await fetch(gatewayUrl(input.baseUrl), {
      method: 'POST',
      headers,
      body: bodyText,
      signal: controller.signal,
    })

    const raw = await response.text()
    const durationMs = performance.now() - started
    const envelope = parseBody(raw)
    const requestId = response.headers.get('X-Request-Id') ?? ''
    const sizeBytes = new TextEncoder().encode(raw).length

    if (isEnvelope(envelope)) {
      return {
        type: 'response',
        httpStatus: response.status,
        code: envelope.code,
        msg: envelope.msg,
        data: envelope.data,
        raw,
        requestId,
        durationMs,
        sizeBytes,
      }
    }

    // 非信封：没有业务码，保留 HTTP 状态并给出状态相关文案，code 固定为 -1。
    return {
      type: 'response',
      httpStatus: response.status,
      code: -1,
      msg: httpFailureMessage(response.status, response.statusText),
      data: undefined,
      raw,
      requestId,
      durationMs,
      sizeBytes,
    }
  } catch (error) {
    const durationMs = performance.now() - started
    const aborted = error instanceof DOMException && error.name === 'AbortError'
    if (aborted) {
      return {
        type: 'network',
        message: `请求超时或被取消（${Math.round((input.timeoutMs ?? DEFAULT_TIMEOUT) / 1000)}s）`,
        requestId: '',
        durationMs,
      }
    }
    return {
      type: 'network',
      message: `无法连接网关：${error instanceof Error ? error.message : '网络错误或 CORS 被拒绝'}`,
      requestId: '',
      durationMs,
    }
  } finally {
    window.clearTimeout(timeout)
  }
}

export interface HealthResult {
  ok: boolean
  status: number
  message: string
  durationMs: number
}

export async function checkHealth(baseUrl: string): Promise<HealthResult> {
  const started = performance.now()
  const controller = new AbortController()
  const timeout = window.setTimeout(() => controller.abort(), 6_000)
  try {
    const response = await fetch(buildUrl(baseUrl, '/v1/health'), {
      cache: 'no-store',
      signal: controller.signal,
    })
    const body = parseBody(await response.text())
    const envelope = isEnvelope(body)
    // 统一契约：只认信封 code===0；非信封（静态页、代理改写）一律判失败。
    const message = envelope ? body.msg : '响应不是统一信封，请检查 Base URL 或代理配置'
    return {
      ok: envelope ? body.code === 0 : false,
      status: response.status,
      message: message || (response.ok ? 'ok' : '服务不可用'),
      durationMs: performance.now() - started,
    }
  } catch (error) {
    return {
      ok: false,
      status: 0,
      message: error instanceof Error ? error.message : '无法连接服务',
      durationMs: performance.now() - started,
    }
  } finally {
    window.clearTimeout(timeout)
  }
}

export interface AdminRequestInput {
  baseUrl: string
  token: string
  path: string
  method?: 'GET' | 'POST'
  query?: Record<string, string | number | undefined>
  body?: unknown
  timeoutMs?: number
}

/** 管理面统一请求：返回 HTTP 状态、业务码与 data，401 时给出明确提示。 */
export async function adminRequest<T>(input: AdminRequestInput): Promise<AdminResult<T>> {
  if (!input.token.trim()) {
    return {
      ok: false,
      httpStatus: 401,
      code: 100401,
      msg: '请先填写管理面 JWT token',
      data: null,
      raw: '',
      requestId: '',
    }
  }

  const params = new URLSearchParams()
  Object.entries(input.query ?? {}).forEach(([key, value]) => {
    if (value !== undefined && value !== '') params.set(key, String(value))
  })
  const query = params.toString()
  const url = buildUrl(input.baseUrl, `${input.path}${query ? `?${query}` : ''}`)

  const controller = new AbortController()
  const timeout = window.setTimeout(() => controller.abort(), input.timeoutMs ?? 30_000)
  try {
    const response = await fetch(url, {
      method: input.method ?? 'GET',
      headers: {
        ...(input.body === undefined ? {} : { 'Content-Type': 'application/json' }),
        Authorization: `Bearer ${input.token.trim()}`,
      },
      body: input.body === undefined ? undefined : JSON.stringify(input.body),
      signal: controller.signal,
    })
    const raw = await response.text()
    const envelope = parseBody(raw)
    const requestId = response.headers.get('X-Request-Id') ?? ''

    if (isEnvelope(envelope)) {
      return {
        // 信封存在时只认 code===0，HTTP 200 不代表业务成功。
        ok: envelope.code === 0,
        httpStatus: response.status,
        code: envelope.code,
        msg: envelope.msg,
        data: (envelope.data as T | undefined) ?? null,
        raw,
        requestId,
      }
    }

    // 非信封：业务码不可用，仅对 401 映射为统一的未认证码，其余交给 HTTP 状态。
    // 管理面契约要求统一信封，非信封一律按失败处理，避免调用方误判成功。
    return {
      ok: false,
      httpStatus: response.status,
      code: response.status === 401 ? 100401 : -1,
      msg: httpFailureMessage(response.status, response.statusText),
      data: null,
      raw,
      requestId,
    }
  } catch (error) {
    const aborted = error instanceof DOMException && error.name === 'AbortError'
    return {
      ok: false,
      httpStatus: 0,
      code: -1,
      msg: aborted
        ? '管理面请求超时'
        : `无法连接管理面：${error instanceof Error ? error.message : '网络错误或 CORS 被拒绝'}`,
      data: null,
      raw: '',
      requestId: '',
    }
  } finally {
    window.clearTimeout(timeout)
  }
}
