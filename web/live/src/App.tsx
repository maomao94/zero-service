import { useCallback, useEffect, useRef, useState } from 'react'
import { LiveKitRoom, RoomAudioRenderer, TrackReference, TrackReferenceOrPlaceholder, VideoTrack, useLocalParticipant, useParticipants, useRoomContext, useTracks } from '@livekit/components-react'
import { Room, RoomEvent, Track } from 'livekit-client'
import { Archive, ArrowRight, Camera, Check, ChevronDown, ChevronLeft, ChevronRight, ChevronUp, Clipboard, Copy, Database, DoorOpen, History, LogOut, Maximize2, MessageSquare, Mic, Minimize2, MonitorUp, MoreHorizontal, Phone, Plus, RefreshCw, Search, Send, Settings2, Shield, ShieldCheck, Sparkles, UserRound, Users, Video, X, ZoomIn, ZoomOut } from 'lucide-react'
import { api, ApiError } from './lib/api'
import type { MeetingInfo, MeetingMessage, ParticipantInfo, SipProviderInfo, TicketReply } from './types'

type Toast = { message: string; tone?: 'error' | 'success' | 'warning' }
type Screen = 'auth' | 'lobby' | 'room' | 'guest'
type JoinPerms = { canPublish: boolean; canSubscribe: boolean; canPublishData: boolean; canPublishSources: string[] | null }
type JoinState = { token: string; meeting: MeetingInfo; perms: JoinPerms }

function initials(name: string) { return name.trim().split(/\s+/).map((part) => part[0]).join('').slice(0, 2).toUpperCase() || 'L' }
function decodeToken(token: string) { try { const payload = token.split('.')[1]; return JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/'))) } catch { return {} } }
function timeLabel(value: string) { return value || '时间未记录' }
function statusMeta(status: number) { return status === 3 ? { label: '已结束', className: 'ended' } : status === 2 ? { label: '进行中', className: 'live' } : { label: '已创建', className: 'created' } }
function formatDuration(startTime: string, endTime?: string) { const start = new Date(startTime).getTime(); if (!Number.isFinite(start)) return '未开始'; const parsedEnd = endTime ? new Date(endTime).getTime() : Date.now(); const end = Number.isFinite(parsedEnd) ? parsedEnd : Date.now(); const diff = Math.max(0, end - start); const hours = Math.floor(diff / 3600000); const minutes = Math.floor((diff % 3600000) / 60000); const seconds = Math.floor((diff % 60000) / 1000); if (hours > 0) return `${hours}时${minutes}分`; if (minutes > 0) return `${minutes}分${seconds}秒`; return `${seconds}秒` }
function formatMeetingCode(value: string): string { const digits = value.replace(/\D/g, '').slice(0, 9); if (digits.length <= 3) return digits; if (digits.length <= 6) return `${digits.slice(0, 3)}-${digits.slice(3)}`; return `${digits.slice(0, 3)}-${digits.slice(3, 6)}-${digits.slice(6)}` }
function stripMeetingCode(value: string): string { return value.replace(/\D/g, '') }
function isMeetingCode(value: string): boolean { return /^\d{9}$/.test(stripMeetingCode(value)) }
function wsUrl(): string { return `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}${(import.meta.env.VITE_API_ROOT || '/live/v1').replace(/\/$/, '')}` }

export default function App() {
  const isGuestPath = location.pathname.replace(/\/+$/, '') === '/guest'
  const queryTicket = new URLSearchParams(location.search).get('ticket')
  const [screen, setScreen] = useState<Screen>(isGuestPath && queryTicket ? 'guest' : localStorage.getItem('live_jwt') ? 'lobby' : 'auth')
  const [token, setToken] = useState(localStorage.getItem('live_jwt') || '')
  const [guest, setGuest] = useState(isGuestPath && Boolean(queryTicket))
  const [identity, setIdentity] = useState('')
  const [name, setName] = useState('')
  const [deptCode, setDeptCode] = useState('')
  const [join, setJoin] = useState<JoinState | null>(null)
  const [toast, setToast] = useState<Toast | null>(null)

  const notify = useCallback((message: string, tone?: Toast['tone']) => setToast({ message, tone }), [])
  useEffect(() => { if (!toast) return; const timer = window.setTimeout(() => setToast(null), 4500); return () => window.clearTimeout(timer) }, [toast])

  useEffect(() => {
    if (!token || guest) return
    api.getCurrentUser().then((user) => { setIdentity(user.userId); setName(user.userName); setDeptCode(user.deptCode) }).catch(() => {
      const claim = decodeToken(token); const id = String(claim.sub || claim.user_id || claim.userId || `user-${Math.random().toString(36).slice(2, 8)}`); const display = String(claim.name || claim.nickname || claim.userName || id)
      setIdentity(id); setName(display)
    })
  }, [token, guest])

  const login = (nextToken: string) => { if (nextToken.split('.').length !== 3) throw new ApiError('Token 格式不正确，应为标准 JWT', 400); const claim = decodeToken(nextToken) as Record<string, unknown>; if (claim.exp && Number(claim.exp) * 1000 < Date.now()) throw new ApiError('Token 已过期', 401); if (!claim.sub && !claim.user_id && !claim.userId) throw new ApiError('Token 无效：缺少用户身份信息', 400); localStorage.setItem('live_jwt', nextToken); setToken(nextToken); setGuest(false); setScreen('lobby') }
  const logout = () => { localStorage.removeItem('live_jwt'); setToken(''); setJoin(null); setGuest(false); setScreen('auth') }
  const enterMeeting = async (meetingNo: string, options?: { canPublish?: boolean; canSubscribe?: boolean; canPublishData?: boolean }) => {
    const isCode = isMeetingCode(meetingNo)
    const reply = await api.joinMeeting(meetingNo, {
      meetingCode: isCode ? stripMeetingCode(meetingNo) : undefined,
      canPublish: options?.canPublish ?? true,
      canSubscribe: options?.canSubscribe ?? true,
      canPublishData: options?.canPublishData ?? true,
    })
    setJoin({ ...reply, perms: { canPublish: reply.canPublish, canSubscribe: reply.canSubscribe, canPublishData: reply.canPublishData, canPublishSources: reply.canPublishSources || null } }); setScreen('room')
  }
  const joinByTicket = useCallback(async (ticket: string) => {
    try {
      const guestId = '访客_' + Math.random().toString(36).slice(2, 8)
      setName(guestId)
      setIdentity(guestId)
      const reply = await api.joinByTicket(ticket)
      setJoin({ ...reply, perms: { canPublish: reply.canPublish, canSubscribe: reply.canSubscribe, canPublishData: reply.canPublishData, canPublishSources: reply.canPublishSources || null } })
      notify('票据验证成功，正在进入会议', 'success')
      return true
    } catch (error) {
      notify((error as Error).message, 'error')
      return false
    }
  }, [notify])

  return <>
    {screen === 'auth' && <AuthView initialToken={token} onLogin={login} notify={notify} />}
    {screen === 'lobby' && <><Topbar name={name} identity={identity} deptCode={deptCode} onLogout={logout} /><LobbyView name={name} identity={identity} onJoin={enterMeeting} notify={notify} /></>}
    {screen === 'room' && join && <><Topbar name={name} identity={identity} deptCode={deptCode} guest={guest} onLogout={logout} /><LiveKitRoom serverUrl={wsUrl()} token={join.token} connect audio={true} video={true} onDisconnected={() => { setJoin(null); setScreen(guest ? 'auth' : 'lobby') }}><MeetingRoom meeting={join.meeting} perms={join.perms} name={name} identity={identity} guest={guest} onLeave={() => { setJoin(null); setScreen(guest ? 'auth' : 'lobby') }} notify={notify} /></LiveKitRoom></>}
    {screen === 'guest' && <GuestView ticket={queryTicket} join={join} name={name} identity={identity} onJoin={joinByTicket} onLeave={() => { setJoin(null); setScreen('auth') }} notify={notify} />}
    {toast && <div className={`toast ${toast.tone || ''}`}><span>{toast.tone === 'success' ? <Check size={16} /> : toast.tone === 'error' ? <X size={16} /> : <Sparkles size={16} />}</span>{toast.message}</div>}
  </>
}

function AuthView({ initialToken, onLogin, notify }: { initialToken: string; onLogin: (token: string) => void; notify: (message: string, tone?: Toast['tone']) => void }) {
  const [value, setValue] = useState(initialToken); const [visible, setVisible] = useState(false); const [error, setError] = useState('');
  const submit = () => { if (!value.trim()) { setError('请输入 JWT Token'); return } try { onLogin(value.trim()) } catch (e) { setError((e as Error).message); notify((e as Error).message, 'error') } }
  return <main className="auth-page"><div className="auth-art"><div className="auth-orbit orbit-one" /><div className="auth-orbit orbit-two" /><div className="auth-art-copy"><span className="kicker">LIVE / VIDEO</span><h1>清晰通话<br /><em>随时开会。</em></h1><p>稳定、低延迟的视频会议，支持多人通话、屏幕共享与会议管理。</p><div className="auth-art-foot"><span><ShieldCheck size={16} />安全入会</span><span><Video size={16} />低延迟通话</span></div></div></div><section className="auth-card"><div className="brand-lockup"><span className="brand-mark">L</span><span>Live 视频会议</span></div><div className="auth-heading"><span className="eyebrow">欢迎回来</span><h2>登录视频会议</h2><p>使用业务 JWT 登录，管理你的会议。</p></div><label className="field-label" htmlFor="jwt">JWT Token</label><div className="secret-input"><input id="jwt" value={value} onChange={(e) => setValue(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && submit()} type={visible ? 'text' : 'password'} placeholder="粘贴 JWT Token" autoComplete="off" /><button type="button" aria-label={visible ? '隐藏 token' : '显示 token'} onClick={() => setVisible(!visible)}>{visible ? '隐藏' : '显示'}</button></div>{error && <div className="form-error">{error}</div>}<button className="button primary wide" onClick={submit}>登录并继续 <ArrowRight size={17} /></button><p className="auth-note">Token 仅保存在当前浏览器，不会上传到第三方。</p></section></main>
}

function Topbar({ name, identity, deptCode, guest, onLogout }: { name: string; identity: string; deptCode?: string; guest?: boolean; onLogout: () => void }) { const displayName = name || identity || '用户'; return <header className="topbar"><div className="topbar-left"><span className="brand-mark">L</span><span className="topbar-title">Live 视频会议</span><span className="topbar-divider" /><span className="topbar-context">视频会议</span></div><div className="topbar-right"><div className="presence"><span className="presence-dot" />在线</div><div className="profile" title={`用户ID: ${identity}\n姓名: ${name}\n部门: ${deptCode || '未设置'}`}><span className="avatar small">{initials(displayName)}</span><span className="profile-copy"><b>{displayName}</b><small>{guest ? '邀请访客' : identity}</small></span></div><button className="icon-button" title={guest ? '退出会议' : '退出登录'} onClick={onLogout}><LogOut size={17} /></button></div></header> }

function LobbyView({ name, identity, onJoin, notify }: { name: string; identity: string; onJoin: (meetingNo: string, options?: { canPublish?: boolean; canSubscribe?: boolean; canPublishData?: boolean }) => Promise<void>; notify: (message: string, tone?: Toast['tone']) => void }) {
  const [title, setTitle] = useState(''); const [meetingNo, setMeetingNo] = useState(''); const [rows, setRows] = useState<MeetingInfo[]>([]); const [total, setTotal] = useState(0); const [mode, setMode] = useState<'mine' | 'all'>('mine'); const [status, setStatus] = useState('0'); const [search, setSearch] = useState(''); const [loading, setLoading] = useState(true);
  const [workspaceTab, setWorkspaceTab] = useState<'meetings' | 'providers'>('meetings')
  const [showAdvanced, setShowAdvanced] = useState(false); const [showTestPhone, setShowTestPhone] = useState(false); const [canPublish, setCanPublish] = useState(true); const [canSubscribe, setCanSubscribe] = useState(true); const [canPublishData, setCanPublishData] = useState(true); const [creating, setCreating] = useState(false); const [joining, setJoining] = useState(false);
  const greeting = (() => { const h = new Date().getHours(); if (h < 6) return '夜深了'; if (h < 12) return '早上好'; if (h < 14) return '中午好'; if (h < 18) return '下午好'; return '晚上好' })();
  const displayName = name || identity || '用户';
  const load = useCallback(async () => { setLoading(true); try { const params: Record<string, string | number> = { page: 1, pageSize: 20 }; if (status !== '0') params.status = status; if (search) params.title = search; const data = mode === 'mine' ? await api.listMyMeetings(1, 20) : await api.listMeetings(params); const filtered = mode === 'mine' ? (data.meetings || []).filter((meeting) => (!search || meeting.title.toLowerCase().includes(search.toLowerCase())) && (status === '0' || String(meeting.status) === status)) : (data.meetings || []); setRows(filtered); setTotal(mode === 'mine' ? filtered.length : data.total || 0) } catch (e) { notify((e as Error).message, 'error') } finally { setLoading(false) } }, [mode, notify, search, status]);
  useEffect(() => { load() }, [load]);
  const create = async () => { if (creating) return; setCreating(true); try { const data = await api.createMeeting(title.trim() || 'Live 会议'); await onJoin(data.meeting.meetingNo) } catch (e) { notify((e as Error).message, 'error') } finally { setCreating(false) } }
  const join = async () => { if (!meetingNo.trim()) return notify('请输入会议号或9位会议码', 'warning'); if (joining) return; setJoining(true); try { await onJoin(meetingNo.trim(), { canPublish, canSubscribe, canPublishData }) } catch (e) { notify((e as Error).message, 'error') } finally { setJoining(false) } }
  const endMeeting = async (meetingNo: string) => { try { await api.endMeeting(meetingNo); notify('会议已结束', 'success'); load() } catch (e) { notify((e as Error).message, 'error') } }
  const handleMeetingNoChange = (e: React.ChangeEvent<HTMLInputElement>) => { const raw = e.target.value; if (/^\d{9}$/.test(stripMeetingCode(raw))) { setMeetingNo(formatMeetingCode(raw)) } else { setMeetingNo(raw) } }
  return <main className="lobby-page"><div className="page-heading"><div><span className="eyebrow">工作台</span><h1>{greeting}，{displayName}</h1><p>管理会议与 SIP 电话联调。</p></div><div className="heading-metric"><span className="metric-icon"><History size={17} /></span><div><b>{total}</b><small>{mode === 'mine' ? '我的会议' : '全部会议'}</small></div></div></div><nav className="workspace-tabs"><button className={workspaceTab === 'meetings' ? 'active' : ''} onClick={() => setWorkspaceTab('meetings')}><Video size={15} />会议与电话</button><button className={workspaceTab === 'providers' ? 'active' : ''} onClick={() => setWorkspaceTab('providers')}><Settings2 size={15} />SIP 供应商</button></nav>{workspaceTab === 'providers' ? <SipProviderPanel notify={notify} /> : <><section className={`test-phone-tool surface ${showTestPhone ? 'open' : ''}`}><button className="test-phone-launcher" onClick={() => setShowTestPhone((visible) => !visible)} aria-expanded={showTestPhone}><span className="tool-icon"><Phone size={16} /></span><span><b>测试电话</b><small>独立 SIP 外呼工具</small></span><span className="tool-status">{showTestPhone ? '收起' : '展开'}<ChevronDown size={15} className={showTestPhone ? 'rotated' : ''} /></span></button>{showTestPhone && <div className="test-phone-content"><div className="test-phone-copy"><span className="eyebrow">SIP TEST</span><p>选择供应商并直接发起外呼，不加入当前会议。</p></div><DialPad notify={notify} /></div>}</section><div className="lobby-layout"><section className="create-column"><div className="surface create-card"><div className="card-kicker"><span className="icon-badge teal"><Plus size={18} /></span><span>创建会议</span></div><h2>创建一场新会议</h2><p>创建后立即进入会议，也可以在会议中生成访客邀请。</p><label className="field-label" htmlFor="meeting-title">会议名称</label><input id="meeting-title" className="text-input" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="例如：产品评审 / 周会" onKeyDown={(e) => e.key === 'Enter' && create()} /><button className="button primary wide" onClick={create} disabled={creating}>{creating ? '创建中…' : '创建并进入'} <ArrowRight size={16} /></button><div className="split-line"><span>或</span></div><div className="card-kicker"><span className="icon-badge amber"><DoorOpen size={17} /></span><span>加入会议</span></div><label className="field-label" htmlFor="meeting-no">会议号</label><input id="meeting-no" className="text-input" value={meetingNo} onChange={handleMeetingNoChange} placeholder="会议号或9位会议码 (000-000-000)" onKeyDown={(e) => e.key === 'Enter' && join()} />
            <button className="text-button small" onClick={() => setShowAdvanced(!showAdvanced)}>{showAdvanced ? <ChevronUp size={14} /> : <ChevronDown size={14} />} 高级选项</button>
            {showAdvanced && <div className="advanced-options"><label className="checkbox-label"><input type="checkbox" checked={canPublish} onChange={(e) => setCanPublish(e.target.checked)} />允许发布音视频</label><label className="checkbox-label"><input type="checkbox" checked={canSubscribe} onChange={(e) => setCanSubscribe(e.target.checked)} />允许订阅音视频</label><label className="checkbox-label"><input type="checkbox" checked={canPublishData} onChange={(e) => setCanPublishData(e.target.checked)} />允许发送消息/数据</label></div>}
            <button className="button secondary wide" onClick={join} disabled={joining}>{joining ? '加入中…' : '加入会议'} <ArrowRight size={16} /></button></div><div className="tip-card"><Shield size={17} /><div><b>会议数据受保护</b><span>只有授权成员可以执行会议管理操作。</span></div></div></section><section className="surface history-card"><div className="section-heading"><div><span className="eyebrow">会议记录</span><div className="section-title-line"><h2>最近的会议</h2><span className="section-count">{total} 场</span></div></div><button className="icon-button" title="刷新会议记录" onClick={load}><RefreshCw size={17} className={loading ? 'spin' : ''} /></button></div><div className="history-tabs"><button className={mode === 'mine' ? 'active' : ''} onClick={() => setMode('mine')}>我的会议</button><button className={mode === 'all' ? 'active' : ''} onClick={() => setMode('all')}>全部会议</button></div><div className="history-filters"><div className="search-input"><Search size={16} /><input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="搜索会议名称" /></div><select value={status} onChange={(e) => setStatus(e.target.value)}><option value="0">全部状态</option><option value="1">已创建</option><option value="2">进行中</option><option value="3">已结束</option></select></div><div className="meeting-list">{loading ? <div className="empty-state"><RefreshCw size={18} className="spin" />加载记录中…</div> : rows.length === 0 ? <div className="empty-state"><Archive size={20} />还没有符合条件的会议</div> : rows.map((meeting) => <MeetingRow key={meeting.meetingNo} meeting={meeting} onJoin={onJoin} onEnd={endMeeting} notify={notify} />)}</div><div className="list-footer"><span>共 {total} 场会议</span><span>当前身份：{identity}</span></div></section></div></>}</main>
}

function MeetingRow({ meeting, onJoin, onEnd, notify }: { meeting: MeetingInfo; onJoin: (no: string) => Promise<void>; onEnd: (no: string) => Promise<void>; notify: (message: string, tone?: Toast['tone']) => void }) {
  const [expanded, setExpanded] = useState(false)
  const [ticketLoading, setTicketLoading] = useState(false)
  const [ticketInfo, setTicketInfo] = useState<{ ticket: string; joinUrl: string; ticketType?: number; canPublish?: boolean; canSubscribe?: boolean; canPublishData?: boolean; canPublishSources?: string[] } | null>(null)
  const [showTicketModal, setShowTicketModal] = useState(false)
  const [ticketError, setTicketError] = useState('')
  const [showDetailModal, setShowDetailModal] = useState(false)
  const [ticketIdentity, setTicketIdentity] = useState('')
  const [ticketName, setTicketName] = useState('')
  const [ticketExpire, setTicketExpire] = useState('3600')
  const [canPublish, setCanPublish] = useState(true)
  const [canSubscribe, setCanSubscribe] = useState(true)
  const [canPublishData, setCanPublishData] = useState(true)
  const [canPublishSources, setCanPublishSources] = useState<string[]>(['camera', 'microphone', 'screen_share'])
  const [ticketType, setTicketType] = useState<1 | 2>(1)
  const [detailData, setDetailData] = useState<{ meeting: MeetingInfo; participants: ParticipantInfo[] } | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [detailError, setDetailError] = useState('')
  const [joining, setJoining] = useState(false)
  const meta = statusMeta(Number(meeting.status))
  const duration = formatDuration(meeting.startTime, meeting.status === 3 ? meeting.endTime : undefined)
  const isActive = Number(meeting.status) !== 3
  const joinNow = async () => { if (joining) return; setJoining(true); try { await onJoin(meeting.meetingNo) } catch (e) { notify((e as Error).message, 'error') } finally { setJoining(false) } }
  const openDetail = async () => {
    setShowDetailModal(true)
    setDetailLoading(true)
    setDetailError('')
    try {
      const [meetingRes, participantsRes] = await Promise.all([
        api.getMeeting(meeting.meetingNo),
        api.listParticipants(meeting.meetingNo)
      ])
      setDetailData({ meeting: meetingRes.meeting, participants: participantsRes.participants || [] })
    } catch (e) {
      const message = (e as Error).message || '加载会议详情失败，请稍后重试'
      setDetailError(message)
      notify(message, 'error')
    } finally {
      setDetailLoading(false)
    }
  }
  const toggleSource = (source: string) => setCanPublishSources((current) => current.includes(source) ? current.filter((s) => s !== source) : [...current, source])
  const generateTicket = async () => {
    const guestIdentity = ticketIdentity.trim() || ('访客_' + Math.random().toString(36).slice(2, 8))
    setTicketError('')
    setTicketLoading(true)
    try {
      const reply = await api.generateTicket(meeting.meetingNo, guestIdentity, ticketName.trim(), Number(ticketExpire) || 3600, canPublish, canSubscribe, canPublishData, canPublishSources, ticketType)
      const joinUrl = `${location.origin}/guest?ticket=${encodeURIComponent(reply.ticket)}`
      setTicketInfo({ ticket: reply.ticket, joinUrl, ticketType: reply.ticketType, canPublish, canSubscribe, canPublishData, canPublishSources })
    } catch (e) {
      const message = (e as Error).message || '生成邀请票据失败，请稍后重试'
      setTicketError(message)
      notify(message, 'error')
    } finally {
      setTicketLoading(false)
    }
  }
  const copyUrl = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      notify('已复制到剪贴板', 'success')
    } catch {
      notify('复制失败', 'error')
    }
  }
  return <>
    <div className={`meeting-row ${expanded ? 'expanded' : ''}`} onClick={() => setExpanded(!expanded)}>
      <div className="meeting-row-info">
        <div className="meeting-row-title">
          <span className={`status-dot ${meta.className}`} />
          <b>{meeting.title || '未命名会议'}</b>
          {meeting.meetingCode && <button className="meeting-code-button" onClick={(e) => { e.stopPropagation(); copyUrl(formatMeetingCode(meeting.meetingCode)) }} title="复制会议码">{formatMeetingCode(meeting.meetingCode)}</button>}
          <span className={`status-pill ${meta.className}`}>{meta.label}</span>
        </div>
        <small>{meeting.meetingNo} · {timeLabel(meeting.createTime)} · 时长 {duration}</small>
      </div>
      <div className="meeting-actions">
        <button className="text-button" onClick={(e) => { e.stopPropagation(); openDetail() }}>详情</button>
        {isActive && <button className="text-button danger" onClick={(e) => { e.stopPropagation(); window.confirm('确定结束该会议？') && onEnd(meeting.meetingNo) }}>结束</button>}
        <button className="button compact secondary" disabled={!isActive || joining} onClick={(e) => { e.stopPropagation(); joinNow() }}>{joining ? '加入中…' : '加入'} <ArrowRight size={14} /></button>
        <button className="icon-button small" onClick={(e) => { e.stopPropagation(); setExpanded(!expanded) }} title="更多操作">
          {expanded ? <ChevronLeft size={16} /> : <ChevronRight size={16} />}
        </button>
      </div>
    </div>
    {expanded && (
      <div className="meeting-expanded">
        <div className="meeting-expanded-actions">
          <button className="soft-button" onClick={() => { setShowTicketModal(true); setTicketInfo(null); setTicketError(''); setTicketIdentity(''); setTicketName(''); setTicketExpire('3600'); setTicketType(1); setCanPublish(true); setCanSubscribe(true); setCanPublishData(true); setCanPublishSources(['camera', 'microphone', 'screen_share']) }}>
            <Clipboard size={15} /> 生成邀请票据
          </button>
        </div>
      </div>
    )}
    {showDetailModal && (
      <div className="modal-overlay" onClick={() => setShowDetailModal(false)}>
        <div className="modal-content modal-lg" onClick={(e) => e.stopPropagation()}>
          <div className="modal-header">
            <h3>会议详情</h3>
            <button className="icon-button" onClick={() => setShowDetailModal(false)}><X size={18} /></button>
          </div>
          <div className="modal-body">
            {detailLoading ? (
              <div className="detail-loading"><RefreshCw size={20} className="spin" /><span>加载中...</span></div>
            ) : detailError ? (
              <div className="detail-empty">{detailError}</div>
            ) : detailData ? (
              <div className="detail-content">
                <div className="detail-section">
                  <h4>基本信息</h4>
                  <div className="detail-grid">
                    <div className="detail-item"><label>会议标题</label><span>{detailData.meeting.title || '未命名会议'}</span></div>
                    <div className="detail-item"><label>会议号</label><span className="mono">{detailData.meeting.meetingNo}</span></div>
                    {detailData.meeting.meetingCode && <div className="detail-item"><label>会议码</label><span className="meeting-code">{formatMeetingCode(detailData.meeting.meetingCode)}</span></div>}
                    <div className="detail-item"><label>状态</label><span className={`status-pill ${statusMeta(Number(detailData.meeting.status)).className}`}>{statusMeta(Number(detailData.meeting.status)).label}</span></div>
                    <div className="detail-item"><label>创建人</label><span>{detailData.meeting.createUser || '-'}</span></div>
                    <div className="detail-item"><label>机构</label><span>{detailData.meeting.deptCode || '-'}</span></div>
                    <div className="detail-item"><label>创建时间</label><span>{timeLabel(detailData.meeting.createTime)}</span></div>
                    <div className="detail-item"><label>开始时间</label><span>{detailData.meeting.startTime ? timeLabel(detailData.meeting.startTime) : '未开始'}</span></div>
                    <div className="detail-item"><label>结束时间</label><span>{detailData.meeting.endTime ? timeLabel(detailData.meeting.endTime) : '进行中'}</span></div>
                    <div className="detail-item"><label>会议时长</label><span>{formatDuration(detailData.meeting.startTime, Number(detailData.meeting.status) === 3 ? detailData.meeting.endTime : undefined)}</span></div>
                  </div>
                </div>
                <div className="detail-section">
                  <h4>参会人 ({detailData.participants.length})</h4>
                  {detailData.participants.length > 0 ? (
                    <div className="detail-participants">
                      {detailData.participants.map((p) => (
                        <div key={p.identity} className="participant-item">
                          <span className="avatar tiny">{initials(p.name)}</span>
                          <div className="participant-info">
                            <b>{p.name || p.identity}</b>
                            <small>{p.identity}</small>
                          </div>
                          <span className={`member-status ${Number(p.status) === 1 ? 'online' : 'offline'}`}>
                            {Number(p.status) === 1 ? '在线' : '已离开'}
                          </span>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="detail-empty">暂无参会人记录</div>
                  )}
                </div>
              </div>
            ) : null}
          </div>
          <div className="modal-footer">
            <button className="button secondary" onClick={() => setShowDetailModal(false)}>关闭</button>
            {isActive && <button className="button primary" onClick={() => { setShowDetailModal(false); joinNow() }}>加入会议</button>}
          </div>
        </div>
      </div>
    )}
    {showTicketModal && (
      <div className="modal-overlay" onClick={() => setShowTicketModal(false)}>
        <div className="modal-content" onClick={(e) => e.stopPropagation()}>
          <div className="modal-header">
            <h3>生成邀请票据</h3>
            <button className="icon-button" onClick={() => setShowTicketModal(false)}><X size={18} /></button>
          </div>
          <div className="modal-body">
            {!ticketInfo ? <>
              <p className="modal-desc">为访客生成会议邀请，无需登录即可加入会议。</p>
              {ticketError && <div className="modal-error" role="alert"><X size={16} /><span>{ticketError}</span></div>}
              <label className="field-label">参会身份（可选）</label>
              <input className="text-input" value={ticketIdentity} onChange={(e) => setTicketIdentity(e.target.value)} placeholder="留空自动生成，例如：device-001" />
              <label className="field-label" style={{ marginTop: 12 }}>展示名称（可选）</label>
              <input className="text-input" value={ticketName} onChange={(e) => setTicketName(e.target.value)} placeholder="例如：1号安全帽" />
              <label className="field-label" style={{ marginTop: 12 }}>有效期（秒）</label>
              <input className="text-input" type="number" value={ticketExpire} onChange={(e) => setTicketExpire(e.target.value)} placeholder="3600" />
              <div className="permission-section">
                <h5>票据类型</h5>
                <div className="ticket-type-selector">
                  <label className={`ticket-type-option ${ticketType === 1 ? 'active' : ''}`} onClick={() => setTicketType(1)}>
                    <input type="radio" name="ticketType" checked={ticketType === 1} onChange={() => setTicketType(1)} />
                    <div className="ticket-type-content">
                      <b>一次性票据</b>
                      <small>消费后删除，只能使用一次</small>
                    </div>
                  </label>
                  <label className={`ticket-type-option ${ticketType === 2 ? 'active' : ''}`} onClick={() => setTicketType(2)}>
                    <input type="radio" name="ticketType" checked={ticketType === 2} onChange={() => setTicketType(2)} />
                    <div className="ticket-type-content">
                      <b>有效期票据</b>
                      <small>消费后保留至过期，可多次使用</small>
                    </div>
                  </label>
                </div>
              </div>
              <div className="permission-section">
                <h5>访客权限</h5>
                <label className="checkbox-label"><input type="checkbox" checked={canPublish} onChange={(e) => setCanPublish(e.target.checked)} />允许发布音视频</label>
                <label className="checkbox-label"><input type="checkbox" checked={canSubscribe} onChange={(e) => setCanSubscribe(e.target.checked)} />允许订阅音视频</label>
                <label className="checkbox-label"><input type="checkbox" checked={canPublishData} onChange={(e) => setCanPublishData(e.target.checked)} />允许发送消息/数据</label>
                <div className="source-permissions">
                  <div className="source-label">允许发布的轨道源</div>
                  <label className="checkbox-label small"><input type="checkbox" checked={canPublishSources.includes('camera')} onChange={() => toggleSource('camera')} />摄像头</label>
                  <label className="checkbox-label small"><input type="checkbox" checked={canPublishSources.includes('microphone')} onChange={() => toggleSource('microphone')} />麦克风</label>
                  <label className="checkbox-label small"><input type="checkbox" checked={canPublishSources.includes('screen_share')} onChange={() => toggleSource('screen_share')} />屏幕共享</label>
                </div>
              </div>
            </> : (
              <div className="ticket-result">
                <div className="ticket-result-header">
                  <span className="ticket-badge">{ticketInfo.ticketType === 2 ? '有效期票据' : '一次性票据'}</span>
                  <span className="ticket-expire-info">{ticketInfo.ticketType === 2 ? '可多次使用' : '仅限一次'}</span>
                </div>
                <div className="ticket-field">
                  <label>票据地址</label>
                  <code>{ticketInfo.joinUrl}</code>
                </div>
                <div className="ticket-field" style={{ marginTop: 10 }}>
                  <label>票据号</label>
                  <code>{ticketInfo.ticket}</code>
                </div>
                <div className="ticket-permissions-summary">
                  <h5>权限配置</h5>
                  <div className="perm-tags">
                    <span className={`perm-tag ${ticketInfo.canPublish ? 'active' : 'disabled'}`}>发布音视频</span>
                    <span className={`perm-tag ${ticketInfo.canSubscribe ? 'active' : 'disabled'}`}>订阅音视频</span>
                    <span className={`perm-tag ${ticketInfo.canPublishData ? 'active' : 'disabled'}`}>发送消息</span>
                  </div>
                  {ticketInfo.canPublishSources && ticketInfo.canPublishSources.length > 0 && (
                    <div className="source-tags">
                      <small>轨道源：</small>
                      {ticketInfo.canPublishSources.map(s => <span key={s} className="source-tag">{s === 'camera' ? '摄像头' : s === 'microphone' ? '麦克风' : s === 'screen_share' ? '屏幕共享' : s}</span>)}
                    </div>
                  )}
                </div>
                <div className="ticket-actions">
                  <button className="button primary compact" onClick={() => copyUrl(ticketInfo.joinUrl)}><Copy size={14} /> 复制地址</button>
                  <button className="button secondary compact" onClick={() => copyUrl(ticketInfo.ticket)}>复制票据号</button>
                </div>
              </div>
            )}
          </div>
          <div className="modal-footer">
            {ticketInfo ? (
              <>
                <button className="button secondary" onClick={() => { setTicketInfo(null); setTicketIdentity('') }}>再生成一个</button>
                <button className="button primary" onClick={() => setShowTicketModal(false)}>完成</button>
              </>
            ) : (
              <>
                <button className="button secondary" onClick={() => setShowTicketModal(false)}>取消</button>
                <button className="button primary" onClick={generateTicket} disabled={ticketLoading}>{ticketLoading ? '生成中...' : '生成票据'}</button>
              </>
            )}
          </div>
        </div>
      </div>
    )}
  </>
}

function registerEchoRpc(room: Room, notify: (message: string, tone?: Toast['tone']) => void) {
  try {
    room.localParticipant.registerRpcMethod('echo', async (data: any) => {
      const lp = room.localParticipant
      const info = { identity: lp.identity, name: lp.name, payload: data.payload, callerIdentity: data.callerIdentity }
      notify(`收到 RPC [${data.callerIdentity}]: ${data.payload}`, 'success')
      return JSON.stringify(info)
    })
  } catch { /* already registered */ }
}

function MeetingRoom({ meeting, perms, name, identity, guest, onLeave, notify }: { meeting: MeetingInfo; perms: JoinPerms; name: string; identity: string; guest: boolean; onLeave: () => void; notify: (message: string, tone?: Toast['tone']) => void }) {
  const [pane, setPane] = useState<'chat' | 'members' | 'manage'>('chat'); const [showPanel, setShowPanel] = useState(true)
  const [echoRegistered, setEchoRegistered] = useState(true)
  const togglePanel = () => setShowPanel((v) => !v)
  const room = useRoomContext()
  useEffect(() => { registerEchoRpc(room, notify); return () => { room.localParticipant.unregisterRpcMethod('echo') } }, [room, notify])
  const toggleEcho = () => { if (echoRegistered) { room.localParticipant.unregisterRpcMethod('echo'); setEchoRegistered(false); notify('已注销 Echo，本端 RPC 调用将返回 Method not supported', 'warning') } else { registerEchoRpc(room, notify); setEchoRegistered(true); notify('已注册 Echo', 'success') } }
  return <main className={`room-page ${showPanel ? '' : 'panel-collapsed'}`}><section className="room-stage"><div className="room-heading"><div><button className="back-button" onClick={() => window.confirm('确定离开会议？') && onLeave()}><ChevronLeft size={16} />{guest ? '离开会议' : '返回大厅'}</button><h1>{meeting.title || 'Live 会议'}</h1><div className="room-id">会议号 <button onClick={() => navigator.clipboard.writeText(meeting.meetingNo).then(() => notify('会议号已复制', 'success')).catch(() => notify('复制失败', 'error'))}><Copy size={13} />{meeting.meetingNo}</button></div></div><div className="room-heading-actions"><span className="live-indicator"><i />已连接</span></div></div><RoomAudioRenderer /><RoomContent perms={perms} onLeave={onLeave} notify={notify} /></section><aside className={`room-sidebar ${showPanel ? '' : 'collapsed'}`}><button className="sidebar-collapse-handle" title={showPanel ? '收起侧栏' : '展开侧栏'} onClick={togglePanel} aria-label={showPanel ? '收起侧栏' : '展开侧栏'} aria-expanded={showPanel}>{showPanel ? <ChevronRight size={15} /> : <ChevronLeft size={15} />}</button>{showPanel && <><nav className="room-tabs"><button className={pane === 'chat' ? 'active' : ''} onClick={() => setPane('chat')}><MessageSquare size={16} />群聊</button><button className={pane === 'members' ? 'active' : ''} onClick={() => setPane('members')}><Users size={16} />成员</button>{!guest && <button className={pane === 'manage' ? 'active' : ''} onClick={() => setPane('manage')}><Settings2 size={16} />管理</button>}</nav>{pane === 'chat' && <ChatPane meetingNo={meeting.meetingNo} identity={identity} name={name} guest={guest} perms={perms} notify={notify} />}{pane === 'members' && <MembersPane meetingNo={meeting.meetingNo} guest={guest} notify={notify} />}{pane === 'manage' && !guest && <ManagePane meetingNo={meeting.meetingNo} notify={notify} onEnd={onLeave} echoRegistered={echoRegistered} onToggleEcho={toggleEcho} />}</>}</aside></main>
}

function RoomContent({ perms, onLeave, notify }: { perms: JoinPerms; onLeave: () => void; notify: (message: string, tone?: Toast['tone']) => void }) {
  const videoTracks = useTracks([Track.Source.Camera, Track.Source.ScreenShare], { onlySubscribed: false })
  const participants = useParticipants()
  const audioOnly = participants.filter(p => {
    const hasVideo = p.getTrackPublication(Track.Source.Camera)?.isSubscribed || p.getTrackPublication(Track.Source.ScreenShare)?.isSubscribed
    const hasAudio = p.getTrackPublication(Track.Source.Microphone)?.isSubscribed
    return !hasVideo && hasAudio && !p.isLocal
  })
  const [selectedTrack, setSelectedTrack] = useState<TrackReferenceOrPlaceholder | null>(null)
  const [showThumbnails, setShowThumbnails] = useState(true)
  const [thumbnailSize, setThumbnailSize] = useState(1)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [fit, setFit] = useState<'contain' | 'cover' | 'fill'>('cover')
  const [zoom, setZoom] = useState(1)
  const mainVideoRef = useRef<HTMLDivElement>(null)
  const allTracks = [...videoTracks, ...audioOnly.map(p => ({ participant: p, source: Track.Source.Microphone, publication: p.getTrackPublication(Track.Source.Microphone) }))] as TrackReferenceOrPlaceholder[]
  useEffect(() => { if (!selectedTrack && allTracks.length > 0) { const screenShare = allTracks.find(t => t.source === Track.Source.ScreenShare); setSelectedTrack(screenShare || allTracks[0]) } }, [allTracks, selectedTrack])
  useEffect(() => { if (selectedTrack) { const stillExists = allTracks.some(t => t.participant.identity === selectedTrack.participant.identity && t.source === selectedTrack.source); if (!stillExists) setSelectedTrack(allTracks[0] || null) } }, [allTracks, selectedTrack]); const toggleFullscreen = async () => { if (!mainVideoRef.current) return; if (!document.fullscreenElement) { try { await mainVideoRef.current.requestFullscreen(); setIsFullscreen(true) } catch { notify('无法进入全屏模式', 'error') } } else { try { await document.exitFullscreen(); setIsFullscreen(false) } catch { notify('无法退出全屏模式', 'error') } } }; useEffect(() => { const handler = () => setIsFullscreen(!!document.fullscreenElement); document.addEventListener('fullscreenchange', handler); return () => document.removeEventListener('fullscreenchange', handler) }, []); const zoomBy = (delta: number) => setZoom((z) => Math.min(3, Math.max(1, Math.round((z + delta) * 100) / 100))); const cycleFit = () => setFit((f) => f === 'cover' ? 'contain' : f === 'contain' ? 'fill' : 'cover'); const fitLabel = fit === 'cover' ? '铺满' : fit === 'contain' ? '适应' : '拉伸';   const selectTrack = (track: TrackReferenceOrPlaceholder) => { setSelectedTrack(track); setZoom(1); setFit('cover') }
  return <div className="room-content-layout">{showThumbnails ? <div className={`thumbnail-strip thumbnail-size-${thumbnailSize}`}><div className="thumbnail-header"><div><span className="thumbnail-title">参会人</span><small>{allTracks.length} 人</small></div><div className="thumbnail-actions"><button className="icon-button small" onClick={() => setThumbnailSize((size) => Math.max(0, size - 1))} title="缩小参会人画面" aria-label="缩小参会人画面" disabled={thumbnailSize === 0}><ZoomOut size={14} /></button><button className="icon-button small" onClick={() => setThumbnailSize((size) => Math.min(2, size + 1))} title="放大参会人画面" aria-label="放大参会人画面" disabled={thumbnailSize === 2}><ZoomIn size={14} /></button><button className="icon-button small" onClick={() => setShowThumbnails(false)} title="收起参会人" aria-label="收起参会人"><ChevronUp size={15} /></button></div></div><div className="thumbnail-row">{allTracks.map((track) => <button key={`${track.participant.identity}-${track.source}`} className={`thumbnail-item ${selectedTrack?.participant.identity === track.participant.identity && selectedTrack?.source === track.source ? 'active' : ''}`} onClick={() => selectTrack(track)} title={`查看 ${track.participant.name || track.participant.identity}`}><TrackTile track={track} /><span className="thumbnail-name">{track.participant.name || track.participant.identity}{track.participant.isLocal ? ' · 我' : ''}</span></button>)}</div></div> : <button className="participants-restore" onClick={() => setShowThumbnails(true)}><Users size={15} /><span>参会人</span><b>{allTracks.length}</b><ChevronDown size={15} /></button>}<div className="main-video-area" ref={mainVideoRef}>{selectedTrack ? <TrackTile key={`${selectedTrack.participant.identity}-${selectedTrack.source}`} track={selectedTrack} fit={fit} zoom={zoom} /> : <div className="room-empty"><span className="empty-camera"><Video size={27} /></span><b>暂时没有可用视频</b><small>成员开启摄像头后会显示在这里</small></div>}<div className="video-toolbar"><button className="toolbar-button" onClick={() => zoomBy(-0.25)} title="缩小" disabled={zoom <= 1}><ZoomOut size={15} /></button><span className="toolbar-zoom">{Math.round(zoom * 100)}%</span><button className="toolbar-button" onClick={() => zoomBy(0.25)} title="放大" disabled={zoom >= 3}><ZoomIn size={15} /></button><button className="toolbar-button text" onClick={() => setZoom(1)} title="重置缩放">重置</button><button className="toolbar-button text" onClick={cycleFit} title="切换显示模式">{fitLabel}</button><button className="toolbar-button" onClick={toggleFullscreen} title={isFullscreen ? '退出全屏' : '全屏'}>{isFullscreen ? <Minimize2 size={15} /> : <Maximize2 size={15} />}</button></div></div><RoomControls perms={perms} onLeave={onLeave} notify={notify} /></div> }
function TrackTile({ track, fit = 'cover', zoom = 1 }: { track: TrackReferenceOrPlaceholder; fit?: 'contain' | 'cover' | 'fill'; zoom?: number }) {
  const title = track.participant.name || track.participant.identity
  const isVideoSubscribed = track.publication?.isSubscribed && track.publication.track && track.source !== Track.Source.Microphone
  const isAudioOnly = track.source === Track.Source.Microphone && !track.participant.getTrackPublication(Track.Source.Camera)?.isSubscribed && !track.participant.getTrackPublication(Track.Source.ScreenShare)?.isSubscribed
  return <div className={`track-tile ${isAudioOnly ? 'audio-only' : ''}`} data-source={track.source === Track.Source.ScreenShare ? 'screenShare' : 'camera'}>
    {isVideoSubscribed ? <VideoTrack trackRef={track as TrackReference} className="track-video" autoPlay muted={track.participant.isLocal} playsInline style={{ objectFit: fit, transform: zoom !== 1 ? `scale(${zoom})` : undefined }} /> : <div className={`track-offline ${isAudioOnly ? 'track-audio-only' : ''}`}><span className="avatar tiny">{initials(title)}</span><span>{title}</span><small>{isAudioOnly ? '语音通话中' : '视频未开启'}</small></div>}
    <div className="track-label"><span className="avatar tiny">{initials(title)}</span><span>{title}{track.participant.isLocal ? ' · 我' : ''}</span></div>
  </div>
}
function RoomControls({ perms, onLeave, notify }: { perms: JoinPerms; onLeave: () => void; notify: (message: string, tone?: Toast['tone']) => void }) { const { localParticipant } = useLocalParticipant(); const camera = localParticipant?.isCameraEnabled ?? false; const mic = localParticipant?.isMicrophoneEnabled ?? false; const screen = localParticipant?.isScreenShareEnabled ?? false; const canPublish = perms.canPublish; const sources = perms.canPublishSources; const allowSource = (source: string) => !sources || sources.includes(source); const allowCamera = canPublish && allowSource('camera'); const allowMic = canPublish && allowSource('microphone'); const allowScreen = canPublish && allowSource('screen_share'); const toggle = async (kind: 'camera' | 'mic' | 'screen') => { try { if (kind === 'camera' && !allowCamera) return notify('当前身份无摄像头发布权限', 'warning'); if (kind === 'mic' && !allowMic) return notify('当前身份无麦克风发布权限', 'warning'); if (kind === 'screen' && !allowScreen) return notify('当前身份无屏幕共享权限', 'warning'); if (kind === 'camera') await localParticipant.setCameraEnabled(!camera); if (kind === 'mic') await localParticipant.setMicrophoneEnabled(!mic); if (kind === 'screen') await localParticipant.setScreenShareEnabled(!screen); } catch (e) { notify(`媒体设备不可用：${(e as Error).message}`, 'error') } }; return <div className="room-controls"><button className={`control ${camera ? 'on' : ''}`} disabled={!allowCamera} onClick={() => toggle('camera')} title="摄像头"><Camera size={19} /><span>{camera ? '关闭视频' : '打开视频'}</span></button><button className={`control ${mic ? 'on' : ''}`} disabled={!allowMic} onClick={() => toggle('mic')} title="麦克风"><Mic size={19} /><span>{mic ? '关闭语音' : '打开语音'}</span></button><button className={`control ${screen ? 'on' : ''}`} disabled={!allowScreen} onClick={() => toggle('screen')} title="共享屏幕"><MonitorUp size={19} /><span>{screen ? '停止共享' : '共享屏幕'}</span></button><button className="control leave" onClick={() => window.confirm('确定离开会议？') && onLeave()} title="离开会议"><LogOut size={19} /><span>离开会议</span></button></div> }

function ChatPane({ meetingNo, identity, name, guest, perms, notify }: { meetingNo: string; identity: string; name: string; guest: boolean; perms: JoinPerms; notify: (message: string, tone?: Toast['tone']) => void }) { const room = useRoomContext(); const [messages, setMessages] = useState<MeetingMessage[]>([]); const [text, setText] = useState(''); const [loading, setLoading] = useState(true); const [sending, setSending] = useState(false); useEffect(() => { if (!guest) api.listMessages(meetingNo).then((data) => setMessages(data.messages || [])).catch(() => undefined).finally(() => setLoading(false)); else setLoading(false) }, [guest, meetingNo]); useEffect(() => { const receive = (payload: Uint8Array, participant?: { identity?: string; name?: string }, _kind?: unknown, topic?: string) => { if (topic !== 'lk.chat' && topic !== 'chat') return; try { const message = JSON.parse(new TextDecoder().decode(payload)); if (!message || typeof message.messageId !== 'string' || typeof message.content !== 'string') return; setMessages((current) => current.some((item) => item.messageId === message.messageId) ? current : [...current, { messageId: message.messageId, content: message.content, messageType: typeof message.messageType === 'string' ? message.messageType : 'text', senderId: participant?.identity || '', senderName: participant?.name || participant?.identity || '系统', createTime: '刚刚' }]) } catch { /* ignore malformed room data */ } }; room.on(RoomEvent.DataReceived, receive); return () => { room.off(RoomEvent.DataReceived, receive) } }, [room]); 	const send = async () => { const content = text.trim(); if (!content || sending) return; if (!perms.canPublishData) return notify('当前身份无消息发送权限', 'warning'); const localId: string = crypto.randomUUID?.() || `msg-${Date.now()}`; let messageId = localId; setSending(true); try { if (!guest) { const reply = await api.reportMessage({ meetingNo, content, messageType: 'text' }); messageId = reply.messageId || localId } const bytes = new TextEncoder().encode(JSON.stringify({ messageId, content, messageType: 'text' })); await room.localParticipant.publishData(bytes, { reliable: true, topic: 'lk.chat' }); setMessages((current) => [...current, { messageId, content, messageType: 'text', senderId: identity, senderName: name || identity, createTime: '刚刚' }]); setText('') } catch (e) { notify(`消息发送失败：${(e as Error).message}`, 'error') } finally { setSending(false) } }; return <div className="side-content chat-content"><div className="side-intro"><div><span className="eyebrow">会议消息</span><h3>群聊</h3></div><span className="live-badge"><i />Live</span></div><div className="chat-list">{loading ? <div className="side-empty">加载历史消息…</div> : messages.length ? messages.map((message) => <div key={message.messageId} className={`chat-item ${message.senderId === identity ? 'mine' : ''}`}><span className="avatar tiny">{initials(message.senderName || message.senderId)}</span><div><div className="chat-meta"><b>{message.senderId === identity ? '我' : message.senderName || message.senderId}</b><time>{message.createTime}</time></div><p>{message.content}</p></div></div>) : <div className="side-empty"><MessageSquare size={20} />还没有消息，打个招呼吧</div>}</div><div className="chat-composer"><input value={text} onChange={(e) => setText(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && send()} placeholder="输入消息，按 Enter 发送" /><button onClick={send} disabled={!perms.canPublishData || sending} title={perms.canPublishData ? '发送' : '当前身份无消息发送权限'}><Send size={17} /></button></div></div> }

function MembersPane({ meetingNo, guest, notify }: { meetingNo: string; guest: boolean; notify: (message: string, tone?: Toast['tone']) => void }) {
  const participants = useParticipants()
  const [history, setHistory] = useState<ParticipantInfo[]>([])
  const load = () => api.listParticipants(meetingNo).then((data) => setHistory(data.participants || [])).catch((e) => notify((e as Error).message, 'error'))
  useEffect(() => { if (!guest) load() }, [guest, meetingNo])
  const liveMembers = participants.map((participant) => ({ identity: participant.identity, name: participant.name || participant.identity, online: true }))
  const onlineIds = new Set(participants.map(p => p.identity))
  const list = guest ? liveMembers : [...history.map(m => ({ ...m, online: onlineIds.has(m.identity) })), ...liveMembers.filter((member) => !history.some((item) => item.identity === member.identity))]
  const onlineCount = list.filter(m => m.online).length
  return <div className="side-content"><div className="side-intro"><div><span className="eyebrow">实时状态</span><h3>成员 <em>{onlineCount}/{list.length}</em></h3></div>{!guest && <button className="icon-button" title="刷新成员" onClick={load}><RefreshCw size={16} /></button>}</div><div className="member-list">{list.length ? list.map((member) => <div className={`member-item ${member.online ? '' : 'offline'}`} key={member.identity}><span className="avatar tiny">{initials(member.name)}</span><div className="member-copy"><b>{member.name}</b><small>{member.identity}</small></div><span className={`member-status ${member.online ? 'online' : 'offline'}`}>{member.online ? '在线' : '离线'}</span></div>) : <div className="side-empty"><Users size={20} />等待成员加入</div>}</div></div>
}

function ManagePane({ meetingNo, notify, onEnd, echoRegistered, onToggleEcho }: { meetingNo: string; notify: (message: string, tone?: Toast['tone']) => void; onEnd: () => void; echoRegistered: boolean; onToggleEcho: () => void }) { const participants = useParticipants(); const [target, setTarget] = useState(''); const [identity, setIdentity] = useState(''); const [inviteName, setInviteName] = useState(''); const [expire, setExpire] = useState('3600'); const [ticket, setTicket] = useState<TicketReply | null>(null); const [members, setMembers] = useState<ParticipantInfo[]>([]); const [generating, setGenerating] = useState(false); const [canPublish, setCanPublish] = useState(true); const [canSubscribe, setCanSubscribe] = useState(true); const [canPublishData, setCanPublishData] = useState(true); const [canPublishSources, setCanPublishSources] = useState<string[]>(['camera', 'microphone', 'screen_share']); const load = () => api.listParticipants(meetingNo).then((data) => setMembers(data.participants || [])).catch(() => undefined); useEffect(() => { load() }, [meetingNo]); const onlineIds = new Set(participants.map(p => p.identity)); const onlineMembers = [...new Map<string, ParticipantInfo>([...participants.map((p) => [p.identity, { identity: p.identity, name: p.name || p.identity, status: 1, joinTime: '', leftTime: '' }] as [string, ParticipantInfo]), ...members.filter((m) => onlineIds.has(m.identity)).map((m) => [m.identity, m] as [string, ParticipantInfo])]).values()]; const mute = async (kind: 'audio' | 'video') => { if (!target) return notify('请先选择成员', 'warning'); try { await api.muteParticipant(meetingNo, target, true, kind); notify('管理指令已发送', 'success') } catch (e) { notify((e as Error).message, 'error') } }; const kick = async () => { if (!target) return notify('请先选择成员', 'warning'); if (!window.confirm(`确定将 ${target} 移出会议？`)) return; try { await api.kickParticipant(meetingNo, target); notify('成员已移出会议', 'success'); load() } catch (e) { notify((e as Error).message, 'error') } }; const toggleSource = (source: string) => { setCanPublishSources(current => current.includes(source) ? current.filter(s => s !== source) : [...current, source]) }; const generate = async () => { if (generating) return; const guestIdentity = identity.trim() || ('访客_' + Math.random().toString(36).slice(2, 8)); setGenerating(true); try { const reply = await api.generateTicket(meetingNo, guestIdentity, inviteName.trim(), Number(expire) || 3600, canPublish, canSubscribe, canPublishData, canPublishSources); setTicket(reply) } catch (e) { notify((e as Error).message, 'error') } finally { setGenerating(false) } }; const share = () => { if (!ticket) return; const url = `${location.origin}/guest?ticket=${encodeURIComponent(ticket.ticket)}`; navigator.clipboard.writeText(url).then(() => notify('邀请链接已复制', 'success')).catch(() => notify('复制失败', 'error')) }; return <div className="side-content manage-content"><div className="side-intro"><div><span className="eyebrow">会议管理</span><h3>会议管理</h3></div><span className="admin-badge"><ShieldCheck size={14} />已授权</span></div><section className="manage-section"><h4>电话拨号</h4><DialPad meetingNo={meetingNo} notify={notify} /></section><section className="manage-section"><h4>成员控制</h4><select className="text-input" value={target} onChange={(e) => setTarget(e.target.value)}><option value="">选择在线成员</option>{onlineMembers.map((member) => <option key={member.identity} value={member.identity}>{member.name || member.identity} ({member.identity})</option>)}</select><div className="manage-actions"><button className="soft-button" onClick={() => mute('audio')}><Mic size={15} />静音语音</button><button className="soft-button" onClick={() => mute('video')}><Video size={15} />关闭视频</button><button className="soft-button danger" onClick={kick}><UserRound size={15} />移出会议</button></div></section><section className="manage-section"><h4>临时会议邀请</h4><input className="text-input" value={identity} onChange={(e) => setIdentity(e.target.value)} placeholder="参会身份（可选，留空自动生成）" /><input className="text-input" value={inviteName} onChange={(e) => setInviteName(e.target.value)} placeholder="显示名称（可选）" /><div className="invite-inline"><input className="text-input" type="number" min="60" max="86400" value={expire} onChange={(e) => setExpire(e.target.value)} /><button className="button secondary" onClick={generate} disabled={generating}>{generating ? '生成中…' : '生成邀请'}</button></div><div className="permission-section"><h5>访客权限</h5><label className="checkbox-label"><input type="checkbox" checked={canPublish} onChange={(e) => setCanPublish(e.target.checked)} />可发布音视频</label><label className="checkbox-label"><input type="checkbox" checked={canSubscribe} onChange={(e) => setCanSubscribe(e.target.checked)} />可订阅音视频</label><label className="checkbox-label"><input type="checkbox" checked={canPublishData} onChange={(e) => setCanPublishData(e.target.checked)} />可发送消息</label><div className="source-permissions"><span className="source-label">可发布轨道：</span><label className="checkbox-label small"><input type="checkbox" checked={canPublishSources.includes('camera')} onChange={() => toggleSource('camera')} />摄像头</label><label className="checkbox-label small"><input type="checkbox" checked={canPublishSources.includes('microphone')} onChange={() => toggleSource('microphone')} />麦克风</label><label className="checkbox-label small"><input type="checkbox" checked={canPublishSources.includes('screen_share')} onChange={() => toggleSource('screen_share')} />屏幕共享</label></div></div>{ticket && <div className="ticket-result"><span>有效期至 {ticket.expireTime}</span><code>{`${location.origin}/guest?ticket=${encodeURIComponent(ticket.ticket)}`}</code><button onClick={share}><Copy size={14} />复制邀请链接</button></div>}</section><RealtimeTools meetingNo={meetingNo} target={target} notify={notify} echoRegistered={echoRegistered} onToggleEcho={onToggleEcho} /><section className="manage-section"><h4>会议状态</h4><button className="end-meeting" onClick={() => window.confirm('结束后所有成员都会离开会议，是否继续？') && api.endMeeting(meetingNo).then(() => { notify('会议已结束', 'success'); onEnd() }).catch((e) => notify((e as Error).message, 'error'))}><X size={15} />结束整个会议</button></section></div> }

function RealtimeTools({ meetingNo, target, notify, echoRegistered, onToggleEcho }: { meetingNo: string; target: string; notify: (message: string, tone?: Toast['tone']) => void; echoRegistered: boolean; onToggleEcho: () => void }) { const [payload, setPayload] = useState('hello data'); const [sending, setSending] = useState(false); const [rpcing, setRpcing] = useState(false); const sendDataViaApi = async () => { if (!payload.trim() || sending) return; setSending(true); try { const localId = crypto.randomUUID?.() || `data-${Date.now()}`; const message = { messageId: localId, content: payload, messageType: 'data' }; const bytes = new TextEncoder().encode(JSON.stringify(message)); await api.sendMeetingData(meetingNo, 'lk.chat', bytesToBase64(bytes)); notify('Data 已发送 (API)', 'success') } catch (e) { notify(`Data 发送失败：${(e as Error).message}`, 'error') } finally { setSending(false) } }; const performRpcViaApi = async () => { if (!target) return notify('请先选择成员', 'warning'); if (rpcing) return; setRpcing(true); try { const reply = await api.performRpc(meetingNo, target, payload); notify(`响应：${reply.response || '无内容'}`, 'success') } catch (e) { notify(`响应失败：${(e as Error).message}`, 'error') } finally { setRpcing(false) } }; return <section className="manage-section"><h4>Data / RPC</h4><input className="text-input" value={payload} onChange={(e) => setPayload(e.target.value)} placeholder="消息内容" /><div className="manage-actions"><button className="soft-button" disabled={sending} onClick={sendDataViaApi}><Database size={15} />{sending ? '发送中...' : 'Send Data (API)'}</button><button className="soft-button" disabled={rpcing} onClick={performRpcViaApi}><MoreHorizontal size={15} />{rpcing ? '调用中...' : 'Perform RPC (API)'}</button></div><div className="manage-actions" style={{marginTop: 10}}><button className={`soft-button ${echoRegistered ? 'danger' : ''}`} onClick={onToggleEcho} title="所有参会人入会时已自动注册 echo（含访客），此处可手动注销以演示失败路径"><MoreHorizontal size={15} />{echoRegistered ? '注销 Echo (SDK)' : '注册 Echo (SDK)'}</button></div></section> }

const consumedTickets = new Set<string>()

function GuestView({ ticket, join, name, identity, onJoin, onLeave, notify }: { ticket: string | null; join: { token: string; meeting: MeetingInfo; perms: JoinPerms } | null; name: string; identity: string; onJoin: (ticket: string) => Promise<boolean>; onLeave: () => void; notify: (message: string, tone?: Toast['tone']) => void }) {
  const [loading, setLoading] = useState(false)
  const [failed, setFailed] = useState(false)
  const [retryCount, setRetryCount] = useState(0)
  useEffect(() => {
    if (!ticket || join || loading || consumedTickets.has(ticket)) return
    consumedTickets.add(ticket)
    setLoading(true)
    setFailed(false)
    onJoin(ticket).then((ok) => { if (!ok) { consumedTickets.delete(ticket); setFailed(true) } }).finally(() => setLoading(false))
  }, [ticket, join, loading, retryCount, onJoin])
  if (join) {
    return <><Topbar name={name} identity={identity} guest onLogout={onLeave} /><LiveKitRoom serverUrl={wsUrl()} token={join.token} connect audio={true} video={true} onDisconnected={onLeave}><MeetingRoom meeting={join.meeting} perms={join.perms} name={name} identity={identity} guest onLeave={onLeave} notify={notify} /></LiveKitRoom></>
  }
  return <main className="guest-gate"><div className="surface guest-gate-card"><div className="brand-lockup"><span className="brand-mark">L</span><span>Live 视频会议</span></div><div className="guest-gate-copy"><span className="eyebrow">邀请访客</span><h2>加入会议</h2><p>{loading ? '正在验证票据，请稍候…' : failed ? '票据无效、已使用或已过期，请联系会议主持人重新生成。' : '正在准备会议…'}</p></div>{loading ? <div className="guest-gate-loading"><RefreshCw size={18} className="spin" /><span>正在连接会议</span></div> : failed ? <button className="button primary wide" onClick={() => { consumedTickets.delete(ticket || ''); setRetryCount((n) => n + 1) }}>重新尝试 <ArrowRight size={15} /></button> : null}</div></main>
}

function bytesToBase64(bytes: Uint8Array) { let binary = ''; for (let index = 0; index < bytes.length; index += 0x8000) binary += String.fromCharCode(...bytes.subarray(index, index + 0x8000)); return btoa(binary) }

type SipProviderForm = { code: string; name: string; address: string; numbers: string; authUsername: string; authPassword: string; status: number }

const emptySipProviderForm: SipProviderForm = { code: '', name: '', address: '', numbers: '', authUsername: '', authPassword: '', status: 1 }

function SipProviderPanel({ notify }: { notify: (message: string, tone?: Toast['tone']) => void }) {
  const [providers, setProviders] = useState<SipProviderInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState('')
  const [saving, setSaving] = useState(false)
  const [mutatingProviderId, setMutatingProviderId] = useState<string | null>(null)
  const [editing, setEditing] = useState<SipProviderInfo | null>(null)
  const [form, setForm] = useState<SipProviderForm>(emptySipProviderForm)
  const [showForm, setShowForm] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setLoadError('')
    try {
      const data = await api.listSipProviders()
      setProviders(data.providers || [])
    } catch (e) {
      const message = (e as Error).message || '加载供应商失败'
      setLoadError(message)
      notify(message, 'error')
    } finally {
      setLoading(false)
    }
  }, [notify])
  useEffect(() => { load() }, [load])

  const openCreate = () => { setEditing(null); setForm(emptySipProviderForm); setShowForm(true) }
  const openEdit = (provider: SipProviderInfo) => {
    setEditing(provider)
    setForm({ code: provider.code, name: provider.name, address: provider.address, numbers: (provider.numbers || []).join('\n'), authUsername: '', authPassword: '', status: provider.status })
    setShowForm(true)
  }
  const closeForm = () => { if (!saving) { setShowForm(false); setEditing(null) } }
  const save = async () => {
    if (saving) return
    const numbers = form.numbers.split(/[\n,，]+/).map((value) => value.trim()).filter(Boolean)
    if (!form.code.trim() || !form.name.trim() || !form.address.trim() || numbers.length === 0) return notify('请填写编码、名称、服务器地址和至少一个主叫号码', 'warning')
    setSaving(true)
    try {
      if (editing) {
        await api.updateSipProvider({ id: editing.id, name: form.name.trim(), address: form.address.trim(), numbers, authUsername: form.authUsername.trim() || undefined, authPassword: form.authPassword || undefined, status: form.status })
        notify('供应商已更新', 'success')
      } else {
        await api.createSipProvider({ code: form.code.trim(), name: form.name.trim(), address: form.address.trim(), numbers, authUsername: form.authUsername.trim() || undefined, authPassword: form.authPassword || undefined })
        notify('供应商已创建', 'success')
      }
      setShowForm(false)
      setEditing(null)
      await load()
    } catch (e) {
      notify((e as Error).message, 'error')
    } finally {
      setSaving(false)
    }
  }
  const toggleStatus = async (provider: SipProviderInfo) => {
    if (mutatingProviderId) return
    setMutatingProviderId(provider.id)
    try {
      await api.updateSipProvider({ id: provider.id, status: provider.status === 1 ? 2 : 1 })
      notify(provider.status === 1 ? '供应商已禁用' : '供应商已启用', 'success')
      await load()
    } catch (e) {
      notify((e as Error).message, 'error')
    } finally {
      setMutatingProviderId(null)
    }
  }
  const remove = async (provider: SipProviderInfo) => {
    if (mutatingProviderId) return
    if (!window.confirm(`确定删除供应商“${provider.name}”？`)) return
    setMutatingProviderId(provider.id)
    try {
      await api.deleteSipProvider(provider.id)
      notify('供应商已删除', 'success')
      await load()
    } catch (e) {
      notify((e as Error).message, 'error')
    } finally {
      setMutatingProviderId(null)
    }
  }

  return <section className="surface provider-panel"><div className="section-heading"><div><span className="eyebrow">SIP PROVIDERS</span><div className="section-title-line"><h2>供应商管理</h2><span className="section-count">{providers.length} 个</span></div><p>配置外呼线路。认证密码只在提交时发送，不会在列表中展示。</p></div><div className="provider-heading-actions"><button className="icon-button" title="刷新供应商" onClick={load} disabled={loading}><RefreshCw size={17} className={loading ? 'spin' : ''} /></button><button className="button primary compact" onClick={openCreate} disabled={Boolean(mutatingProviderId)}><Plus size={15} />新增供应商</button></div></div><div className="provider-list">{loading ? <div className="empty-state"><RefreshCw size={18} className="spin" />加载供应商中…</div> : loadError ? <div className="inline-state error" role="alert"><span>供应商加载失败：{loadError}</span><button className="text-button" onClick={load}>重试</button></div> : providers.length === 0 ? <div className="empty-state"><Settings2 size={20} />暂无 SIP 供应商，请先新增线路配置</div> : providers.map((provider) => { const mutating = mutatingProviderId === provider.id; const numbers = provider.numbers || []; return <article className="provider-row" key={provider.id}><div className="provider-main"><div className="provider-name"><b>{provider.name}</b><code>{provider.code}</code><span className={`status-pill ${provider.status === 1 ? 'live' : 'ended'}`}>{provider.status === 1 ? '已启用' : '已禁用'}</span></div><span>{provider.address}</span><small>主叫号码：{numbers.length ? numbers.join('、') : '未配置'} · 创建于 {timeLabel(provider.createTime)}</small>{provider.sipTrunkId && <small>Trunk ID: <code>{provider.sipTrunkId}</code></small>}</div><div className="provider-actions"><button className="text-button" disabled={Boolean(mutatingProviderId)} onClick={() => openEdit(provider)}>编辑</button><button className="text-button" disabled={Boolean(mutatingProviderId)} onClick={() => toggleStatus(provider)}>{mutating ? '处理中…' : provider.status === 1 ? '禁用' : '启用'}</button><button className="text-button danger" disabled={Boolean(mutatingProviderId)} onClick={() => remove(provider)}>删除</button></div></article> })}</div>{showForm && <div className="modal-overlay" onClick={closeForm}><div className="modal-content" onClick={(e) => e.stopPropagation()}><div className="modal-header"><h3>{editing ? '编辑 SIP 供应商' : '新增 SIP 供应商'}</h3><button className="icon-button" onClick={closeForm} disabled={saving}><X size={18} /></button></div><div className="modal-body provider-form"><label className="field-label">供应商编码</label><input className="text-input" value={form.code} disabled={Boolean(editing) || saving} onChange={(e) => setForm({ ...form, code: e.target.value })} placeholder="例如：telnyx" /><label className="field-label">供应商名称</label><input className="text-input" value={form.name} disabled={saving} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="例如：Telnyx 生产线路" /><label className="field-label">SIP 服务器地址</label><input className="text-input" value={form.address} disabled={saving} onChange={(e) => setForm({ ...form, address: e.target.value })} placeholder="例如：sip.telnyx.com" /><label className="field-label">主叫号码</label><textarea className="text-input" value={form.numbers} disabled={saving} onChange={(e) => setForm({ ...form, numbers: e.target.value })} placeholder="每行一个号码，也可使用逗号分隔" /><div className="provider-form-grid"><div><label className="field-label">认证用户名</label><input className="text-input" value={form.authUsername} disabled={saving} onChange={(e) => setForm({ ...form, authUsername: e.target.value })} placeholder={editing ? '留空保持不变' : '可选'} /></div><div><label className="field-label">认证密码</label><input className="text-input" type="password" value={form.authPassword} disabled={saving} autoComplete="new-password" onChange={(e) => setForm({ ...form, authPassword: e.target.value })} placeholder={editing ? '留空保持不变' : '可选'} /></div></div>{editing && <><label className="field-label">状态</label><select className="text-input" value={form.status} disabled={saving} onChange={(e) => setForm({ ...form, status: Number(e.target.value) })}><option value={1}>启用</option><option value={2}>禁用</option></select></>}</div><div className="modal-footer"><button className="button secondary" onClick={closeForm} disabled={saving}>取消</button><button className="button primary" onClick={save} disabled={saving}>{saving ? '保存中…' : '保存'}</button></div></div></div>}</section>
}

function DialPad({ meetingNo, notify }: { meetingNo?: string; notify: (message: string, tone?: Toast['tone']) => void }) {
  const [callee, setCallee] = useState('')
  const [providers, setProviders] = useState<SipProviderInfo[]>([])
  const [providerCode, setProviderCode] = useState('')
  const [dialing, setDialing] = useState(false)
  const [loadingProviders, setLoadingProviders] = useState(true)
  const [providerError, setProviderError] = useState('')
  const [dialError, setDialError] = useState('')
  const [lastCall, setLastCall] = useState<{ number: string; callId: string; meetingNo?: string } | null>(null)

  const loadProviders = useCallback(async () => {
    setLoadingProviders(true)
    setProviderError('')
    try {
      const data = await api.listSipProviders()
      const enabled = (data.providers || []).filter((provider) => provider.status === 1)
      setProviders(enabled)
      setProviderCode((current) => enabled.some((provider) => provider.code === current) ? current : enabled.length === 1 ? enabled[0].code : '')
    } catch (e) {
      const message = (e as Error).message || '加载供应商失败'
      setProviders([])
      setProviderCode('')
      setProviderError(message)
      notify(message, 'error')
    } finally {
      setLoadingProviders(false)
    }
  }, [notify])
  useEffect(() => { loadProviders() }, [loadProviders])

  const dial = async () => {
    if (dialing) return
    const number = callee.trim()
    if (!number) return notify('请输入被叫号码', 'warning')
    if (!providerCode) return notify('请选择供应商', 'warning')
    setDialError('')
    setDialing(true)
    try {
      const result = await api.dialSip({ calleeNumber: number, meetingNo, providerCode })
      notify(`已呼叫 ${number}，通话ID: ${result.sipCallId}`, 'success')
      setLastCall({ number, callId: result.sipCallId, meetingNo: result.meeting?.meetingNo })
      setCallee('')
    } catch (e) {
      const message = (e as Error).message || '拨号失败，请稍后重试'
      setDialError(message)
      notify(message, 'error')
    } finally {
      setDialing(false)
    }
  }

  return <div className="dial-pad">
    <div className="dial-fields"><select className="text-input" value={providerCode} onChange={(e) => setProviderCode(e.target.value)} disabled={loadingProviders || providers.length === 0}>
      <option value="">{loadingProviders ? '加载供应商中…' : providers.length === 0 ? '没有可用供应商' : '选择供应商'}</option>
      {providers.map((p) => <option key={p.code} value={p.code}>{p.name} ({p.code})</option>)}
    </select><input className="text-input" value={callee} onChange={(e) => setCallee(e.target.value)} placeholder="输入被叫电话号码" onKeyDown={(e) => e.key === 'Enter' && dial()} /></div>
    <button className="button primary wide" onClick={dial} disabled={dialing || !callee.trim() || !providerCode}>
      <Phone size={15} />{dialing ? '呼叫中…' : '拨号'}
    </button>
    {providerError && <div className="inline-state error" role="alert"><span>供应商加载失败：{providerError}</span><button className="text-button" onClick={loadProviders} disabled={loadingProviders}>重试</button></div>}
    {dialError && <div className="inline-state error" role="alert"><span>拨号失败：{dialError}</span></div>}
    {lastCall && <div className="dial-result"><Check size={15} /><span>已呼叫 {lastCall.number}</span><small>{lastCall.meetingNo ? `会议 ${lastCall.meetingNo} · ` : '独立外呼 · '}Call ID {lastCall.callId}</small></div>}
  </div>
}
