import { useCallback, useEffect, useMemo, useRef, useState, type MouseEvent as ReactMouseEvent } from 'react'
import { io, type Socket } from 'socket.io-client'
import { 
  Plug, Unplug, Send, Radio, Globe, DoorOpen, DoorClosed, 
  List, Plus, Trash2, Settings, Zap, 
  MessageSquare, Users, Server, Download,
  Copy, Check, AlertCircle,
  Info, Wifi, WifiOff, Loader2,
  Eye, Terminal, ChevronDown, ChevronRight,
  CheckCircle2, Circle, Hash, RefreshCw, KeyRound
} from 'lucide-react'
import { decodeJwtClaims } from './lib/token'
import { buildClaims, signJwt, type AuthType } from './lib/jwt'

type ConnectionState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting'
type LogType = 'system' | 'connect' | 'disconnect' | 'message' | 'receive' | 'error'
type TabType = 'config' | 'message' | 'room' | 'event'

interface LogEntry {
  id: string
  type: LogType
  message: string
  timestamp: string
  detail?: string
}

interface RegisteredEvent {
  name: string
  registered: boolean
}

interface RoomsPage {
  total: number
  page: number
  pageSize: number
  totalPages: number
  rooms: string[]
}

interface EventStat {
  count: number
  lastAt: string
}

function parseRoomsAck(ack: unknown): RoomsPage | null {
  try {
    const raw = typeof ack === 'string' ? JSON.parse(ack) : ack
    const payload = (raw as { payload?: unknown } | null)?.payload
    if (!payload || typeof payload !== 'object') return null
    const data = payload as Partial<RoomsPage>
    if (!Array.isArray(data.rooms)) return null
    return {
      total: Number(data.total) || 0,
      page: Number(data.page) || 1,
      pageSize: Number(data.pageSize) || 20,
      totalPages: Number(data.totalPages) || 0,
      rooms: data.rooms.map((room) => String(room)),
    }
  } catch {
    return null
  }
}

// 生产环境留空时走当前站点同源，由 nginx 的 8090 站点反代 /socket.io/ 到 socketgtw。
const DEFAULT_SERVER_URL = import.meta.env.VITE_SOCKET_URL || location.origin

function generateReqId(): string {
  return `req-${Date.now()}-${Math.random().toString(36).slice(2, 11)}`
}

function formatTime(date: Date): string {
  return date.toLocaleTimeString('zh-CN', { hour12: false })
}

export default function App() {
  const [socket, setSocket] = useState<Socket | null>(null)
  const [connectionState, setConnectionState] = useState<ConnectionState>('disconnected')
  const [socketId, setSocketId] = useState<string>('')
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [registeredEvents, setRegisteredEvents] = useState<RegisteredEvent[]>([])
  const [autoScroll, setAutoScroll] = useState(true)
  const [logFilter, setLogFilter] = useState<LogType | 'all'>('all')
  const [activeTab, setActiveTab] = useState<TabType>('config')
  const [copied, setCopied] = useState<string | null>(null)
  const [consoleOpen, setConsoleOpen] = useState(true)
  const [consoleHeight, setConsoleHeight] = useState(260)
  const [expandedLogs, setExpandedLogs] = useState<Set<string>>(new Set())
  const [unreadCount, setUnreadCount] = useState(0)
  const consoleBodyRef = useRef<HTMLDivElement>(null)
  const lastLogIdRef = useRef<string | null>(null)

  // 配置状态
  const [serverUrl, setServerUrl] = useState(DEFAULT_SERVER_URL)
  const [authToken, setAuthToken] = useState('')
  
  // 消息状态
  const [eventName, setEventName] = useState('custom_event')
  const [payload, setPayload] = useState('Hello SocketIO!')
  const [broadcastRoom, setBroadcastRoom] = useState('test-room')
  
  // 房间状态
  const [roomName, setRoomName] = useState('test-room')
  
  // 事件状态
  const [newEventName, setNewEventName] = useState('')
  const [tokenAuthType, setTokenAuthType] = useState<AuthType>('user')
  const [tokenId, setTokenId] = useState('user-10001')
  const [tokenName, setTokenName] = useState('测试用户')
  const [tokenDept, setTokenDept] = useState('dept-001')
  const [tokenExpire, setTokenExpire] = useState(3600)
  const [tokenSecret, setTokenSecret] = useState(import.meta.env.VITE_JWT_SECRET || '629c6233-1a76-471b-bd25-b87208762219')
  const [generatingToken, setGeneratingToken] = useState(false)
  const [roomsPage, setRoomsPage] = useState<RoomsPage | null>(null)
  const [roomsLoading, setRoomsLoading] = useState(false)
  const [eventStats, setEventStats] = useState<Record<string, EventStat>>({})

  // 解析当前 Token 的 claims
  const tokenClaims = useMemo(() => decodeJwtClaims(authToken), [authToken])

  // 从 localStorage 加载已注册的事件
  useEffect(() => {
    try {
      const savedEvents = localStorage.getItem('socketio_test_events')
      if (savedEvents) {
        const events = JSON.parse(savedEvents) as string[]
        setRegisteredEvents(events.map(name => ({ name, registered: false })))
      }
    } catch (error) {
      console.error('Failed to load registered events from localStorage:', error)
    }
  }, [])

  // 保存已注册的事件到 localStorage
  const saveRegisteredEvents = useCallback((events: RegisteredEvent[]) => {
    try {
      const eventNames = events.map(e => e.name)
      localStorage.setItem('socketio_test_events', JSON.stringify(eventNames))
    } catch (error) {
      console.error('Failed to save registered events to localStorage:', error)
    }
  }, [])

  // 添加日志
  const addLog = useCallback((type: LogType, message: string, detail?: string) => {
    const newLog: LogEntry = {
      id: `log-${Date.now()}-${Math.random().toString(36).slice(2, 11)}`,
      type,
      message,
      timestamp: formatTime(new Date()),
      detail
    }
    setLogs(prev => {
      const newLogs = [...prev, newLog]
      if (newLogs.length > 200) {
        return newLogs.slice(-200)
      }
      return newLogs
    })
  }, [])

  // 清空日志
  const clearLogs = useCallback(() => {
    setLogs([])
    addLog('system', '日志已清空')
  }, [addLog])

  const generateLocalToken = useCallback(async () => {
    if (!tokenId.trim()) {
      addLog('error', '请输入用户/设备 ID')
      return
    }
    setGeneratingToken(true)
    try {
      const token = await signJwt(
        buildClaims({
          authType: tokenAuthType,
          id: tokenId.trim(),
          name: tokenName.trim() || undefined,
          dept: tokenAuthType === 'user' ? tokenDept.trim() || undefined : undefined,
          expireSeconds: Math.max(60, tokenExpire || 3600),
        }),
        tokenSecret.trim(),
      )
      setAuthToken(token)
      addLog('system', `已在本地生成${tokenAuthType === 'user' ? '用户' : '设备'} Token`)
    } catch (error) {
      addLog('error', (error as Error).message)
    } finally {
      setGeneratingToken(false)
    }
  }, [tokenAuthType, tokenId, tokenName, tokenDept, tokenExpire, tokenSecret, addLog])

  const recordReceive = useCallback((name: string) => {
    const at = formatTime(new Date())
    setEventStats((prev) => ({
      ...prev,
      [name]: { count: (prev[name]?.count ?? 0) + 1, lastAt: at },
    }))
  }, [])

  // 注册自定义事件监听器
  const registerCustomEventListener = useCallback((eventName: string) => {
    if (!socket) return

    const handler = (data: unknown) => {
      addLog('receive', `收到 ${eventName}`, JSON.stringify(data, null, 2))
      recordReceive(eventName)
    }

    socket.on(eventName, handler)
    setRegisteredEvents(prev => {
      const exists = prev.some(e => e.name === eventName)
      if (exists) return prev.map(e => e.name === eventName ? { ...e, registered: true } : e)
      const newEvents = [...prev, { name: eventName, registered: true }]
      saveRegisteredEvents(newEvents)
      return newEvents
    })
    addLog('system', `注册事件监听: ${eventName}`)
  }, [socket, addLog, saveRegisteredEvents, recordReceive])

  // 移除自定义事件监听器
  const removeCustomEventListener = useCallback((eventName: string) => {
    if (!socket) return

    socket.off(eventName)
    setRegisteredEvents(prev => {
      const newEvents = prev.filter(e => e.name !== eventName)
      saveRegisteredEvents(newEvents)
      return newEvents
    })
    addLog('system', `移除事件监听: ${eventName}`)
  }, [socket, addLog, saveRegisteredEvents])

  // 添加新的事件监听
  const addNewEvent = useCallback(() => {
    if (!newEventName.trim()) return
    registerCustomEventListener(newEventName.trim())
    setNewEventName('')
  }, [newEventName, registerCustomEventListener])

  // 连接服务器
  const connect = useCallback(() => {
    if (connectionState === 'connected' || connectionState === 'connecting') {
      return
    }

    addLog('system', `正在连接到 ${serverUrl}`)
    setConnectionState('connecting')

    const socketOptions: Record<string, unknown> = {
      transports: ['websocket', 'polling'],
      reconnection: true,
      reconnectionAttempts: 5,
      reconnectionDelay: 1000,
    }

    // 构建 auth 对象
    const auth: Record<string, string> = {}
    if (authToken) {
      auth.token = authToken
    }

    if (Object.keys(auth).length > 0) {
      socketOptions.auth = auth
    }

    if (authToken) {
      addLog('system', `使用 Token 认证`)
    }

    const newSocket = io(serverUrl, socketOptions)

    newSocket.io.on('ping', () => {
      addLog('system', '收到服务器 ping')
    })

    newSocket.on('connect', () => {
      setConnectionState('connected')
      setSocketId(newSocket.id || '')
      addLog('connect', `连接成功，Socket ID: ${newSocket.id}`)

      // 直接在本次 socket 实例上注册默认事件与已保存事件（避免闭包旧实例）
      const eventNames = new Set<string>()
      if (eventName) {
        eventNames.add(eventName)
      }
      registeredEvents.forEach(event => eventNames.add(event.name))
      eventNames.forEach(name => {
        newSocket.on(name, (data: unknown) => {
          addLog('receive', `收到 ${name}`, JSON.stringify(data, null, 2))
          recordReceive(name)
        })
      })
      if (eventNames.size > 0) {
        const nextEvents = Array.from(eventNames).map(name => ({ name, registered: true }))
        setRegisteredEvents(nextEvents)
        saveRegisteredEvents(nextEvents)
      }
    })

    newSocket.on('reconnect_attempt', () => {
      setConnectionState('reconnecting')
      addLog('system', '尝试重连...')
    })

    newSocket.on('disconnect', (reason) => {
      setConnectionState('disconnected')
      setSocketId('')
      addLog('disconnect', `连接断开，原因: ${reason}`)
    })

    newSocket.on('connect_error', (error) => {
      addLog('error', `连接错误: ${error.message}`)
    })

    newSocket.on('__down__', (data: unknown) => {
      addLog('receive', '收到 __down__ 事件', JSON.stringify(data, null, 2))
      recordReceive('__down__')
    })

    newSocket.on('__stat_down__', (data: unknown) => {
      addLog('receive', '收到 __stat_down__ 事件', JSON.stringify(data, null, 2))
      recordReceive('__stat_down__')
    })

    setSocket(newSocket)
  }, [serverUrl, authToken, eventName, addLog, registeredEvents, saveRegisteredEvents, connectionState, recordReceive])

  // 断开连接
  const disconnect = useCallback(() => {
    if (socket) {
      socket.disconnect()
      setSocket(null)
    }
  }, [socket])

  // 发送上行消息
  const sendUp = useCallback(() => {
    if (!socket || connectionState !== 'connected') {
      addLog('error', '请先连接服务器')
      return
    }

    const reqId = generateReqId()
    socket.emit('__up__', {
      payload,
      reqId
    }, (ack: unknown) => {
      addLog('message', '收到 ACK 回调', JSON.stringify(ack, null, 2))
    })

    addLog('message', '发送 __up__ 事件', JSON.stringify({ payload, reqId }, null, 2))
  }, [socket, connectionState, payload, addLog])

  // 发送房间广播
  const sendRoom = useCallback(() => {
    if (!socket || connectionState !== 'connected') {
      addLog('error', '请先连接服务器')
      return
    }

    const reqId = generateReqId()
    socket.emit('__room_broadcast_up__', {
      reqId,
      room: broadcastRoom,
      event: eventName,
      payload
    }, (ack: unknown) => {
      addLog('message', '收到房间广播 ACK', JSON.stringify(ack, null, 2))
    })

    addLog('message', `发送房间广播到 ${broadcastRoom}`, JSON.stringify({ payload, reqId, event: eventName }, null, 2))
  }, [socket, connectionState, broadcastRoom, eventName, payload, addLog])

  // 发送全局广播
  const sendGlobal = useCallback(() => {
    if (!socket || connectionState !== 'connected') {
      addLog('error', '请先连接服务器')
      return
    }

    const reqId = generateReqId()
    socket.emit('__global_broadcast_up__', {
      reqId,
      event: eventName,
      payload
    }, (ack: unknown) => {
      addLog('message', '收到全局广播 ACK', JSON.stringify(ack, null, 2))
    })

    addLog('message', '发送全局广播', JSON.stringify({ payload, reqId, event: eventName }, null, 2))
  }, [socket, connectionState, eventName, payload, addLog])

  // 加入房间
  const joinRoom = useCallback(() => {
    if (!socket || connectionState !== 'connected') {
      addLog('error', '请先连接服务器')
      return
    }

    const reqId = generateReqId()
    socket.emit('__join_room_up__', {
      reqId,
      room: roomName
    }, (ack: unknown) => {
      addLog('message', '加入房间 ACK', JSON.stringify(ack, null, 2))
    })

    addLog('message', `正在加入房间: ${roomName}`)
  }, [socket, connectionState, roomName, addLog])

  // 离开房间
  const leaveRoom = useCallback(() => {
    if (!socket || connectionState !== 'connected') {
      addLog('error', '请先连接服务器')
      return
    }

    const reqId = generateReqId()
    socket.emit('__leave_room_up__', {
      reqId,
      room: roomName
    }, (ack: unknown) => {
      addLog('message', '离开房间 ACK', JSON.stringify(ack, null, 2))
    })

    addLog('message', `正在离开房间: ${roomName}`)
  }, [socket, connectionState, roomName, addLog])

  // 查询房间列表
  const queryRoomsPage = useCallback(() => {
    if (!socket || connectionState !== 'connected') {
      addLog('error', '请先连接服务器')
      return
    }

    const reqId = generateReqId()
    setRoomsLoading(true)
    const timer = window.setTimeout(() => setRoomsLoading(false), 5000)
    socket.emit('__rooms_page_up__', {
      reqId,
      page: 1,
      pageSize: 20
    }, (ack: unknown) => {
      window.clearTimeout(timer)
      setRoomsLoading(false)
      const parsed = parseRoomsAck(ack)
      if (parsed) setRoomsPage(parsed)
      addLog('message', parsed ? `房间列表：${parsed.total} 个` : '房间列表查询结果', JSON.stringify(ack, null, 2))
    })

    addLog('message', '正在查询房间列表')
  }, [socket, connectionState, addLog])

  // 自动滚动到底部
  useEffect(() => {
    if (autoScroll && consoleOpen && consoleBodyRef.current) {
      consoleBodyRef.current.scrollTop = consoleBodyRef.current.scrollHeight
    }
  }, [logs, logFilter, autoScroll, consoleOpen])

  // 收起时累计未读日志数
  useEffect(() => {
    const lastId = logs.length > 0 ? logs[logs.length - 1].id : null
    if (consoleOpen) {
      setUnreadCount(0)
    } else if (lastId && lastId !== lastLogIdRef.current) {
      setUnreadCount(prev => prev + 1)
    }
    lastLogIdRef.current = lastId
  }, [logs, consoleOpen])

  // 拖拽调整控制台高度
  const startResize = useCallback((e: ReactMouseEvent) => {
    e.preventDefault()
    const startY = e.clientY
    const startHeight = consoleHeight
    const onMove = (ev: MouseEvent) => {
      const next = Math.min(Math.max(startHeight + startY - ev.clientY, 120), window.innerHeight * 0.8)
      setConsoleHeight(next)
    }
    const onUp = () => {
      window.removeEventListener('mousemove', onMove)
      window.removeEventListener('mouseup', onUp)
      document.body.style.userSelect = ''
    }
    document.body.style.userSelect = 'none'
    window.addEventListener('mousemove', onMove)
    window.addEventListener('mouseup', onUp)
  }, [consoleHeight])

  // 展开/收起日志详情
  const toggleLogDetail = useCallback((id: string) => {
    setExpandedLogs(prev => {
      const next = new Set(prev)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }, [])

  // 复制到剪贴板
  const copyToClipboard = useCallback((text: string, id: string) => {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(id)
      setTimeout(() => setCopied(null), 2000)
    })
  }, [])

  // 过滤日志
  const filteredLogs = logFilter === 'all' ? logs : logs.filter(log => log.type === logFilter)

  // 导出日志
  const exportLogs = useCallback(() => {
    const logText = filteredLogs.map(log => 
      `[${log.timestamp}] [${log.type.toUpperCase()}] ${log.message}${log.detail ? '\n' + log.detail : ''}`
    ).join('\n')
    
    const blob = new Blob([logText], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `socketio-logs-${new Date().toISOString().slice(0, 10)}.txt`
    a.click()
    URL.revokeObjectURL(url)
  }, [filteredLogs])

  // 获取连接状态颜色
  const getConnectionColor = () => {
    switch (connectionState) {
      case 'connected': return '#10b981'
      case 'connecting': return '#f59e0b'
      case 'reconnecting': return '#f59e0b'
      case 'disconnected': return '#ef4444'
    }
  }

  // 获取连接状态文本
  const getConnectionText = () => {
    switch (connectionState) {
      case 'connected': return '已连接'
      case 'connecting': return '连接中...'
      case 'reconnecting': return '重连中...'
      case 'disconnected': return '未连接'
    }
  }

  // 获取日志类型图标
  const getLogIcon = (type: LogType) => {
    switch (type) {
      case 'system': return <Info size={14} />
      case 'connect': return <Wifi size={14} />
      case 'disconnect': return <WifiOff size={14} />
      case 'message': return <Send size={14} />
      case 'receive': return <MessageSquare size={14} />
      case 'error': return <AlertCircle size={14} />
    }
  }

  // 获取日志类型颜色
  const getLogColor = (type: LogType) => {
    switch (type) {
      case 'system': return '#6366f1'
      case 'connect': return '#10b981'
      case 'disconnect': return '#f59e0b'
      case 'message': return '#3b82f6'
      case 'receive': return '#8b5cf6'
      case 'error': return '#ef4444'
    }
  }


  const TAB_META = {
    config: {
      label: '连接配置',
      title: '连接配置',
      description: '填写网关地址与 JWT，建立或断开 Socket.IO 会话。',
      icon: Settings,
    },
    message: {
      label: '消息发送',
      title: '消息发送',
      description: '发送上行消息、房间广播与全局广播，ACK 会写入底部控制台。',
      icon: Send,
    },
    room: {
      label: '房间管理',
      title: '房间管理',
      description: '加入、离开房间，或拉取当前房间列表。',
      icon: Users,
    },
    event: {
      label: '事件监听',
      title: '事件监听',
      description: '注册自定义事件，收到的载荷会写入底部控制台。',
      icon: Radio,
    },
  } as const

  const LOG_LABELS: Record<LogType | 'all', string> = {
    all: '全部',
    system: '系统',
    connect: '连接',
    disconnect: '断开',
    message: '发送',
    receive: '接收',
    error: '错误',
  }

  const connected = connectionState === 'connected'
  const eventStatEntries = Object.entries(eventStats).sort((a, b) => b[1].count - a[1].count)

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="brand">
          <span className="brand-mark" aria-hidden="true">
            <Zap size={18} />
          </span>
          <div className="brand-copy">
            <strong>SocketIO 网关</strong>
            <span>调试控制台 · socketgtw</span>
          </div>
        </div>

        <div className="topbar-mid">
          <span className="endpoint" title={serverUrl}>
            {serverUrl}
          </span>
          <span className="conn-pill" data-state={connectionState}>
            <i style={{ background: getConnectionColor() }} aria-hidden="true" />
            {getConnectionText()}
          </span>
          {socketId && (
            <span className="socket-id" title={socketId}>
              {socketId}
            </span>
          )}
        </div>

        <div className="topbar-actions">
          <button
            type="button"
            className="btn btn-connect"
            data-connected={connected}
            onClick={connected ? disconnect : connect}
            disabled={connectionState === 'connecting'}
          >
            {connected ? (
              <>
                <Unplug size={15} />
                断开
              </>
            ) : connectionState === 'connecting' ? (
              <>
                <Loader2 size={15} className="spin" />
                连接中
              </>
            ) : (
              <>
                <Plug size={15} />
                连接
              </>
            )}
          </button>
          <button
            type="button"
            className="icon-btn topbar-console"
            onClick={() => setConsoleOpen((prev) => !prev)}
            aria-label={consoleOpen ? '收起控制台' : '展开控制台'}
            aria-pressed={consoleOpen}
          >
            <Terminal size={16} />
          </button>
        </div>
      </header>

      <div className="app-body">
        <aside className="rail">
          <p className="rail-label">工作区</p>
          <nav className="rail-nav" aria-label="调试工作区">
            {(Object.keys(TAB_META) as TabType[]).map((tab) => {
              const meta = TAB_META[tab]
              const Icon = meta.icon
              return (
                <button
                  key={tab}
                  type="button"
                  className="rail-item"
                  data-active={activeTab === tab}
                  aria-current={activeTab === tab ? 'page' : undefined}
                  onClick={() => setActiveTab(tab)}
                >
                  <Icon size={16} />
                  <span>{meta.label}</span>
                </button>
              )
            })}
          </nav>

          <div className="rail-block">
            <p className="rail-label">当前会话</p>
            <dl className="rail-facts">
              <div>
                <dt>Socket ID</dt>
                <dd title={socketId || undefined}>{socketId || '—'}</dd>
              </div>
              <div>
                <dt>Token</dt>
                <dd>{authToken ? `${Object.keys(tokenClaims).length} claims` : '未设置'}</dd>
              </div>
              <div>
                <dt>事件</dt>
                <dd>{registeredEvents.length}</dd>
              </div>
            </dl>
          </div>

          <div className="rail-foot">
            <button
              type="button"
              className="rail-console"
              data-open={consoleOpen}
              onClick={() => setConsoleOpen((prev) => !prev)}
            >
              <Terminal size={15} />
              <span>控制台</span>
              <span className="rail-console-count">{logs.length}</span>
            </button>
          </div>
        </aside>

        <main className="workspace">
          <header className="workspace-head">
            <div>
              <h1>{TAB_META[activeTab].title}</h1>
              <p>{TAB_META[activeTab].description}</p>
            </div>
          </header>

          <div className="workspace-scroll">
            {activeTab === 'config' && (
              <>
              <div className="grid-2">
                <section className="card">
                  <header className="card-head">
                    <h2>连接参数</h2>
                    <p>连接成功后地址与 Token 会被锁定。</p>
                  </header>
                  <div className="card-body">
                    <label className="field">
                      <span className="field-label">
                        <Server size={13} />
                        服务器地址
                      </span>
                      <input
                        type="text"
                        value={serverUrl}
                        onChange={(e) => setServerUrl(e.target.value)}
                        placeholder="http://127.0.0.1:11003"
                        disabled={connected}
                      />
                      <small>生产环境由 nginx 将 /socket.io/ 反代到 socketgtw。</small>
                    </label>

                    <label className="field">
                      <span className="field-label">
                        <Zap size={13} />
                        Authorization Token
                      </span>
                      <input
                        type="password"
                        value={authToken}
                        onChange={(e) => setAuthToken(e.target.value)}
                        placeholder="粘贴 JWT Token"
                        disabled={connected}
                      />
                      <small>可直接粘贴，或在下方「本地签发 Token」生成；不再依赖 live 子系统。</small>
                    </label>

                    <div className="form-actions">
                      <button
                        type="button"
                        className="btn btn-primary"
                        onClick={connected ? disconnect : connect}
                        disabled={connectionState === 'connecting'}
                      >
                        {connected ? (
                          <>
                            <Unplug size={15} />
                            断开连接
                          </>
                        ) : connectionState === 'connecting' ? (
                          <>
                            <Loader2 size={15} className="spin" />
                            连接中…
                          </>
                        ) : (
                          <>
                            <Plug size={15} />
                            连接服务器
                          </>
                        )}
                      </button>
                      <button
                        type="button"
                        className="btn btn-ghost"
                        onClick={() => {
                          setAuthToken('')
                          setServerUrl(DEFAULT_SERVER_URL)
                        }}
                        disabled={connected}
                      >
                        重置
                      </button>
                    </div>

                    <div className="card-divider" />
                    <dl className="event-facts">
                      <div>
                        <dt>传输</dt>
                        <dd>WebSocket · Polling 回退</dd>
                      </div>
                      <div>
                        <dt>重连</dt>
                        <dd>5 次 · 1s 间隔</dd>
                      </div>
                      <div>
                        <dt>认证</dt>
                        <dd>JWT（可选）</dd>
                      </div>
                    </dl>
                  </div>
                </section>

                <div className="stack">
                <section className="card">
                  <header className="card-head">
                    <h2>Token Claims</h2>
                    <p>{authToken ? `${Object.keys(tokenClaims).length} 个字段` : '尚未粘贴 Token'}</p>
                  </header>
                  <div className="card-body">
                    {authToken ? (
                      Object.keys(tokenClaims).length > 0 ? (
                        <dl className="claims-list">
                          {Object.entries(tokenClaims).map(([key, value]) => (
                            <div key={key} className="claim-item">
                              <dt>{key}</dt>
                              <dd>{String(value)}</dd>
                            </div>
                          ))}
                        </dl>
                      ) : (
                        <div className="empty">
                          <Eye size={18} />
                          <p>Token 无法解析</p>
                          <small>请确认粘贴的是完整的 JWT。</small>
                        </div>
                      )
                    ) : (
                      <div className="empty">
                        <Eye size={18} />
                        <p>粘贴 Token 后解析 claims</p>
                        <small>claims 只用于本地展示，不会上传。</small>
                      </div>
                    )}
                  </div>
                </section>

                <section className="card card--soft">
                  <header className="card-head">
                    <h2>连接检查</h2>
                    <p>{connected ? '会话已建立' : '按顺序确认'}</p>
                  </header>
                  <div className="card-body">
                    <ul className="checklist">
                      {[
                        {
                          label: '网关地址',
                          done: serverUrl.trim().length > 0,
                          value: serverUrl.trim() ? '已填写' : '待填写',
                        },
                        {
                          label: 'JWT 认证',
                          done: Boolean(authToken),
                          value: authToken ? '已粘贴' : '可选',
                        },
                        {
                          label: 'Socket 会话',
                          done: connected,
                          value: connected ? '已连接' : '未连接',
                        },
                        {
                          label: '事件监听',
                          done: registeredEvents.length > 0,
                          value: registeredEvents.length > 0 ? `${registeredEvents.length} 个` : '未注册',
                        },
                      ].map((item) => (
                        <li key={item.label} data-done={item.done}>
                          {item.done ? <CheckCircle2 size={14} /> : <Circle size={14} />}
                          <span className="check-label">{item.label}</span>
                          <span className="check-value">{item.value}</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                </section>
                </div>
              </div>
              <section className="card card--soft token-card">
                <header className="card-head">
                  <h2>本地签发 Token</h2>
                  <p>HS256 · 不请求后端</p>
                </header>
                <div className="card-body">
                  <div className="token-form">
                    <div className="token-form-row">
                      <div className="token-type-tabs">
                        <button
                          type="button"
                          data-active={tokenAuthType === 'user'}
                          onClick={() => {
                            setTokenAuthType('user')
                            setTokenId('user-10001')
                            setTokenName('测试用户')
                          }}
                        >
                          用户
                        </button>
                        <button
                          type="button"
                          data-active={tokenAuthType === 'device'}
                          onClick={() => {
                            setTokenAuthType('device')
                            setTokenId('device-001')
                            setTokenName('测试设备')
                          }}
                        >
                          设备
                        </button>
                      </div>
                      <label className="field">
                        <span className="field-label">ID</span>
                        <input
                          type="text"
                          value={tokenId}
                          onChange={(e) => setTokenId(e.target.value)}
                          placeholder={tokenAuthType === 'user' ? 'user-10001' : 'device-001'}
                        />
                      </label>
                      <label className="field">
                        <span className="field-label">名称</span>
                        <input
                          type="text"
                          value={tokenName}
                          onChange={(e) => setTokenName(e.target.value)}
                          placeholder={tokenAuthType === 'user' ? '测试用户' : '测试设备'}
                        />
                      </label>
                      {tokenAuthType === 'user' && (
                        <label className="field">
                          <span className="field-label">部门</span>
                          <input
                            type="text"
                            value={tokenDept}
                            onChange={(e) => setTokenDept(e.target.value)}
                            placeholder="dept-001"
                          />
                        </label>
                      )}
                      <label className="field token-expire">
                        <span className="field-label">有效期（秒）</span>
                        <input
                          type="number"
                          min={60}
                          value={tokenExpire}
                          onChange={(e) => setTokenExpire(Number(e.target.value))}
                        />
                      </label>
                    </div>

                    <div className="token-form-row">
                      <label className="field token-secret">
                        <span className="field-label">签名密钥 AccessSecret</span>
                        <input
                          type="password"
                          value={tokenSecret}
                          onChange={(e) => setTokenSecret(e.target.value)}
                          placeholder="与 livegtw / socketgtw 配置一致"
                        />
                        <small>本地 HS256 签名，不请求后端；密钥只保存在当前页面。</small>
                      </label>
                      <button
                        type="button"
                        className="btn btn-primary"
                        onClick={() => void generateLocalToken()}
                        disabled={generatingToken}
                      >
                        {generatingToken ? <Loader2 size={14} className="spin" /> : <KeyRound size={14} />}
                        生成并填入
                      </button>
                    </div>
                  </div>
                </div>
              </section>
              </>
            )}

            {activeTab === 'message' && (
              <div className="grid-2 grid-2--wide">
                <section className="card">
                  <header className="card-head">
                    <h2>消息内容</h2>
                    <p>上行消息直接发给网关，广播按房间或全局分发。</p>
                  </header>
                  <div className="card-body">
                    <label className="field">
                      <span className="field-label">
                        <Radio size={13} />
                        事件名称
                      </span>
                      <input
                        type="text"
                        value={eventName}
                        onChange={(e) => setEventName(e.target.value)}
                        placeholder="custom_event"
                      />
                    </label>

                    <label className="field">
                      <span className="field-label">
                        <MessageSquare size={13} />
                        消息内容
                      </span>
                      <textarea
                        value={payload}
                        onChange={(e) => setPayload(e.target.value)}
                        onKeyDown={(e) => {
                          if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                            e.preventDefault()
                            sendUp()
                          }
                        }}
                        placeholder="输入要发送的消息内容"
                        rows={6}
                      />
                      <small>Ctrl / ⌘ + Enter 发送上行消息 · {payload.length} 字符</small>
                    </label>

                    <div className="preset-row">
                      <button type="button" onClick={() => setPayload('Hello SocketIO!')}>
                        纯文本
                      </button>
                      <button type="button" onClick={() => setPayload('{"type":"ping"}')}>
                        JSON
                      </button>
                    </div>

                    <div className="form-actions">
                      <button
                        type="button"
                        className="btn btn-primary"
                        onClick={sendUp}
                        disabled={!connected}
                      >
                        <Send size={15} />
                        发送上行消息
                      </button>
                    </div>

                    <div className="card-divider" />

                    <label className="field">
                      <span className="field-label">
                        <Users size={13} />
                        目标房间
                      </span>
                      <input
                        type="text"
                        value={broadcastRoom}
                        onChange={(e) => setBroadcastRoom(e.target.value)}
                        placeholder="test-room"
                      />
                    </label>

                    <div className="form-actions">
                      <button
                        type="button"
                        className="btn btn-ghost"
                        onClick={sendRoom}
                        disabled={!connected}
                      >
                        <Radio size={15} />
                        房间广播
                      </button>
                      <button
                        type="button"
                        className="btn btn-ghost"
                        onClick={sendGlobal}
                        disabled={!connected}
                      >
                        <Globe size={15} />
                        全局广播
                      </button>
                    </div>
                  </div>
                </section>

                <aside className="stack">
                  <section className="card card--soft">
                    <header className="card-head">
                      <h2>发送说明</h2>
                    </header>
                    <div className="card-body">
                      <dl className="event-facts">
                        <div>
                          <dt>上行</dt>
                          <dd>__up__</dd>
                        </div>
                        <div>
                          <dt>房间广播</dt>
                          <dd>__room_broadcast_up__</dd>
                        </div>
                        <div>
                          <dt>全局广播</dt>
                          <dd>__global_broadcast_up__</dd>
                        </div>
                      </dl>
                      <p className="hint">ACK 回调与接收事件都会写入底部控制台。</p>
                    </div>
                  </section>

                  <section className="card card--soft">
                    <header className="card-head">
                      <h2>当前载荷</h2>
                      <p>{payload.length} 字符</p>
                    </header>
                    <div className="card-body">
                      <pre className="payload-preview">{payload || '（空）'}</pre>
                    </div>
                  </section>
                </aside>
              </div>
            )}

            {activeTab === 'room' && (
              <>
              <div className="grid-2">
                <section className="card">
                  <header className="card-head">
                    <h2>房间操作</h2>
                    <p>加入或离开指定房间，也可以拉取房间列表。</p>
                  </header>
                  <div className="card-body">
                    <label className="field">
                      <span className="field-label">
                        <DoorOpen size={13} />
                        房间名称
                      </span>
                      <input
                        type="text"
                        value={roomName}
                        onChange={(e) => setRoomName(e.target.value)}
                        placeholder="test-room"
                      />
                    </label>

                    <div className="form-actions">
                      <button
                        type="button"
                        className="btn btn-primary"
                        onClick={joinRoom}
                        disabled={!connected}
                      >
                        <DoorOpen size={15} />
                        加入房间
                      </button>
                      <button
                        type="button"
                        className="btn btn-ghost"
                        onClick={leaveRoom}
                        disabled={!connected}
                      >
                        <DoorClosed size={15} />
                        离开房间
                      </button>
                      <button
                        type="button"
                        className="btn btn-ghost"
                        onClick={queryRoomsPage}
                        disabled={!connected}
                      >
                        <List size={15} />
                        查询列表
                      </button>
                    </div>
                  </div>
                </section>

                <section className="card card--soft">
                  <header className="card-head">
                    <h2>房间事件</h2>
                  </header>
                  <div className="card-body">
                    <dl className="event-facts">
                      <div>
                        <dt>加入</dt>
                        <dd>__join_room_up__</dd>
                      </div>
                      <div>
                        <dt>离开</dt>
                        <dd>__leave_room_up__</dd>
                      </div>
                      <div>
                        <dt>列表</dt>
                        <dd>__rooms_page_up__</dd>
                      </div>
                    </dl>
                    <p className="hint">列表查询固定取第 1 页、每页 20 条，结果写入控制台。</p>
                  </div>
                </section>
              </div>

              <section className="card rooms-card">
                <header className="card-head">
                  <div>
                    <h2>房间列表</h2>
                    <p>
                      {roomsPage
                        ? `共 ${roomsPage.total} 个 · 第 ${roomsPage.page}/${roomsPage.totalPages || 1} 页`
                        : '查询当前连接可见的房间'}
                    </p>
                  </div>
                  <button
                    type="button"
                    className="btn btn-ghost"
                    onClick={queryRoomsPage}
                    disabled={!connected || roomsLoading}
                  >
                    {roomsLoading ? <Loader2 size={14} className="spin" /> : <RefreshCw size={14} />}
                    查询
                  </button>
                </header>
                <div className="card-body">
                  {roomsPage ? (
                    roomsPage.rooms.length > 0 ? (
                      <ul className="room-grid">
                        {roomsPage.rooms.map((room) => (
                          <li key={room}>
                            <button
                              type="button"
                              onClick={() => setRoomName(room)}
                              title="填入房间名称"
                            >
                              <Hash size={12} />
                              <span>{room}</span>
                            </button>
                          </li>
                        ))}
                      </ul>
                    ) : (
                      <div className="empty">
                        <DoorOpen size={18} />
                        <p>当前连接没有可见房间</p>
                        <small>加入房间后再次查询。</small>
                      </div>
                    )
                  ) : (
                    <div className="empty">
                      <List size={18} />
                      <p>尚未查询房间列表</p>
                      <small>连接后点击「查询」拉取第 1 页。</small>
                    </div>
                  )}
                </div>
              </section>
              </>
            )}

            {activeTab === 'event' && (
              <>
              <div className="grid-2 grid-2--wide">
                <section className="card">
                  <header className="card-head">
                    <h2>注册监听</h2>
                    <p>自定义事件会在当前连接上监听，收到后写入控制台。</p>
                  </header>
                  <div className="card-body">
                    <div className="inline-form">
                      <input
                        type="text"
                        value={newEventName}
                        onChange={(e) => setNewEventName(e.target.value)}
                        placeholder="输入事件名称，如 meeting_invite"
                        onKeyDown={(e) => e.key === 'Enter' && addNewEvent()}
                      />
                      <button
                        type="button"
                        className="btn btn-primary"
                        onClick={addNewEvent}
                        disabled={!newEventName.trim() || !connected}
                      >
                        <Plus size={15} />
                        添加
                      </button>
                    </div>

                    <div className="event-list">
                      <div className="event-list-head">
                        <span>已注册事件</span>
                        <span>{registeredEvents.length} 个</span>
                      </div>
                      {registeredEvents.length === 0 ? (
                        <div className="empty">
                          <Radio size={18} />
                          <p>暂无事件监听</p>
                          <small>添加事件名称后点击「添加」。</small>
                        </div>
                      ) : (
                        <ul className="event-items">
                          {registeredEvents.map((event) => (
                            <li key={event.name} className="event-item">
                              <div className="event-info">
                                <span className="event-name">{event.name}</span>
                                <span className="event-status" data-active={event.registered}>
                                  {event.registered ? '监听中' : '未监听'}
                                </span>
                              </div>
                              <button
                                type="button"
                                className="icon-btn danger"
                                onClick={() => removeCustomEventListener(event.name)}
                                title="移除监听"
                                aria-label={`移除 ${event.name} 监听`}
                              >
                                <Trash2 size={14} />
                              </button>
                            </li>
                          ))}
                        </ul>
                      )}
                    </div>
                  </div>
                </section>

                <section className="card card--soft">
                  <header className="card-head">
                    <h2>内置事件</h2>
                  </header>
                  <div className="card-body">
                    <dl className="event-facts">
                      <div>
                        <dt>下行</dt>
                        <dd>__down__</dd>
                      </div>
                      <div>
                        <dt>统计</dt>
                        <dd>__stat_down__</dd>
                      </div>
                    </dl>
                    <p className="hint">连接时自动监听，无需手动注册。</p>
                  </div>
                </section>
              </div>

              <section className="card rooms-card">
                <header className="card-head">
                  <div>
                    <h2>接收统计</h2>
                    <p>本次会话按事件名累计</p>
                  </div>
                  <span className="card-meta">{eventStatEntries.length} 个事件</span>
                </header>
                <div className="card-body">
                  {eventStatEntries.length > 0 ? (
                    <table className="stats-table">
                      <thead>
                        <tr>
                          <th scope="col">事件</th>
                          <th scope="col">次数</th>
                          <th scope="col">最近</th>
                        </tr>
                      </thead>
                      <tbody>
                        {eventStatEntries.map(([name, stat]) => (
                          <tr key={name}>
                            <td>{name}</td>
                            <td>{stat.count}</td>
                            <td>{stat.lastAt}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  ) : (
                    <div className="empty">
                      <Radio size={18} />
                      <p>尚未收到事件</p>
                      <small>连接并注册监听后，接收统计会显示在这里。</small>
                    </div>
                  )}
                </div>
              </section>
              </>
            )}
          </div>
        </main>
      </div>

      {consoleOpen ? (
        <section className="console" style={{ height: consoleHeight }} aria-label="调试控制台">
          <div className="console-resizer" onMouseDown={startResize} title="拖拽调整高度" />
          <header className="console-head">
            <div className="console-title">
              <Terminal size={14} />
              <span>控制台</span>
              <span className="console-count">{filteredLogs.length}</span>
            </div>
            <div className="console-tools">
              <div className="console-filters" role="group" aria-label="日志类型筛选">
                {(Object.keys(LOG_LABELS) as (LogType | 'all')[]).map((value) => (
                  <button
                    key={value}
                    type="button"
                    data-active={logFilter === value}
                    aria-pressed={logFilter === value}
                    onClick={() => setLogFilter(value)}
                  >
                    {value !== 'all' && (
                      <i style={{ background: getLogColor(value as LogType) }} aria-hidden="true" />
                    )}
                    {LOG_LABELS[value]}
                  </button>
                ))}
              </div>
              <label className="switch">
                <input
                  type="checkbox"
                  checked={autoScroll}
                  onChange={(e) => setAutoScroll(e.target.checked)}
                />
                <span>自动滚动</span>
              </label>
              <button type="button" className="icon-btn" onClick={exportLogs} title="导出日志" aria-label="导出日志">
                <Download size={14} />
              </button>
              <button type="button" className="icon-btn" onClick={clearLogs} title="清空日志" aria-label="清空日志">
                <Trash2 size={14} />
              </button>
              <button
                type="button"
                className="icon-btn"
                onClick={() => setConsoleOpen(false)}
                title="收起控制台"
                aria-label="收起控制台"
              >
                <ChevronDown size={15} />
              </button>
            </div>
          </header>

          <div className="console-body" ref={consoleBodyRef} role="log" aria-live="polite">
            {filteredLogs.length === 0 ? (
              <div className="console-empty">
                <Terminal size={16} />
                <span>连接服务器后，事件、ACK 与错误会显示在这里</span>
              </div>
            ) : (
              filteredLogs.map((log) => (
                <div key={log.id} className="console-entry">
                  <div
                    className="console-line"
                    data-expandable={Boolean(log.detail)}
                    role={log.detail ? 'button' : undefined}
                    tabIndex={log.detail ? 0 : undefined}
                    onClick={() => log.detail && toggleLogDetail(log.id)}
                    onKeyDown={(e) => {
                      if (log.detail && (e.key === 'Enter' || e.key === ' ')) {
                        e.preventDefault()
                        toggleLogDetail(log.id)
                      }
                    }}
                  >
                    <span className="console-time">{log.timestamp}</span>
                    <span
                      className="console-type"
                      style={{ color: getLogColor(log.type), background: `${getLogColor(log.type)}1f` }}
                    >
                      {getLogIcon(log.type)}
                      {log.type.toUpperCase()}
                    </span>
                    <span className="console-message">{log.message}</span>
                    {log.detail && (
                      <span className="console-chevron" data-open={expandedLogs.has(log.id)}>
                        <ChevronRight size={12} />
                      </span>
                    )}
                  </div>
                  {log.detail && expandedLogs.has(log.id) && (
                    <div className="console-detail-wrap">
                      <button
                        type="button"
                        className="console-copy"
                        onClick={() => copyToClipboard(log.detail!, log.id)}
                        title="复制详情"
                      >
                        {copied === log.id ? <Check size={12} /> : <Copy size={12} />}
                      </button>
                      <pre className="console-detail">{log.detail}</pre>
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </section>
      ) : (
        <button
          type="button"
          className="console-fab"
          onClick={() => setConsoleOpen(true)}
          title="打开控制台"
        >
          <Terminal size={15} />
          <span>控制台</span>
          {unreadCount > 0 && (
            <span className="console-fab-badge">{unreadCount > 99 ? '99+' : unreadCount}</span>
          )}
        </button>
      )}
    </div>
  )
}
