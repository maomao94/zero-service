import { Braces, Github, GitBranch, Video, Wifi } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

/** 自动检测周期（毫秒），状态条与文案共用 */
export const POLL_INTERVAL_MS = 15000

export interface ModuleDefinition {
  id: string
  name: string
  description: string
  /** 源码目录或站点标识，展示在卡片副标题 */
  source: string
  /** 访问位置标签，展示在卡片副标题（端口或路径） */
  endpoint: string
  /** 卡片强调色（图标与悬停描边），仅用于区分模块 */
  accent: string
  icon: LucideIcon
  url: string
  /** 同源健康检查地址，由 nginx / vite 代理到子系统 */
  healthUrl: string
  features: string[]
  group: 'service' | 'tool'
}

/** 外部站点入口：不参与健康检测，只做跳转 */
export interface ExternalRepo {
  id: string
  name: string
  repo: string
  description: string
  url: string
  icon: LucideIcon
  accent: string
  features: string[]
}

export interface EndpointDefinition {
  path: string
  title: string
  upstream: string
  note: string
}

// 子系统访问地址：优先使用 VITE_* 覆盖；未配置时按当前站点主机 + 端口推导，
// 适配 nginx 反代 / 远端访问（避免写死 localhost 导致打开与探活都失效）。
function systemUrl(override: string | undefined, port: number) {
  if (override) return override
  // 80/443 为默认端口，不拼接，避免出现 https://host:80 这类无效地址
  const suffix = port === 80 || port === 443 ? '' : `:${port}`
  return `${location.protocol}//${location.hostname}${suffix}`
}

// 路径型站点：openapi 控制台挂在 Live 站点的 /openapi/ 下，不能按工作台端口推导。
function pathUrl(override: string | undefined, path: string) {
  if (override) return override
  return `${liveSiteOrigin()}${path}`
}

function liveSiteOrigin(): string {
  try {
    return new URL(liveBaseUrl).origin
  } catch {
    return liveBaseUrl
  }
}

/** Live 站点基址：/v1、/openapi、/recordings 等端点都挂在该站点下 */
export const liveBaseUrl = systemUrl(import.meta.env.VITE_LIVE_URL, 80)

export const modules: ModuleDefinition[] = [
  {
    id: 'live',
    name: 'Live 视频会议',
    description: '多人视频会议与屏幕共享，包含会议管理、录制回放和 SIP 电话接入。',
    source: 'web/live',
    endpoint: ':80',
    accent: '#4f46e5',
    icon: Video,
    url: liveBaseUrl,
    healthUrl: '/health/live',
    features: ['视频通话', '屏幕共享', '录制回放', 'SIP 电话'],
    group: 'service',
  },
  {
    id: 'socketio',
    name: 'SocketIO 网关测试',
    description: 'Socket.IO 网关联调工具，覆盖用户域与设备域连接、房间和事件调试。',
    source: 'web/socketio',
    endpoint: ':8090',
    accent: '#0b7c7b',
    icon: Wifi,
    url: systemUrl(import.meta.env.VITE_SOCKETIO_URL, 8090),
    healthUrl: '/health/socketio',
    features: ['消息收发', '房间管理', '事件监听', '用户与设备域'],
    group: 'service',
  },
  {
    id: 'openapi',
    name: 'OpenAPI 测试控制台',
    description: 'app 开放 API 调试台，验证 Basic 鉴权、protojson 载荷与六位错误码。',
    source: 'web/openapi',
    endpoint: '/openapi/',
    accent: '#a16207',
    icon: Braces,
    url: pathUrl(import.meta.env.VITE_OPENAPI_URL, '/openapi/'),
    healthUrl: '/health/openapi',
    features: ['19 个 method', '应用与密钥', '错误码速查', '请求历史'],
    group: 'service',
  },
  {
    id: 'gitlab',
    name: 'GitLab 代码仓库',
    description: '公司代码仓库入口，托管 Zero Service 全部仓库、合并请求与流水线。',
    source: 'gitlab.localhost',
    endpoint: ':443',
    accent: '#c2410c',
    icon: GitBranch,
    url: import.meta.env.VITE_GITLAB_URL ?? 'https://gitlab.localhost/',
    healthUrl: '/health/gitlab',
    features: ['代码托管', '合并请求', 'CI 流水线'],
    group: 'tool',
  },
]

/** 项目公开仓库：工作台内的 GitHub 入口 */
export const externalRepos: ExternalRepo[] = [
  {
    id: 'github',
    name: 'GitHub 代码仓库',
    repo: 'maomao94 / zero-service',
    description: '项目公开源码仓库，查看提交历史、分支与 Issue，也可直接克隆。',
    url: import.meta.env.VITE_GITHUB_URL ?? 'https://github.com/maomao94/zero-service',
    icon: Github,
    accent: '#57606a',
    features: ['源码浏览', '提交历史', 'Issue / PR'],
  },
]

/** 同源端点速查：路径与上游来自 deploy/nginx 的站点配置 */
export const endpoints: EndpointDefinition[] = [
  {
    path: '/v1/',
    title: '开放 API 网关',
    upstream: 'opengtw · 11005',
    note: 'Podman / Modbus method 调用',
  },
  {
    path: '/live/v1/',
    title: '会议网关 HTTP API',
    upstream: 'livegtw · 11002',
    note: '会议、成员与录制管理',
  },
  {
    path: '/socket.io/',
    title: 'Socket.IO 长连接',
    upstream: 'socketgtw · 11003',
    note: '会议邀请与消息推送',
  },
  {
    path: '/recordings/',
    title: '会议录制回放',
    upstream: '本地只读挂载',
    note: '支持 Range 分段播放',
  },
  {
    path: '/openapi/',
    title: '开放 API 测试控制台',
    upstream: '静态站点',
    note: '调试 method 与应用凭据',
  },
]
