export type TabKey = 'debug' | 'apps' | 'errors'

export interface Settings {
  /** 留空表示当前站点同源（生产走 nginx，开发走 vite 代理） */
  baseUrl: string
  appId: string
  secret: string
  /** 管理面 JWT，仅保存在当前浏览器 */
  adminToken: string
  lastMethod: string
}

export type MethodCategory = 'podman' | 'modbus-read' | 'modbus-write'

export interface MethodField {
  name: string
  type: string
  required?: boolean
  note: string
}

export interface MethodDefinition {
  key: string
  label: string
  category: MethodCategory
  rpc: string
  description: string
  example: string
  fields: MethodField[]
  tips: string[]
  write?: boolean
}

export interface GatewayEnvelope<T = unknown> {
  code: number
  msg: string
  data?: T
}

export interface GatewayCallResponse {
  type: 'response'
  httpStatus: number
  code: number
  msg: string
  data?: unknown
  raw: string
  requestId: string
  durationMs: number
  sizeBytes: number
}

export interface GatewayCallFailure {
  type: 'network'
  message: string
  /** 网络/超时路径没有响应头，固定为空字符串，避免 undefined 传播。 */
  requestId: string
  durationMs: number
}

export type GatewayCallResult = GatewayCallResponse | GatewayCallFailure

export interface HistoryEntry {
  id: string
  at: string
  method: string
  tid: string
  dataText: string
  baseUrl: string
  appId: string
  type: GatewayCallResult['type']
  httpStatus: number
  code: number
  msg: string
  durationMs: number
  requestId: string
}

export interface AppRecord {
  appId: string
  name: string
  remark: string
  status: number
  lastUsedAt?: string
  createdAt: string
}

export interface AppListData {
  items: AppRecord[]
  total: number
  pageNum: number
  pageSize: number
}

export interface CreateAppData {
  appId: string
  name: string
  secret: string
  status: number
  createdAt: string
}

export interface ResetSecretData {
  appId: string
  secret: string
  oldSecretExpireAt: string
}

export interface AdminResult<T> {
  /** 信封存在时仅以 code===0 为 true；管理面非信封响应一律视为失败。 */
  ok: boolean
  httpStatus: number
  code: number
  msg: string
  data: T | null
  raw: string
  /** 服务端回显的 X-Request-Id，缺失时为 ''。 */
  requestId: string
}

export type ToastKind = 'success' | 'error' | 'info'

export interface ToastMessage {
  id: number
  kind: ToastKind
  text: string
}
