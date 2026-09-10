import { useCallback, useEffect, useMemo, useRef, useState, type MouseEvent as ReactMouseEvent } from 'react'
import { io, type Socket } from 'socket.io-client'
import { 
  Plug, Unplug, Send, Radio, Globe, DoorOpen, DoorClosed, 
  List, Plus, Trash2, Settings, Zap, 
  MessageSquare, Users, Server, Filter, Download,
  Copy, Check, AlertCircle,
  Info, Wifi, WifiOff, Loader2,
  Eye, Terminal, ChevronDown, ChevronRight
} from 'lucide-react'
import { decodeJwtClaims } from './lib/token'

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
  const [serverUrl, setServerUrl] = useState(import.meta.env.VITE_SOCKET_URL || 'http://127.0.0.1:11003')
  const [authToken, setAuthToken] = useState('')
  
  // 消息状态
  const [eventName, setEventName] = useState('custom_event')
  const [payload, setPayload] = useState('Hello SocketIO!')
  const [broadcastRoom, setBroadcastRoom] = useState('test-room')
  
  // 房间状态
  const [roomName, setRoomName] = useState('test-room')
  
  // 事件状态
  const [newEventName, setNewEventName] = useState('')

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

  // 注册自定义事件监听器
  const registerCustomEventListener = useCallback((eventName: string) => {
    if (!socket) return

    const handler = (data: unknown) => {
      addLog('receive', `收到 ${eventName}`, JSON.stringify(data, null, 2))
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
  }, [socket, addLog, saveRegisteredEvents])

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
    })

    newSocket.on('__stat_down__', (data: unknown) => {
      addLog('receive', '收到 __stat_down__ 事件', JSON.stringify(data, null, 2))
    })

    setSocket(newSocket)
  }, [serverUrl, authToken, eventName, addLog, registeredEvents, saveRegisteredEvents, connectionState])

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
    socket.emit('__rooms_page_up__', {
      reqId,
      page: 1,
      pageSize: 20
    }, (ack: unknown) => {
      addLog('message', '房间列表查询结果', JSON.stringify(ack, null, 2))
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

  return (
    <div className="app" style={{ paddingBottom: consoleOpen ? consoleHeight : 64 }}>
      {/* 顶部导航 */}
      <header className="header">
        <div className="header-left">
          <div className="logo">
            <Zap size={24} />
            <span>SocketIO 调试工具</span>
          </div>
          <div className="connection-status" style={{ color: getConnectionColor() }}>
            {connectionState === 'connected' ? <Wifi size={16} /> : <WifiOff size={16} />}
            <span>{getConnectionText()}</span>
            {socketId && <span className="socket-id">({socketId})</span>}
          </div>
        </div>
        <div className="header-right">
          <button 
            className="btn btn-primary"
            onClick={connectionState === 'connected' ? disconnect : connect}
            disabled={connectionState === 'connecting'}
          >
            {connectionState === 'connected' ? (
              <>
                <Unplug size={16} />
                断开连接
              </>
            ) : connectionState === 'connecting' ? (
              <>
                <Loader2 size={16} className="spinner" />
                连接中...
              </>
            ) : (
              <>
                <Plug size={16} />
                连接
              </>
            )}
          </button>
        </div>
      </header>

      <div className="main">
        {/* 侧边栏 */}
        <aside className="sidebar">
          <nav className="nav">
            <button 
              className={`nav-item ${activeTab === 'config' ? 'active' : ''}`}
              onClick={() => setActiveTab('config')}
            >
              <Settings size={18} />
              <span>连接配置</span>
            </button>
            <button 
              className={`nav-item ${activeTab === 'message' ? 'active' : ''}`}
              onClick={() => setActiveTab('message')}
            >
              <Send size={18} />
              <span>消息发送</span>
            </button>
            <button 
              className={`nav-item ${activeTab === 'room' ? 'active' : ''}`}
              onClick={() => setActiveTab('room')}
            >
              <Users size={18} />
              <span>房间管理</span>
            </button>
            <button 
              className={`nav-item ${activeTab === 'event' ? 'active' : ''}`}
              onClick={() => setActiveTab('event')}
            >
              <Radio size={18} />
              <span>事件监听</span>
            </button>
            <button 
              className={`nav-item ${consoleOpen ? 'active' : ''}`}
              onClick={() => setConsoleOpen(prev => !prev)}
            >
              <Terminal size={18} />
              <span>控制台</span>
              {logs.length > 0 && (
                <span className={`badge ${!consoleOpen && unreadCount > 0 ? 'badge-alert' : ''}`}>
                  {!consoleOpen && unreadCount > 0 ? unreadCount : logs.length}
                </span>
              )}
            </button>
          </nav>
        </aside>

        {/* 内容区域 */}
        <main className="content">
          {/* 连接配置 */}
          {activeTab === 'config' && (
            <div className="panel">
              <div className="panel-header">
                <h2>
                  <Settings size={20} />
                  连接配置
                </h2>
              </div>
              <div className="panel-body">
                <div className="form-group">
                  <label>
                    <Server size={14} />
                    服务器地址
                  </label>
                  <input
                    type="text"
                    value={serverUrl}
                    onChange={(e) => setServerUrl(e.target.value)}
                    placeholder="http://127.0.0.1:11003"
                    disabled={connectionState === 'connected'}
                  />
                </div>

                <div className="form-group">
                  <label>
                    <Zap size={14} />
                    Authorization Token
                  </label>
                  <input
                    type="password"
                    value={authToken}
                    onChange={(e) => setAuthToken(e.target.value)}
                    placeholder="输入认证 Token（可在 live 子系统签发）"
                    disabled={connectionState === 'connected'}
                  />
                  <small className="form-hint">
                    Token 由 live 子系统签发（POST /live/v1/generateToken），此处仅用于粘贴使用
                  </small>
                </div>

                {authToken && (
                  <div className="claims-box">
                    <div className="claims-header">
                      <Eye size={14} />
                      <span>Token Claims 解析</span>
                      <small>
                        {Object.keys(tokenClaims).length > 0
                          ? `${Object.keys(tokenClaims).length} 个 claims`
                          : '无法解析'}
                      </small>
                    </div>
                    {Object.keys(tokenClaims).length > 0 ? (
                      <div className="claims-list">
                        {Object.entries(tokenClaims).map(([key, value]) => (
                          <div key={key} className="claim-item">
                            <span className="claim-key">{key}</span>
                            <span className="claim-value">{String(value)}</span>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="claims-empty">Token 格式无效或不是 JWT</div>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}

          {/* 消息发送 */}
          {activeTab === 'message' && (
            <div className="panel">
              <div className="panel-header">
                <h2>
                  <Send size={20} />
                  消息发送
                </h2>
              </div>
              <div className="panel-body">
                <div className="form-group">
                  <label>
                    <Radio size={14} />
                    事件名称
                  </label>
                  <input
                    type="text"
                    value={eventName}
                    onChange={(e) => setEventName(e.target.value)}
                    placeholder="输入事件名称"
                  />
                </div>

                <div className="form-group">
                  <label>
                    <MessageSquare size={14} />
                    消息内容
                  </label>
                  <textarea
                    value={payload}
                    onChange={(e) => setPayload(e.target.value)}
                    placeholder="输入要发送的消息内容"
                    rows={4}
                  />
                </div>

                <div className="button-group">
                  <button 
                    className="btn btn-primary"
                    onClick={sendUp}
                    disabled={connectionState !== 'connected'}
                  >
                    <Send size={16} />
                    发送上行消息
                  </button>
                </div>

                <div className="divider" />

                <div className="form-group">
                  <label>
                    <Users size={14} />
                    目标房间
                  </label>
                  <input
                    type="text"
                    value={broadcastRoom}
                    onChange={(e) => setBroadcastRoom(e.target.value)}
                    placeholder="输入房间名称"
                  />
                </div>

                <div className="button-group">
                  <button 
                    className="btn btn-secondary"
                    onClick={sendRoom}
                    disabled={connectionState !== 'connected'}
                  >
                    <Radio size={16} />
                    发送房间广播
                  </button>
                  <button 
                    className="btn btn-secondary"
                    onClick={sendGlobal}
                    disabled={connectionState !== 'connected'}
                  >
                    <Globe size={16} />
                    发送全局广播
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* 房间管理 */}
          {activeTab === 'room' && (
            <div className="panel">
              <div className="panel-header">
                <h2>
                  <Users size={20} />
                  房间管理
                </h2>
              </div>
              <div className="panel-body">
                <div className="form-group">
                  <label>
                    <DoorOpen size={14} />
                    房间名称
                  </label>
                  <input
                    type="text"
                    value={roomName}
                    onChange={(e) => setRoomName(e.target.value)}
                    placeholder="输入房间名称"
                  />
                </div>

                <div className="button-group">
                  <button 
                    className="btn btn-primary"
                    onClick={joinRoom}
                    disabled={connectionState !== 'connected'}
                  >
                    <DoorOpen size={16} />
                    加入房间
                  </button>
                  <button 
                    className="btn btn-secondary"
                    onClick={leaveRoom}
                    disabled={connectionState !== 'connected'}
                  >
                    <DoorClosed size={16} />
                    离开房间
                  </button>
                  <button 
                    className="btn btn-outline"
                    onClick={queryRoomsPage}
                    disabled={connectionState !== 'connected'}
                  >
                    <List size={16} />
                    查询房间列表
                  </button>
                </div>
              </div>
            </div>
          )}

          {/* 事件监听 */}
          {activeTab === 'event' && (
            <div className="panel">
              <div className="panel-header">
                <h2>
                  <Radio size={20} />
                  事件监听
                </h2>
              </div>
              <div className="panel-body">
                <div className="form-group">
                  <label>
                    <Plus size={14} />
                    添加新事件监听
                  </label>
                  <div className="input-group">
                    <input
                      type="text"
                      value={newEventName}
                      onChange={(e) => setNewEventName(e.target.value)}
                      placeholder="输入事件名称"
                      onKeyDown={(e) => e.key === 'Enter' && addNewEvent()}
                    />
                    <button 
                      className="btn btn-primary"
                      onClick={addNewEvent}
                      disabled={!newEventName.trim() || connectionState !== 'connected'}
                    >
                      <Plus size={16} />
                      添加
                    </button>
                  </div>
                </div>

                <div className="event-list">
                  <div className="event-list-header">
                    <span>已注册的事件监听</span>
                    <span className="event-count">{registeredEvents.length} 个事件</span>
                  </div>
                  {registeredEvents.length === 0 ? (
                    <div className="event-empty">
                      <Radio size={32} />
                      <p>暂无注册的事件监听</p>
                      <small>添加事件名称后点击"添加"按钮</small>
                    </div>
                  ) : (
                    <div className="event-items">
                      {registeredEvents.map((event) => (
                        <div key={event.name} className="event-item">
                          <div className="event-info">
                            <span className="event-name">{event.name}</span>
                            <span className={`event-status ${event.registered ? 'active' : ''}`}>
                              {event.registered ? '监听中' : '未监听'}
                            </span>
                          </div>
                          <button
                            className="btn btn-icon btn-danger"
                            onClick={() => removeCustomEventListener(event.name)}
                            title="移除监听"
                          >
                            <Trash2 size={14} />
                          </button>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}

        </main>
      </div>

      {/* 底部控制台 */}
      {consoleOpen ? (
        <section className="console" style={{ height: consoleHeight }}>
          <div className="console-resizer" onMouseDown={startResize} title="拖拽调整高度" />
          <div className="console-header">
            <div className="console-title">
              <Terminal size={14} />
              <span>控制台</span>
              <span className="console-count">{filteredLogs.length} 条</span>
            </div>
            <div className="console-actions">
              <div className="console-filter">
                <Filter size={12} />
                <select
                  value={logFilter}
                  onChange={(e) => setLogFilter(e.target.value as LogType | 'all')}
                >
                  <option value="all">全部</option>
                  <option value="system">系统</option>
                  <option value="connect">连接</option>
                  <option value="disconnect">断开</option>
                  <option value="message">发送</option>
                  <option value="receive">接收</option>
                  <option value="error">错误</option>
                </select>
              </div>
              <label className="console-checkbox">
                <input
                  type="checkbox"
                  checked={autoScroll}
                  onChange={(e) => setAutoScroll(e.target.checked)}
                />
                <span>自动滚动</span>
              </label>
              <button className="console-btn" onClick={exportLogs} title="导出日志">
                <Download size={13} />
              </button>
              <button className="console-btn" onClick={clearLogs} title="清空日志">
                <Trash2 size={13} />
              </button>
              <button className="console-btn" onClick={() => setConsoleOpen(false)} title="收起控制台">
                <ChevronDown size={15} />
              </button>
            </div>
          </div>
          <div className="console-body" ref={consoleBodyRef}>
            {filteredLogs.length === 0 ? (
              <div className="console-empty">暂无日志记录，连接服务器后日志将在此显示</div>
            ) : (
              filteredLogs.map((log) => (
                <div key={log.id} className="console-entry">
                  <div
                    className={`console-line ${log.detail ? 'expandable' : ''}`}
                    onClick={() => log.detail && toggleLogDetail(log.id)}
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
                      <span className={`console-chevron ${expandedLogs.has(log.id) ? 'open' : ''}`}>
                        <ChevronRight size={12} />
                      </span>
                    )}
                  </div>
                  {log.detail && expandedLogs.has(log.id) && (
                    <div className="console-detail-wrap">
                      <button
                        className="console-btn console-copy"
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
        <button className="console-fab" onClick={() => setConsoleOpen(true)} title="打开控制台">
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
