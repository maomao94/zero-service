import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { LiveKitRoom, RoomAudioRenderer, TrackReferenceOrPlaceholder, useLocalParticipant, useParticipants, useRoomContext, useTracks } from '@livekit/components-react'
import { RoomEvent, Track } from 'livekit-client'
import { Archive, ArrowRight, Camera, Check, ChevronDown, ChevronLeft, ChevronRight, ChevronUp, Clipboard, Copy, Database, DoorOpen, History, LayoutGrid, LogOut, Maximize2, MessageSquare, Mic, Minimize2, MonitorUp, MoreHorizontal, PanelRight, Plus, RefreshCw, Search, Send, Settings2, Shield, ShieldCheck, Sparkles, UserRound, Users, Video, X, ZoomIn, ZoomOut } from 'lucide-react'
import { api, ApiError } from './lib/api'
import type { MeetingInfo, MeetingMessage, ParticipantInfo, TicketReply } from './types'

type Toast = { message: string; tone?: 'error' | 'success' | 'warning' }
type Screen = 'auth' | 'lobby' | 'room' | 'guest'
type JoinPerms = { canPublish: boolean; canSubscribe: boolean; canPublishData: boolean; canPublishSources: string[] | null }
type JoinState = { token: string; wsUrl: string; meeting: MeetingInfo; perms: JoinPerms }

function initials(name: string) { return name.trim().split(/\s+/).map((part) => part[0]).join('').slice(0, 2).toUpperCase() || 'L' }
function decodeToken(token: string) { try { const payload = token.split('.')[1]; return JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/'))) } catch { return {} } }
function decodeJoinPerms(token: string): JoinPerms { const claim = decodeToken(token) as Record<string, unknown>; const video = (claim.video || {}) as Record<string, unknown>; return { canPublish: video.canPublish === true, canSubscribe: video.canSubscribe !== false, canPublishData: video.canPublishData !== false, canPublishSources: Array.isArray(video.canPublishSources) ? (video.canPublishSources as string[]) : null } }
function timeLabel(value: string) { return value || '时间未记录' }
function statusMeta(status: number) { return status === 3 ? { label: '已结束', className: 'ended' } : status === 2 ? { label: '进行中', className: 'live' } : { label: '已创建', className: 'created' } }
function formatDuration(startTime: string, endTime?: string) { const start = new Date(startTime).getTime(); const end = endTime ? new Date(endTime).getTime() : Date.now(); const diff = Math.max(0, end - start); const hours = Math.floor(diff / 3600000); const minutes = Math.floor((diff % 3600000) / 60000); const seconds = Math.floor((diff % 60000) / 1000); if (hours > 0) return `${hours}时${minutes}分`; if (minutes > 0) return `${minutes}分${seconds}秒`; return `${seconds}秒` }

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

  const login = (nextToken: string) => { const claim = decodeToken(nextToken); if (claim.exp && Number(claim.exp) * 1000 < Date.now()) throw new ApiError('Token 已过期', 401); localStorage.setItem('live_jwt', nextToken); setToken(nextToken); setGuest(false); setScreen('lobby') }
  const logout = () => { localStorage.removeItem('live_jwt'); setToken(''); setJoin(null); setGuest(false); setScreen('auth') }
  const enterMeeting = async (meetingNo: string) => { const reply = await api.joinMeeting(meetingNo); setJoin({ ...reply, perms: { canPublish: reply.canPublish, canSubscribe: reply.canSubscribe, canPublishData: reply.canPublishData, canPublishSources: reply.canPublishSources || null } }); setScreen('room') }
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
    {screen === 'room' && join && <><Topbar name={name} identity={identity} deptCode={deptCode} guest={guest} onLogout={logout} /><LiveKitRoom serverUrl={join.wsUrl} token={join.token} connect audio={true} video={true} onDisconnected={() => { setJoin(null); setScreen(guest ? 'auth' : 'lobby') }}><MeetingRoom meeting={join.meeting} perms={join.perms} name={name} identity={identity} guest={guest} onLeave={() => { setJoin(null); setScreen(guest ? 'auth' : 'lobby') }} notify={notify} /></LiveKitRoom></>}
    {screen === 'guest' && <GuestView ticket={queryTicket} join={join} name={name} identity={identity} onJoin={joinByTicket} onLeave={() => { setJoin(null); setScreen('auth') }} notify={notify} />}
    {toast && <div className={`toast ${toast.tone || ''}`}><span>{toast.tone === 'success' ? <Check size={16} /> : toast.tone === 'error' ? <X size={16} /> : <Sparkles size={16} />}</span>{toast.message}</div>}
  </>
}

function AuthView({ initialToken, onLogin, notify }: { initialToken: string; onLogin: (token: string) => void; notify: (message: string, tone?: Toast['tone']) => void }) {
  const [value, setValue] = useState(initialToken); const [visible, setVisible] = useState(false); const [error, setError] = useState('');
  const submit = () => { if (!value.trim()) { setError('请输入 JWT Token'); return } try { onLogin(value.trim()) } catch (e) { setError((e as Error).message); notify((e as Error).message, 'error') } }
  return <main className="auth-page"><div className="auth-art"><div className="auth-orbit orbit-one" /><div className="auth-orbit orbit-two" /><div className="auth-art-copy"><span className="kicker">LIVE / MEETING</span><h1>让每一次会面<br /><em>清晰而从容。</em></h1><p>一个面向团队的实时会议空间，连接画面、声音、消息和协作。</p><div className="auth-art-foot"><span><ShieldCheck size={16} />端到端鉴权</span><span><Video size={16} />低延迟媒体</span></div></div></div><section className="auth-card"><div className="brand-lockup"><span className="brand-mark">L</span><span>Live 会议中心</span></div><div className="auth-heading"><span className="eyebrow">欢迎回来</span><h2>进入你的会议空间</h2><p>使用业务 JWT 登录，继续管理会议与参会者。</p></div><label className="field-label" htmlFor="jwt">JWT Token</label><div className="secret-input"><input id="jwt" value={value} onChange={(e) => setValue(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && submit()} type={visible ? 'text' : 'password'} placeholder="粘贴 JWT Token" autoComplete="off" /><button type="button" aria-label={visible ? '隐藏 token' : '显示 token'} onClick={() => setVisible(!visible)}>{visible ? '隐藏' : '显示'}</button></div>{error && <div className="form-error">{error}</div>}<button className="button primary wide" onClick={submit}>登录并继续 <ArrowRight size={17} /></button><p className="auth-note">Token 仅保存在当前浏览器，不会上传到第三方。</p></section></main>
}

function Topbar({ name, identity, deptCode, guest, onLogout }: { name: string; identity: string; deptCode?: string; guest?: boolean; onLogout: () => void }) { const displayName = name || identity || '用户'; return <header className="topbar"><div className="topbar-left"><span className="brand-mark">L</span><span className="topbar-title">Live 会议中心</span><span className="topbar-divider" /><span className="topbar-context">实时协作</span></div><div className="topbar-right"><div className="presence"><span className="presence-dot" />在线</div><div className="profile" title={`用户ID: ${identity}\n姓名: ${name}\n部门: ${deptCode || '未设置'}`}><span className="avatar small">{initials(displayName)}</span><span className="profile-copy"><b>{displayName}</b><small>{guest ? '邀请访客' : identity}</small></span></div><button className="icon-button" title={guest ? '退出会议' : '退出登录'} onClick={onLogout}><LogOut size={17} /></button></div></header> }

function LobbyView({ name, identity, onJoin, notify }: { name: string; identity: string; onJoin: (meetingNo: string) => Promise<void>; notify: (message: string, tone?: Toast['tone']) => void }) {
  const [title, setTitle] = useState(''); const [meetingNo, setMeetingNo] = useState(''); const [rows, setRows] = useState<MeetingInfo[]>([]); const [total, setTotal] = useState(0); const [mode, setMode] = useState<'mine' | 'all'>('mine'); const [status, setStatus] = useState('0'); const [search, setSearch] = useState(''); const [loading, setLoading] = useState(true);
  const greeting = (() => { const h = new Date().getHours(); if (h < 6) return '夜深了'; if (h < 12) return '早上好'; if (h < 14) return '中午好'; if (h < 18) return '下午好'; return '晚上好' })();
  const displayName = name || identity || '用户';
  const load = useCallback(async () => { setLoading(true); try { const params: Record<string, string | number> = { page: 1, pageSize: 20 }; if (status !== '0') params.status = status; if (search) params.title = search; const data = mode === 'mine' ? await api.listMyMeetings(1, 20) : await api.listMeetings(params); const filtered = mode === 'mine' ? (data.meetings || []).filter((meeting) => (!search || meeting.title.toLowerCase().includes(search.toLowerCase())) && (status === '0' || String(meeting.status) === status)) : (data.meetings || []); setRows(filtered); setTotal(mode === 'mine' ? filtered.length : data.total || 0) } catch (e) { notify((e as Error).message, 'error') } finally { setLoading(false) } }, [mode, notify, search, status]);
  useEffect(() => { load() }, [load]);
  const create = async () => { try { const data = await api.createMeeting(title.trim() || 'Live 会议'); await onJoin(data.meeting.meetingNo) } catch (e) { notify((e as Error).message, 'error') } }
  const join = async () => { if (!meetingNo.trim()) return notify('请输入会议号', 'warning'); try { await onJoin(meetingNo.trim()) } catch (e) { notify((e as Error).message, 'error') } }
  const endMeeting = async (meetingNo: string) => { try { await api.endMeeting(meetingNo); notify('会议已结束', 'success'); load() } catch (e) { notify((e as Error).message, 'error') } }
  return <main className="lobby-page"><div className="page-heading"><div><span className="eyebrow">工作台</span><h1>{greeting}，{displayName}</h1><p>准备好开始今天的协作了吗？</p></div><div className="heading-metric"><span className="metric-icon"><History size={17} /></span><div><b>{total}</b><small>我的会议</small></div></div></div><div className="lobby-layout"><section className="create-column"><div className="surface create-card"><div className="card-kicker"><span className="icon-badge teal"><Plus size={18} /></span><span>新建空间</span></div><h2>创建一场新会议</h2><p>会议空间会立即开放，你可以在房间内生成临时邀请。</p><label className="field-label" htmlFor="meeting-title">会议名称</label><input id="meeting-title" className="text-input" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="例如：产品评审 / 周会" onKeyDown={(e) => e.key === 'Enter' && create()} /><button className="button primary wide" onClick={create}>创建并进入 <ArrowRight size={16} /></button><div className="split-line"><span>或</span></div><div className="card-kicker"><span className="icon-badge amber"><DoorOpen size={17} /></span><span>加入空间</span></div><label className="field-label" htmlFor="meeting-no">会议号</label><input id="meeting-no" className="text-input" value={meetingNo} onChange={(e) => setMeetingNo(e.target.value)} placeholder="M20260902001" onKeyDown={(e) => e.key === 'Enter' && join()} /><button className="button secondary wide" onClick={join}>加入会议 <ArrowRight size={16} /></button></div><div className="tip-card"><Shield size={17} /><div><b>会议数据受保护</b><span>只有授权成员可以执行会议管理操作。</span></div></div></section><section className="surface history-card"><div className="section-heading"><div><span className="eyebrow">会议记录</span><h2>最近的会议</h2></div><button className="icon-button" title="刷新会议记录" onClick={load}><RefreshCw size={17} className={loading ? 'spin' : ''} /></button></div><div className="history-tabs"><button className={mode === 'mine' ? 'active' : ''} onClick={() => setMode('mine')}>我的会议</button><button className={mode === 'all' ? 'active' : ''} onClick={() => setMode('all')}>全部会议</button></div><div className="history-filters"><div className="search-input"><Search size={16} /><input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="搜索会议名称" /></div><select value={status} onChange={(e) => setStatus(e.target.value)}><option value="0">全部状态</option><option value="1">已创建</option><option value="2">进行中</option><option value="3">已结束</option></select></div><div className="meeting-list">{loading ? <div className="empty-state"><RefreshCw size={18} className="spin" />加载记录中…</div> : rows.length === 0 ? <div className="empty-state"><Archive size={20} />还没有符合条件的会议</div> : rows.map((meeting) => <MeetingRow key={meeting.meetingNo} meeting={meeting} onJoin={onJoin} onEnd={endMeeting} notify={notify} />)}</div><div className="list-footer"><span>共 {total} 场会议</span><span>当前身份：{identity}</span></div></section></div></main>
}

function MeetingRow({ meeting, onJoin, onEnd, notify }: { meeting: MeetingInfo; onJoin: (no: string) => Promise<void>; onEnd: (no: string) => Promise<void>; notify: (message: string, tone?: Toast['tone']) => void }) {
  const [expanded, setExpanded] = useState(false)
  const [ticketLoading, setTicketLoading] = useState(false)
  const [ticketInfo, setTicketInfo] = useState<{ ticket: string; joinUrl: string } | null>(null)
  const [showTicketModal, setShowTicketModal] = useState(false)
  const [showDetailModal, setShowDetailModal] = useState(false)
  const [ticketIdentity, setTicketIdentity] = useState('')
  const [ticketName, setTicketName] = useState('')
  const [ticketExpire, setTicketExpire] = useState('3600')
  const [canPublish, setCanPublish] = useState(true)
  const [canSubscribe, setCanSubscribe] = useState(true)
  const [canPublishData, setCanPublishData] = useState(true)
  const [canPublishSources, setCanPublishSources] = useState<string[]>(['camera', 'microphone', 'screen_share'])
  const [detailData, setDetailData] = useState<{ meeting: MeetingInfo; participants: ParticipantInfo[] } | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const meta = statusMeta(Number(meeting.status))
  const duration = formatDuration(meeting.startTime, meeting.status === 3 ? meeting.endTime : undefined)
  const isActive = Number(meeting.status) !== 3
  const openDetail = async () => {
    setShowDetailModal(true)
    setDetailLoading(true)
    try {
      const [meetingRes, participantsRes] = await Promise.all([
        api.getMeeting(meeting.meetingNo),
        api.listParticipants(meeting.meetingNo)
      ])
      setDetailData({ meeting: meetingRes.meeting, participants: participantsRes.participants || [] })
    } catch (e) {
      notify((e as Error).message, 'error')
    } finally {
      setDetailLoading(false)
    }
  }
  const toggleSource = (source: string) => setCanPublishSources((current) => current.includes(source) ? current.filter((s) => s !== source) : [...current, source])
  const generateTicket = async () => {
    const guestIdentity = ticketIdentity.trim() || ('访客_' + Math.random().toString(36).slice(2, 8))
    setTicketLoading(true)
    try {
      const reply = await api.generateTicket(meeting.meetingNo, guestIdentity, ticketName.trim(), Number(ticketExpire) || 3600, canPublish, canSubscribe, canPublishData, canPublishSources)
      const joinUrl = `${location.origin}/guest?ticket=${encodeURIComponent(reply.ticket)}`
      setTicketInfo({ ticket: reply.ticket, joinUrl })
    } catch (e) {
      notify((e as Error).message, 'error')
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
          <span className={`status-pill ${meta.className}`}>{meta.label}</span>
        </div>
        <small>{meeting.meetingNo} · {timeLabel(meeting.createTime)} · 时长 {duration}</small>
      </div>
      <div className="meeting-actions">
        <button className="text-button" onClick={(e) => { e.stopPropagation(); openDetail() }}>详情</button>
        {isActive && <button className="text-button danger" onClick={(e) => { e.stopPropagation(); window.confirm('确定结束该会议？') && onEnd(meeting.meetingNo) }}>结束</button>}
        <button className="button compact secondary" disabled={!isActive} onClick={(e) => { e.stopPropagation(); onJoin(meeting.meetingNo) }}>加入 <ArrowRight size={14} /></button>
        <button className="icon-button small" onClick={(e) => { e.stopPropagation(); setExpanded(!expanded) }} title="更多操作">
          {expanded ? <ChevronLeft size={16} /> : <ChevronRight size={16} />}
        </button>
      </div>
    </div>
    {expanded && (
      <div className="meeting-expanded">
        <div className="meeting-expanded-actions">
          <button className="soft-button" onClick={() => { setShowTicketModal(true); setTicketInfo(null); setTicketIdentity(''); setTicketName(''); setTicketExpire('3600'); setCanPublish(true); setCanSubscribe(true); setCanPublishData(true); setCanPublishSources(['camera', 'microphone', 'screen_share']) }}>
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
            ) : detailData ? (
              <div className="detail-content">
                <div className="detail-section">
                  <h4>基本信息</h4>
                  <div className="detail-grid">
                    <div className="detail-item"><label>会议标题</label><span>{detailData.meeting.title || '未命名会议'}</span></div>
                    <div className="detail-item"><label>会议号</label><span className="mono">{detailData.meeting.meetingNo}</span></div>
                    <div className="detail-item"><label>状态</label><span className={`status-pill ${statusMeta(Number(detailData.meeting.status)).className}`}>{statusMeta(Number(detailData.meeting.status)).label}</span></div>
                    <div className="detail-item"><label>创建人</label><span>{detailData.meeting.createUser || '-'}</span></div>
                    <div className="detail-item"><label>机构</label><span>{detailData.meeting.deptCode || '-'}</span></div>
                    <div className="detail-item"><label>创建时间</label><span>{timeLabel(detailData.meeting.createTime)}</span></div>
                    <div className="detail-item"><label>开始时间</label><span>{timeLabel(detailData.meeting.startTime)}</span></div>
                    <div className="detail-item"><label>结束时间</label><span>{timeLabel(detailData.meeting.endTime) || '进行中'}</span></div>
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
            {isActive && <button className="button primary" onClick={() => { setShowDetailModal(false); onJoin(meeting.meetingNo) }}>加入会议</button>}
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
              <p className="modal-desc">为临时设备（如安全帽）生成邀请票据，无需登录即可加入会议。</p>
              <label className="field-label">参会身份（可选）</label>
              <input className="text-input" value={ticketIdentity} onChange={(e) => setTicketIdentity(e.target.value)} placeholder="留空自动生成，例如：device-001" />
              <label className="field-label" style={{ marginTop: 12 }}>展示名称（可选）</label>
              <input className="text-input" value={ticketName} onChange={(e) => setTicketName(e.target.value)} placeholder="例如：1号安全帽" />
              <label className="field-label" style={{ marginTop: 12 }}>有效期（秒）</label>
              <input className="text-input" type="number" value={ticketExpire} onChange={(e) => setTicketExpire(e.target.value)} placeholder="3600" />
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
                <div className="ticket-field">
                  <label>票据地址</label>
                  <code>{ticketInfo.joinUrl}</code>
                </div>
                <div className="ticket-field" style={{ marginTop: 10 }}>
                  <label>票据号</label>
                  <code>{ticketInfo.ticket}</code>
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

function MeetingRoom({ meeting, perms, name, identity, guest, onLeave, notify }: { meeting: MeetingInfo; perms: JoinPerms; name: string; identity: string; guest: boolean; onLeave: () => void; notify: (message: string, tone?: Toast['tone']) => void }) {
  const [pane, setPane] = useState<'chat' | 'members' | 'manage'>('chat'); const [showPanel, setShowPanel] = useState(true)
  return <main className="room-page"><section className="room-stage"><div className="room-heading"><div><button className="back-button" onClick={onLeave}><ChevronLeft size={16} />{guest ? '离开会议' : '返回大厅'}</button><h1>{meeting.title || 'Live 会议'}</h1><div className="room-id">会议号 <button onClick={() => navigator.clipboard.writeText(meeting.meetingNo).then(() => notify('会议号已复制', 'success'))}><Copy size={13} />{meeting.meetingNo}</button></div></div><div className="room-heading-actions"><span className="live-indicator"><i />已连接</span><button className="icon-button mobile-panel" title="打开侧栏" onClick={() => setShowPanel(!showPanel)} style={guest ? { display: 'none' } : undefined}><PanelRight size={18} /></button></div></div><RoomAudioRenderer /><RoomContent perms={perms} name={name} identity={identity} onLeave={onLeave} notify={notify} /></section>{!guest && <aside className={`room-sidebar ${showPanel ? '' : 'collapsed'}`}><nav className="room-tabs"><button className={pane === 'chat' ? 'active' : ''} onClick={() => setPane('chat')}><MessageSquare size={16} />群聊</button><button className={pane === 'members' ? 'active' : ''} onClick={() => setPane('members')}><Users size={16} />成员</button><button className={pane === 'manage' ? 'active' : ''} onClick={() => setPane('manage')}><Settings2 size={16} />管理</button></nav>{pane === 'chat' && <ChatPane meetingNo={meeting.meetingNo} identity={identity} name={name} guest={guest} perms={perms} notify={notify} />}{pane === 'members' && <MembersPane meetingNo={meeting.meetingNo} notify={notify} />}{pane === 'manage' && <ManagePane meetingNo={meeting.meetingNo} guest={guest} notify={notify} onEnd={onLeave} />}</aside>}</main>
}

function RoomContent({ perms, name, identity, onLeave, notify }: { perms: JoinPerms; name: string; identity: string; onLeave: () => void; notify: (message: string, tone?: Toast['tone']) => void }) { const tracks = useTracks([Track.Source.Camera, Track.Source.ScreenShare], { onlySubscribed: false }); const [selectedTrack, setSelectedTrack] = useState<TrackReferenceOrPlaceholder | null>(null); const [showThumbnails, setShowThumbnails] = useState(true); const [isFullscreen, setIsFullscreen] = useState(false); const [fit, setFit] = useState<'contain' | 'cover' | 'fill'>('cover'); const [zoom, setZoom] = useState(1); const mainVideoRef = useRef<HTMLDivElement>(null); useEffect(() => { if (!selectedTrack && tracks.length > 0) { const screenShare = tracks.find(t => t.source === Track.Source.ScreenShare); setSelectedTrack(screenShare || tracks[0]) } }, [tracks, selectedTrack]); useEffect(() => { if (selectedTrack) { const stillExists = tracks.some(t => t.participant.identity === selectedTrack.participant.identity && t.source === selectedTrack.source); if (!stillExists) { setSelectedTrack(tracks[0] || null) } } }, [tracks, selectedTrack]); const toggleFullscreen = async () => { if (!mainVideoRef.current) return; if (!document.fullscreenElement) { try { await mainVideoRef.current.requestFullscreen(); setIsFullscreen(true) } catch (e) { notify('无法进入全屏模式', 'error') } } else { try { await document.exitFullscreen(); setIsFullscreen(false) } catch (e) { notify('无法退出全屏模式', 'error') } } }; useEffect(() => { const handler = () => setIsFullscreen(!!document.fullscreenElement); document.addEventListener('fullscreenchange', handler); return () => document.removeEventListener('fullscreenchange', handler) }, []); const zoomBy = (delta: number) => setZoom((z) => Math.min(3, Math.max(1, Math.round((z + delta) * 100) / 100))); const cycleFit = () => setFit((f) => f === 'cover' ? 'contain' : f === 'contain' ? 'fill' : 'cover'); const fitLabel = fit === 'cover' ? '铺满' : fit === 'contain' ? '适应' : '拉伸'; return <div className="room-content-layout"><div className={`thumbnail-strip ${showThumbnails ? '' : 'hidden'}`}><div className="thumbnail-header"><span className="thumbnail-title">参会人</span><button className="icon-button small" onClick={() => setShowThumbnails(!showThumbnails)} title={showThumbnails ? '隐藏参会人' : '显示参会人'}>{showThumbnails ? <ChevronUp size={14} /> : <ChevronDown size={14} />}</button></div><div className="thumbnail-row">{tracks.map((track) => <div key={`${track.participant.identity}-${track.source}`} className={`thumbnail-item ${selectedTrack?.participant.identity === track.participant.identity && selectedTrack?.source === track.source ? 'active' : ''}`} onClick={() => { setSelectedTrack(track); setZoom(1); setFit('cover') }}><TrackTile track={track} /><div className="thumbnail-name">{track.participant.name || track.participant.identity}{track.participant.isLocal ? ' · 我' : ''}</div></div>)}</div></div><div className="main-video-area" ref={mainVideoRef}>{selectedTrack ? <TrackTile key={`${selectedTrack.participant.identity}-${selectedTrack.source}`} track={selectedTrack} fit={fit} zoom={zoom} /> : <div className="room-empty"><span className="empty-camera"><Video size={27} /></span><b>等待成员加入</b><small>开启摄像头后，你会出现在这里</small></div>}<div className="video-toolbar"><button className="toolbar-button" onClick={() => zoomBy(-0.25)} title="缩小" disabled={zoom <= 1}><ZoomOut size={15} /></button><span className="toolbar-zoom">{Math.round(zoom * 100)}%</span><button className="toolbar-button" onClick={() => zoomBy(0.25)} title="放大" disabled={zoom >= 3}><ZoomIn size={15} /></button><button className="toolbar-button text" onClick={() => setZoom(1)} title="重置缩放">重置</button><button className="toolbar-button text" onClick={cycleFit} title="切换显示模式">{fitLabel}</button><button className="toolbar-button" onClick={toggleFullscreen} title={isFullscreen ? '退出全屏' : '全屏'}>{isFullscreen ? <Minimize2 size={15} /> : <Maximize2 size={15} />}</button></div></div><RoomControls perms={perms} onLeave={onLeave} notify={notify} /></div> }
function TrackTile({ track, fit = 'cover', zoom = 1 }: { track: TrackReferenceOrPlaceholder; fit?: 'contain' | 'cover' | 'fill'; zoom?: number }) { const title = track.participant.name || track.participant.identity; const mediaStream = (track.publication?.track as { mediaStream?: MediaStream } | undefined)?.mediaStream ?? null; const videoRef = useRef<HTMLVideoElement>(null); useEffect(() => { const video = videoRef.current; if (!video || !mediaStream) return; video.srcObject = mediaStream; video.play().catch(() => undefined); return () => { if (video.srcObject === mediaStream) video.srcObject = null } }, [mediaStream]); return <div className="track-tile" data-source={track.source === Track.Source.ScreenShare ? 'screenShare' : 'camera'}>{mediaStream ? <video ref={videoRef} className="track-video" autoPlay muted playsInline style={{ objectFit: fit, transform: zoom !== 1 ? `scale(${zoom})` : undefined }} /> : <div className="track-offline"><span className="avatar tiny">{initials(title)}</span><span>{title}</span></div>}<div className="track-label"><span className="avatar tiny">{initials(title)}</span><span>{title}{track.participant.isLocal ? ' · 我' : ''}</span></div></div> }
function RoomControls({ perms, onLeave, notify }: { perms: JoinPerms; onLeave: () => void; notify: (message: string, tone?: Toast['tone']) => void }) { const { localParticipant } = useLocalParticipant(); const camera = localParticipant?.isCameraEnabled ?? false; const mic = localParticipant?.isMicrophoneEnabled ?? false; const screen = localParticipant?.isScreenShareEnabled ?? false; const canPublish = perms.canPublish; const sources = perms.canPublishSources; const allowSource = (source: string) => !sources || sources.includes(source); const allowCamera = canPublish && allowSource('camera'); const allowMic = canPublish && allowSource('microphone'); const allowScreen = canPublish && allowSource('screen_share'); const toggle = async (kind: 'camera' | 'mic' | 'screen') => { try { if (kind === 'camera' && !allowCamera) return notify('当前身份无摄像头发布权限', 'warning'); if (kind === 'mic' && !allowMic) return notify('当前身份无麦克风发布权限', 'warning'); if (kind === 'screen' && !allowScreen) return notify('当前身份无屏幕共享权限', 'warning'); if (kind === 'camera') await localParticipant.setCameraEnabled(!camera); if (kind === 'mic') await localParticipant.setMicrophoneEnabled(!mic); if (kind === 'screen') await localParticipant.setScreenShareEnabled(!screen); } catch (e) { notify(`媒体设备不可用：${(e as Error).message}`, 'error') } }; return <div className="room-controls"><button className={`control ${camera ? 'on' : ''}`} disabled={!allowCamera} onClick={() => toggle('camera')} title="摄像头"><Camera size={19} /><span>{camera ? '关闭视频' : '打开视频'}</span></button><button className={`control ${mic ? 'on' : ''}`} disabled={!allowMic} onClick={() => toggle('mic')} title="麦克风"><Mic size={19} /><span>{mic ? '关闭语音' : '打开语音'}</span></button><button className={`control ${screen ? 'on' : ''}`} disabled={!allowScreen} onClick={() => toggle('screen')} title="共享屏幕"><MonitorUp size={19} /><span>{screen ? '停止共享' : '共享屏幕'}</span></button><button className="control leave" onClick={() => window.confirm('确定离开会议？') && onLeave()} title="离开会议"><LogOut size={19} /><span>离开会议</span></button></div> }

function ChatPane({ meetingNo, identity, name, guest, perms, notify }: { meetingNo: string; identity: string; name: string; guest: boolean; perms: JoinPerms; notify: (message: string, tone?: Toast['tone']) => void }) { const room = useRoomContext(); const [messages, setMessages] = useState<MeetingMessage[]>([]); const [text, setText] = useState(''); const [loading, setLoading] = useState(true); useEffect(() => { if (!guest) api.listMessages(meetingNo).then((data) => setMessages(data.messages || [])).catch(() => undefined).finally(() => setLoading(false)); else setLoading(false) }, [guest, meetingNo]); useEffect(() => { const receive = (payload: Uint8Array, participant?: { identity?: string; name?: string }, _kind?: unknown, topic?: string) => { if (topic !== 'lk.chat' && topic !== 'chat') return; try { const message = JSON.parse(new TextDecoder().decode(payload)); setMessages((current) => current.some((item) => item.messageId === message.messageId) ? current : [...current, { ...message, senderId: participant?.identity || '', senderName: participant?.name || participant?.identity || '系统', createTime: '刚刚' }]) } catch { /* ignore malformed room data */ } }; room.on(RoomEvent.DataReceived, receive); return () => { room.off(RoomEvent.DataReceived, receive) } }, [room]); 	const send = async () => { const content = text.trim(); if (!content) return; if (!perms.canPublishData) return notify('当前身份无消息发送权限', 'warning'); const localId = crypto.randomUUID?.() || `msg-${Date.now()}`; const message = { messageId: localId, content, messageType: 'text' }; try { await room.localParticipant.publishData(new TextEncoder().encode(JSON.stringify(message)), { reliable: true, topic: 'lk.chat' }); if (!guest) await api.reportMessage({ meetingNo, content, messageType: 'text' }); setMessages((current) => [...current, { ...message, senderId: identity, senderName: name || identity, createTime: '刚刚' }]); setText('') } catch (e) { notify(`消息发送失败：${(e as Error).message}`, 'error') } }; return <div className="side-content chat-content"><div className="side-intro"><div><span className="eyebrow">实时频道</span><h3>群聊</h3></div><span className="live-badge"><i />Live</span></div><div className="chat-list">{loading ? <div className="side-empty">加载历史消息…</div> : messages.length ? messages.map((message) => <div key={message.messageId} className={`chat-item ${message.senderId === identity ? 'mine' : ''}`}><span className="avatar tiny">{initials(message.senderName || message.senderId)}</span><div><div className="chat-meta"><b>{message.senderId === identity ? '我' : message.senderName || message.senderId}</b><time>{message.createTime}</time></div><p>{message.content}</p></div></div>) : <div className="side-empty"><MessageSquare size={20} />还没有消息，打个招呼吧</div>}</div><div className="chat-composer"><input value={text} onChange={(e) => setText(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && send()} placeholder="输入消息，按 Enter 发送" /><button onClick={send} title="发送"><Send size={17} /></button></div></div> }

function MembersPane({ meetingNo, notify }: { meetingNo: string; notify: (message: string, tone?: Toast['tone']) => void }) { const participants = useParticipants(); const [history, setHistory] = useState<ParticipantInfo[]>([]); const load = () => api.listParticipants(meetingNo).then((data) => setHistory(data.participants || [])).catch((e) => notify((e as Error).message, 'error')); useEffect(() => { load() }, [meetingNo]); const onlineIds = new Set(participants.map(p => p.identity)); const list = history.map(m => ({ ...m, online: onlineIds.has(m.identity) })); const onlineCount = list.filter(m => m.online).length; return <div className="side-content"><div className="side-intro"><div><span className="eyebrow">实时状态</span><h3>成员 <em>{onlineCount}/{list.length}</em></h3></div><button className="icon-button" title="刷新成员" onClick={load}><RefreshCw size={16} /></button></div><div className="member-list">{list.length ? list.map((member) => <div className={`member-item ${member.online ? '' : 'offline'}`} key={member.identity}><span className="avatar tiny">{initials(member.name)}</span><div className="member-copy"><b>{member.name}</b><small>{member.identity}</small></div><span className={`member-status ${member.online ? 'online' : 'offline'}`}>{member.online ? '在线' : '离线'}</span></div>) : <div className="side-empty"><Users size={20} />暂时没有成员记录</div>}</div></div> }

function ManagePane({ meetingNo, guest, notify, onEnd }: { meetingNo: string; guest: boolean; notify: (message: string, tone?: Toast['tone']) => void; onEnd: () => void }) { const participants = useParticipants(); const [target, setTarget] = useState(''); const [identity, setIdentity] = useState(''); const [inviteName, setInviteName] = useState(''); const [expire, setExpire] = useState('3600'); const [ticket, setTicket] = useState<TicketReply | null>(null); const [members, setMembers] = useState<ParticipantInfo[]>([]); const [canPublish, setCanPublish] = useState(true); const [canSubscribe, setCanSubscribe] = useState(true); const [canPublishData, setCanPublishData] = useState(true); const [canPublishSources, setCanPublishSources] = useState<string[]>(['camera', 'microphone', 'screen_share']); const load = () => api.listParticipants(meetingNo).then((data) => setMembers(data.participants || [])).catch(() => undefined); useEffect(() => { if (!guest) load() }, [guest, meetingNo]); const onlineIds = new Set(participants.map(p => p.identity)); const onlineMembers = members.filter(m => onlineIds.has(m.identity)); const mute = async (kind: 'audio' | 'video') => { if (!target) return notify('请先选择成员', 'warning'); try { await api.muteParticipant(meetingNo, target, true, kind); notify('管理指令已发送', 'success') } catch (e) { notify((e as Error).message, 'error') } }; const kick = async () => { if (!target) return notify('请先选择成员', 'warning'); if (!window.confirm(`确定将 ${target} 移出会议？`)) return; try { await api.kickParticipant(meetingNo, target); notify('成员已移出会议', 'success'); load() } catch (e) { notify((e as Error).message, 'error') } }; const toggleSource = (source: string) => { setCanPublishSources(current => current.includes(source) ? current.filter(s => s !== source) : [...current, source]) }; const generate = async () => { const guestIdentity = identity.trim() || ('访客_' + Math.random().toString(36).slice(2, 8)); try { const reply = await api.generateTicket(meetingNo, guestIdentity, inviteName.trim(), Number(expire) || 3600, canPublish, canSubscribe, canPublishData, canPublishSources); setTicket(reply) } catch (e) { notify((e as Error).message, 'error') } }; const share = () => { if (!ticket) return; const url = `${location.origin}/guest?ticket=${encodeURIComponent(ticket.ticket)}`; navigator.clipboard.writeText(url).then(() => notify('邀请链接已复制', 'success')) }; if (guest) return <div className="side-content"><div className="guest-lock"><Shield size={24} /><h3>访客模式</h3><p>临时邀请可以加入会议并使用媒体和群聊，但不能管理成员或生成新的邀请。</p></div></div>; return <div className="side-content manage-content"><div className="side-intro"><div><span className="eyebrow">主持人工具</span><h3>会议管理</h3></div><span className="admin-badge"><ShieldCheck size={14} />已授权</span></div><section className="manage-section"><h4>成员控制</h4><select className="text-input" value={target} onChange={(e) => setTarget(e.target.value)}><option value="">选择在线成员</option>{onlineMembers.map((member) => <option key={member.identity} value={member.identity}>{member.name || member.identity} ({member.identity})</option>)}</select><div className="manage-actions"><button className="soft-button" onClick={() => mute('audio')}><Mic size={15} />静音语音</button><button className="soft-button" onClick={() => mute('video')}><Video size={15} />关闭视频</button><button className="soft-button danger" onClick={kick}><UserRound size={15} />移出会议</button></div></section><section className="manage-section"><h4>临时会议邀请</h4><input className="text-input" value={identity} onChange={(e) => setIdentity(e.target.value)} placeholder="参会身份（可选，留空自动生成）" /><input className="text-input" value={inviteName} onChange={(e) => setInviteName(e.target.value)} placeholder="显示名称（可选）" /><div className="invite-inline"><input className="text-input" type="number" min="60" max="86400" value={expire} onChange={(e) => setExpire(e.target.value)} /><button className="button secondary" onClick={generate}>生成邀请</button></div><div className="permission-section"><h5>访客权限</h5><label className="checkbox-label"><input type="checkbox" checked={canPublish} onChange={(e) => setCanPublish(e.target.checked)} />可发布音视频</label><label className="checkbox-label"><input type="checkbox" checked={canSubscribe} onChange={(e) => setCanSubscribe(e.target.checked)} />可订阅音视频</label><label className="checkbox-label"><input type="checkbox" checked={canPublishData} onChange={(e) => setCanPublishData(e.target.checked)} />可发送消息</label><div className="source-permissions"><span className="source-label">可发布轨道：</span><label className="checkbox-label small"><input type="checkbox" checked={canPublishSources.includes('camera')} onChange={() => toggleSource('camera')} />摄像头</label><label className="checkbox-label small"><input type="checkbox" checked={canPublishSources.includes('microphone')} onChange={() => toggleSource('microphone')} />麦克风</label><label className="checkbox-label small"><input type="checkbox" checked={canPublishSources.includes('screen_share')} onChange={() => toggleSource('screen_share')} />屏幕共享</label></div></div>{ticket && <div className="ticket-result"><span>有效期至 {ticket.expireTime}</span><code>{`${location.origin}/guest?ticket=${encodeURIComponent(ticket.ticket)}`}</code><button onClick={share}><Copy size={14} />复制邀请链接</button></div>}</section><RealtimeTools meetingNo={meetingNo} target={target} guest={guest} notify={notify} /><section className="manage-section"><h4>会议状态</h4><button className="end-meeting" onClick={() => window.confirm('结束后所有成员都会离开会议，是否继续？') && api.endMeeting(meetingNo).then(() => { notify('会议已结束', 'success'); onEnd() }).catch((e) => notify((e as Error).message, 'error'))}><X size={15} />结束整个会议</button></section></div> }

function RealtimeTools({ meetingNo, target, guest, notify }: { meetingNo: string; target: string; guest: boolean; notify: (message: string, tone?: Toast['tone']) => void }) { const room = useRoomContext(); const [payload, setPayload] = useState('hello data'); const send = async () => { if (!payload.trim()) return; try { const bytes = new TextEncoder().encode(payload); await room.localParticipant.publishData(bytes, { reliable: true, topic: 'data' }); if (!guest) await api.sendMeetingData(meetingNo, 'data', bytesToBase64(bytes)); notify('Data 已发送', 'success') } catch (e) { notify(`Data 发送失败：${(e as Error).message}`, 'error') } }; const rpc = async () => { if (!target) return notify('请先选择成员', 'warning'); try { const reply = await api.performRpc(meetingNo, target, payload); notify(`RPC 返回：${reply.response || '无内容'}`, 'success') } catch (e) { notify(`RPC 失败：${(e as Error).message}`, 'error') } }; return <section className="manage-section"><h4>Data / RPC</h4><input className="text-input" value={payload} onChange={(e) => setPayload(e.target.value)} placeholder="消息内容" /><div className="manage-actions"><button className="soft-button" onClick={send}><Database size={15} />发送 Data</button><button className="soft-button" onClick={rpc}><MoreHorizontal size={15} />服务端 RPC</button></div></section> }

const consumedTickets = new Set<string>()

function GuestView({ ticket, join, name, identity, onJoin, onLeave, notify }: { ticket: string | null; join: { token: string; wsUrl: string; meeting: MeetingInfo; perms: JoinPerms } | null; name: string; identity: string; onJoin: (ticket: string) => Promise<boolean>; onLeave: () => void; notify: (message: string, tone?: Toast['tone']) => void }) {
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
    return <><Topbar name={name} identity={identity} guest onLogout={onLeave} /><LiveKitRoom serverUrl={join.wsUrl} token={join.token} connect audio={true} video={true} onDisconnected={onLeave}><MeetingRoom meeting={join.meeting} perms={join.perms} name={name} identity={identity} guest onLeave={onLeave} notify={notify} /></LiveKitRoom></>
  }
  return <main className="guest-gate"><div className="surface guest-gate-card"><div className="brand-lockup"><span className="brand-mark">L</span><span>Live 会议中心</span></div><div className="guest-gate-copy"><span className="eyebrow">邀请访客</span><h2>加入会议</h2><p>{loading ? '正在验证票据，请稍候…' : failed ? '票据无效、已使用或已过期，请联系会议主持人重新生成。' : '正在准备会议空间…'}</p></div>{loading ? <div className="guest-gate-loading"><RefreshCw size={18} className="spin" /><span>正在连接会议</span></div> : failed ? <button className="button primary wide" onClick={() => { consumedTickets.delete(ticket || ''); setRetryCount((n) => n + 1) }}>重新尝试 <ArrowRight size={15} /></button> : null}</div></main>
}

function bytesToBase64(bytes: Uint8Array) { let binary = ''; for (let index = 0; index < bytes.length; index += 0x8000) binary += String.fromCharCode(...bytes.subarray(index, index + 0x8000)); return btoa(binary) }
