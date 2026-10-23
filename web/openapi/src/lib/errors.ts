export interface ErrorDoc {
  code: number
  http: number
  title: string
  description: string
  suggestion?: string
  folded?: boolean
  group: '成功' | '开放平台' | '参数' | '数据' | '认证' | '业务' | '基础设施' | '网关' | '系统'
}

/**
 * 与 docs/error-codes.md §4 全量码表保持同源。
 * v2 码位冻结，只允许追加；新增 BC=30（开放平台）错误码时同步更新此处。
 */
export const ERROR_DOCS: ErrorDoc[] = [
  {
    code: 0,
    http: 200,
    title: '成功',
    description: '请求已正常处理，data 为对应 method 的 protojson 结果。',
    group: '成功',
  },
  {
    code: 100101,
    http: 400,
    title: '参数错误',
    description: 'data 不是合法 protojson，或字段类型/取值不符合 proto 定义。',
    suggestion: '检查 JSON 字段名、类型与枚举名称，未知字段会被拒绝。',
    group: '参数',
  },
  {
    code: 100102,
    http: 400,
    title: '缺少必要参数',
    description: '缺少 method、appId、id 等必要字段。',
    suggestion: '补齐请求字段后重试。',
    group: '参数',
  },
  {
    code: 100103,
    http: 400,
    title: '参数不合法',
    description: '字段格式或范围不符合要求。',
    group: '参数',
  },
  {
    code: 100201,
    http: 500,
    title: '数据库错误',
    description: 'resource 库或下游数据库访问失败；开放面会折叠为 100999。',
    suggestion: '检查数据库连接与 SQL 执行日志。',
    folded: true,
    group: '数据',
  },
  {
    code: 100202,
    http: 404,
    title: '记录不存在',
    description: '指定的 Pod、镜像或应用不存在。',
    suggestion: '先用列表接口确认资源 ID。',
    group: '数据',
  },
  {
    code: 100203,
    http: 409,
    title: '记录已存在',
    description: '唯一约束冲突。',
    group: '数据',
  },
  {
    code: 100204,
    http: 409,
    title: '数据冲突',
    description: '并发更新或状态冲突。',
    group: '数据',
  },
  {
    code: 100301,
    http: 500,
    title: '缓存错误',
    description: 'Redis 等缓存读写失败；开放面会折叠为 100999。',
    suggestion: '检查缓存实例状态与网络连通性。',
    folded: true,
    group: '基础设施',
  },
  {
    code: 100302,
    http: 500,
    title: '缓存未命中',
    description: '查询的缓存 key 不存在，需要回源或重建缓存。',
    suggestion: '确认 key 是否已过期或从未写入，必要时回源后重试。',
    group: '基础设施',
  },
  {
    code: 100303,
    http: 503,
    title: '消息队列错误',
    description: '消息队列不可达或生产/消费失败；开放面会折叠为 100999。',
    suggestion: '确认 MQ 实例可用、topic 存在且消费者正常。',
    folded: true,
    group: '基础设施',
  },
  {
    code: 100401,
    http: 401,
    title: '未认证',
    description: 'JWT 缺失、无效或已过期。',
    suggestion: '重新获取管理面 JWT；确认签发密钥与 JwtAuth.AccessSecret 一致。',
    group: '认证',
  },
  {
    code: 100402,
    http: 403,
    title: '无权限访问',
    description: '当前身份没有执行该操作的权限。',
    group: '认证',
  },
  {
    code: 100501,
    http: 400,
    title: '业务处理失败',
    description: '下游业务逻辑拒绝处理。',
    group: '业务',
  },
  {
    code: 100502,
    http: 409,
    title: '业务状态不允许',
    description: '当前资源状态无法执行该操作。',
    group: '业务',
  },
  {
    code: 100503,
    http: 409,
    title: '重复操作',
    description: '相同操作已经执行过。',
    group: '业务',
  },
  {
    code: 100601,
    http: 503,
    title: '远程调用失败',
    description: '下游 gRPC 不可达；开放面会折叠为 100999。',
    suggestion: '确认 podengine / openapi 已启动且端口可达。',
    folded: true,
    group: '基础设施',
  },
  {
    code: 100602,
    http: 503,
    title: '第三方服务异常',
    description: '外部依赖返回异常；开放面会折叠为 100999。',
    folded: true,
    group: '基础设施',
  },
  {
    code: 100701,
    http: 429,
    title: '请求过于频繁',
    description: '触发平台限流。',
    suggestion: '降低请求频率后重试。',
    group: '基础设施',
  },
  {
    code: 100801,
    http: 404,
    title: '接口不存在',
    description: '访问的路径未在网关路由表中注册。',
    suggestion: '核对请求路径；开放面调用统一走 POST /v1/gateway。',
    group: '网关',
  },
  {
    code: 100802,
    http: 405,
    title: '请求方法不允许',
    description: '该路径不支持当前 HTTP 方法。',
    suggestion: '确认 GET/POST 与路由定义一致，开放面只接受 POST /v1/gateway。',
    group: '网关',
  },
  {
    code: 100997,
    http: 504,
    title: '系统超时',
    description: '下游调用超过 120s 或请求被取消。',
    suggestion: '缩短查询范围，检查 podengine 与 Docker daemon 是否响应缓慢。',
    group: '系统',
  },
  {
    code: 100998,
    http: 500,
    title: '系统内部错误',
    description: '服务内部异常；开放面会折叠为 100999 返回。',
    suggestion: '查看 opengtw / openapi 日志中的真实错误。',
    folded: true,
    group: '系统',
  },
  {
    code: 100999,
    http: 500,
    title: '未知错误 / 系统繁忙',
    description: '基础设施类错误折叠后的对外错误码，真实原因只写日志。',
    suggestion: '联系服务端排查数据库、缓存、消息队列或下游 RPC。',
    group: '系统',
  },
  {
    code: 130101,
    http: 403,
    title: '应用不存在或已停用',
    description: 'Basic 凭据中的应用 ID 不存在，或应用已被停用。',
    suggestion: '到「应用管理」确认应用状态；不存在与停用统一返回该错误。',
    group: '开放平台',
  },
  {
    code: 130102,
    http: 401,
    title: '密钥无效',
    description: '缺少 Basic 凭据，或 secret 与当前有效密钥不匹配。',
    suggestion: '确认 Authorization: Basic base64(appId:secret)；重置密钥后旧密钥保留 24h。',
    group: '开放平台',
  },
  {
    code: 130103,
    http: 404,
    title: '方法未开放',
    description: 'method 未在 facade/openapi 的路由表中注册。',
    suggestion: '当前仅开放 pod.list、pod.get、pod.stats、image.list。',
    group: '开放平台',
  },
  {
    code: 130104,
    http: 429,
    title: '请求超过限制',
    description: '请求频率超过开放平台限制。',
    suggestion: '降低请求频率后重试。',
    group: '开放平台',
  },
]

const ERROR_MAP = new Map(ERROR_DOCS.map((item) => [item.code, item]))

export function lookupError(code: number): ErrorDoc | undefined {
  return ERROR_MAP.get(code)
}

export function errorTone(code: number): 'ok' | 'warn' | 'error' {
  if (code === 0) return 'ok'
  const doc = lookupError(code)
  if (doc && doc.http >= 400 && doc.http < 500) return 'warn'
  return 'error'
}
