/**
 * 本地签发 JWT（HS256），用于工作台联调，不请求后端。
 * 算法与 common/tool/jwtutil.go 的 signToken 一致：
 * header = {"alg":"HS256","typ":"JWT"}，payload 使用 user-id / auth-type 等 claims。
 */

export interface JwtClaims {
  [key: string]: string | number
}

export type AuthType = 'user' | 'device'

export interface ClaimsOptions {
  authType: AuthType
  id: string
  name?: string
  dept?: string
  expireSeconds: number
}

function base64Url(bytes: Uint8Array): string {
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

/** 生成与 livegtw GenerateUserToken / GenerateDeviceToken 相同的 claims */
export function buildClaims(options: ClaimsOptions): JwtClaims {
  const now = Math.floor(Date.now() / 1000)
  const claims: JwtClaims = {
    exp: now + options.expireSeconds,
    iat: now,
    'user-id': options.id,
    'auth-type': options.authType,
  }
  if (options.name) claims['user-name'] = options.name
  if (options.authType === 'user' && options.dept) claims['dept-code'] = options.dept
  return claims
}

/** 解码 JWT payload（不校验签名），用于后端不可达时按 claims 展示用户信息 */
export function decodeJwtClaims(token: string): Record<string, unknown> {
  try {
    const part = token.split('.')[1]
    if (!part) return {}
    const padded = part.replace(/-/g, '+').replace(/_/g, '/').padEnd(Math.ceil(part.length / 4) * 4, '=')
    const binary = atob(padded)
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
    return JSON.parse(new TextDecoder().decode(bytes)) as Record<string, unknown>
  } catch {
    return {}
  }
}

/** 使用 WebCrypto 签名；localhost / https 之外的 http 源站不支持 */
export async function signJwt(claims: JwtClaims, secret: string): Promise<string> {
  if (!secret) throw new Error('请输入签名密钥')
  if (!globalThis.crypto?.subtle) {
    throw new Error('当前环境不支持 WebCrypto，请通过 localhost 或 https 访问')
  }

  const encoder = new TextEncoder()
  const header = base64Url(encoder.encode(JSON.stringify({ alg: 'HS256', typ: 'JWT' })))
  const payload = base64Url(encoder.encode(JSON.stringify(claims)))
  const key = await crypto.subtle.importKey(
    'raw',
    encoder.encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  )
  const signature = await crypto.subtle.sign('HMAC', key, encoder.encode(`${header}.${payload}`))
  return `${header}.${payload}.${base64Url(new Uint8Array(signature))}`
}
