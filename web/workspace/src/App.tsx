import { Activity, ExternalLink, Layout, Monitor, RefreshCw, Video, Wifi, Zap } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'

interface SubSystem {
  id: string
  name: string
  description: string
  icon: React.ReactNode
  url: string
  port: number
  color: string
  features: string[]
}

type HealthStatus = 'checking' | 'online' | 'offline'

const STATUS_TEXT: Record<HealthStatus, string> = {
  checking: '检测中…',
  online: '运行中',
  offline: '未启动',
}

const subSystems: SubSystem[] = [
  {
    id: 'live',
    name: 'Live 视频会议',
    description: '实时视频会议系统，支持多人通话、屏幕共享与会议管理',
    icon: <Video size={32} />,
    url: 'http://localhost:5178',
    port: 5178,
    color: '#6366f1',
    features: ['视频通话', '屏幕共享', '会议管理', 'SIP 电话']
  },
  {
    id: 'socketio',
    name: 'SocketIO 网关测试',
    description: 'SocketIO 消息网关测试工具，支持用户域和设备域连接',
    icon: <Wifi size={32} />,
    url: 'http://localhost:5179',
    port: 5179,
    color: '#10b981',
    features: ['消息收发', '房间管理', '事件监听', '用户/设备域']
  }
]

async function ping(url: string, timeoutMs = 2500): Promise<boolean> {
  try {
    const controller = new AbortController()
    const timer = window.setTimeout(() => controller.abort(), timeoutMs)
    await fetch(url, { mode: 'no-cors', cache: 'no-store', signal: controller.signal })
    window.clearTimeout(timer)
    return true
  } catch {
    return false
  }
}

export default function App() {
  const [health, setHealth] = useState<Record<string, HealthStatus>>(() =>
    Object.fromEntries(subSystems.map((system) => [system.id, 'checking']))
  )
  const [lastCheck, setLastCheck] = useState<string>('')

  const checkAll = useCallback(async () => {
    await Promise.all(subSystems.map(async (system) => {
      const online = await ping(system.url)
      setHealth((current) => ({ ...current, [system.id]: online ? 'online' : 'offline' }))
    }))
    setLastCheck(new Date().toLocaleTimeString('zh-CN', { hour12: false }))
  }, [])

  useEffect(() => {
    checkAll()
    const timer = window.setInterval(checkAll, 15000)
    return () => window.clearInterval(timer)
  }, [checkAll])

  const openSystem = useCallback((system: SubSystem) => {
    window.open(system.url, '_blank', 'noopener')
  }, [])

  const openAll = () => {
    if (!window.confirm('将同时打开所有子系统页面，请允许浏览器弹窗')) return
    subSystems.forEach(openSystem)
  }

  return (
    <div className="workspace">
      {/* 顶部导航 */}
      <header className="workspace-header">
        <div className="header-left">
          <div className="logo">
            <Layout size={28} />
            <span>Zero 工作台</span>
          </div>
        </div>
        <div className="header-right">
          <div className="header-info">
            <Monitor size={16} />
            <span>开发环境</span>
          </div>
        </div>
      </header>

      {/* 主内容区 */}
      <main className="workspace-main">
        {/* 欢迎区域 */}
        <section className="welcome-section">
          <div className="welcome-content">
            <h1>欢迎使用 Zero 工作台</h1>
            <p>统一管理和访问各个子系统，提升开发和测试效率</p>
          </div>
          <div className="welcome-stats">
            <div className="stat-item">
              <Zap size={20} />
              <div>
                <span className="stat-value">{subSystems.length}</span>
                <span className="stat-label">可用系统</span>
              </div>
            </div>
            <div className="stat-item">
              <Activity size={20} />
              <div>
                <span className="stat-value">开发</span>
                <span className="stat-label">当前模式</span>
              </div>
            </div>
          </div>
        </section>

        {/* 子系统卡片 */}
        <section className="systems-section">
          <div className="section-heading-row">
            <h2>可用子系统</h2>
            {lastCheck && <span className="last-check">上次检测 {lastCheck}</span>}
          </div>
          <div className="systems-grid">
            {subSystems.map((system) => (
              <div
                key={system.id}
                className="system-card"
                role="button"
                tabIndex={0}
                onClick={() => openSystem(system)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault()
                    openSystem(system)
                  }
                }}
                style={{ '--accent-color': system.color } as React.CSSProperties}
              >
                <div className="card-header">
                  <div className="card-icon" style={{ backgroundColor: `${system.color}15`, color: system.color }}>
                    {system.icon}
                  </div>
                  <div className={`card-status ${health[system.id]}`}>
                    <span className="status-dot" />
                    <span>{STATUS_TEXT[health[system.id]]}</span>
                  </div>
                </div>

                <div className="card-content">
                  <h3>{system.name}</h3>
                  <p>{system.description}</p>

                  <div className="card-features">
                    {system.features.map((feature, index) => (
                      <span key={index} className="feature-tag">
                        {feature}
                      </span>
                    ))}
                  </div>
                </div>

                <div className="card-footer">
                  <div className="card-url">
                    <Monitor size={14} />
                    <span>localhost:{system.port}</span>
                  </div>
                  <button
                    type="button"
                    className="open-button"
                    style={{ backgroundColor: system.color }}
                    onClick={(event) => {
                      event.stopPropagation()
                      openSystem(system)
                    }}
                  >
                    <ExternalLink size={16} />
                    <span>打开</span>
                  </button>
                </div>
              </div>
            ))}
          </div>
        </section>

        {/* 快速操作 */}
        <section className="quick-actions">
          <h2>快速操作</h2>
          <div className="actions-grid">
            <button type="button" className="action-button" onClick={checkAll}>
              <Activity size={20} />
              <span>检测系统状态</span>
            </button>
            <button
              type="button"
              className="action-button"
              onClick={openAll}
            >
              <ExternalLink size={20} />
              <span>打开所有系统</span>
            </button>
            <button type="button" className="action-button" onClick={() => window.location.reload()}>
              <RefreshCw size={20} />
              <span>刷新工作台</span>
            </button>
          </div>
        </section>
      </main>

      {/* 底部信息 */}
      <footer className="workspace-footer">
        <p>Zero Service 工作台 v1.0.0 | 开发环境</p>
      </footer>
    </div>
  )
}
