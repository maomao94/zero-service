// Token claims 解析工具（Token 由 live 子系统签发，此处仅解析展示）

// 解析 JWT payload（不校验签名），仅用于展示 claims
export function decodeJwtClaims(token: string): Record<string, unknown> {
  try {
    const payload = token.split('.')[1]
    if (!payload) return {}
    const normalized = payload.replace(/-/g, '+').replace(/_/g, '/')
    const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
    const escaped = Array.prototype.map.call(atob(padded), (c: string) => {
      return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2)
    }).join('')
    const parsed = JSON.parse(decodeURIComponent(escaped))
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}
