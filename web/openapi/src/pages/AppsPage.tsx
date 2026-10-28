import {
  AppWindow,
  ChevronLeft,
  ChevronRight,
  Eye,
  EyeOff,
  KeyRound,
  LoaderCircle,
  LockKeyhole,
  Plus,
  RefreshCw,
  RotateCcw,
  Search,
  ShieldCheck,
  Power,
} from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { adminRequest, buildUrl } from '../lib/http'
import { formatDateTime, shortId } from '../lib/format'
import type {
  AdminResult,
  AppListData,
  AppRecord,
  CreateAppData,
  ResetSecretData,
  RevealSignKeyData,
  Settings,
  ToastKind,
} from '../types'
import ConfirmDialog from '../components/ConfirmDialog'
import CopyButton from '../components/CopyButton'
import SecretReveal from '../components/SecretReveal'

interface AppsPageProps {
  settings: Settings
  onChange: (patch: Partial<Settings>) => void
  notify: (kind: ToastKind, text: string) => void
}

interface ConfirmState {
  type: 'disable' | 'enable' | 'reset' | 'reveal'
  app: AppRecord
}

interface RevealState {
  title: string
  appId: string
  secret: string
  signKey?: string
  hint?: string
}

export default function AppsPage({ settings, onChange, notify }: AppsPageProps) {
  const [showToken, setShowToken] = useState(false)
  const [apps, setApps] = useState<AppRecord[]>([])
  const [total, setTotal] = useState(0)
  const [pageNum, setPageNum] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [name, setName] = useState('')
  const [remark, setRemark] = useState('')
  const [signRequired, setSignRequired] = useState(false)
  const [creating, setCreating] = useState(false)
  const [reveal, setReveal] = useState<RevealState | null>(null)
  const [confirm, setConfirm] = useState<ConfirmState | null>(null)
  const [confirmLoading, setConfirmLoading] = useState(false)
  const initialized = useRef(false)
  const revealRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (reveal) revealRef.current?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }, [reveal])

  const loadApps = useCallback(async (nextPage = pageNum, nextSize = pageSize) => {
    if (!settings.adminToken.trim()) {
      setApps([])
      setTotal(0)
      setError('')
      return
    }
    setLoading(true)
    setError('')
    const result: AdminResult<AppListData> = await adminRequest<AppListData>({
      baseUrl: settings.baseUrl,
      token: settings.adminToken,
      path: '/open-admin/v1/app/list',
      query: { keyword, status: status || undefined, pageNum: nextPage, pageSize: nextSize },
    })
    setLoading(false)
    if (!result.ok || !result.data) {
      setError(adminFailure(result, settings.adminToken))
      return
    }
    setApps(result.data.items ?? [])
    setTotal(result.data.total ?? 0)
    setPageNum(result.data.pageNum || nextPage)
  }, [keyword, pageNum, pageSize, settings.adminToken, settings.baseUrl, status])

  useEffect(() => {
    if (initialized.current) return
    initialized.current = true
    if (settings.adminToken.trim()) void loadApps(1)
  }, [loadApps, settings.adminToken])

  const handleCreate = async () => {
    if (!name.trim()) {
      notify('error', '请填写应用名称')
      return
    }
    setCreating(true)
    const result = await adminRequest<CreateAppData>({
      baseUrl: settings.baseUrl,
      token: settings.adminToken,
      path: '/open-admin/v1/app/create',
      method: 'POST',
      body: { name: name.trim(), remark: remark.trim(), signRequired },
    })
    setCreating(false)
    if (!result.ok || !result.data) {
      notify('error', adminFailure(result, settings.adminToken))
      return
    }
    setName('')
    setRemark('')
    setSignRequired(false)
    setReveal({
      title: '应用创建成功',
      appId: result.data.appId,
      secret: result.data.secret,
      signKey: result.data.signKey,
      hint: result.data.signRequired ? 'secret 与 signKey 仅返回一次，请立即保存。' : 'secret 仅返回一次，请立即保存。',
    })
    notify('success', '应用已创建')
    void loadApps(1)
  }

  const executeConfirm = async () => {
    if (!confirm) return
    setConfirmLoading(true)
    const { type, app } = confirm
    let result: AdminResult<unknown>
    if (type === 'reset') {
      result = await adminRequest<ResetSecretData>({
        baseUrl: settings.baseUrl,
        token: settings.adminToken,
        path: '/open-admin/v1/app/secret/reset',
        method: 'POST',
        body: { appId: app.appId },
      })
    } else if (type === 'reveal') {
      result = await adminRequest<RevealSignKeyData>({
        baseUrl: settings.baseUrl,
        token: settings.adminToken,
        path: '/open-admin/v1/app/sign-key/reveal',
        method: 'POST',
        body: { appId: app.appId },
      })
    } else {
      result = await adminRequest<unknown>({
        baseUrl: settings.baseUrl,
        token: settings.adminToken,
        path: '/open-admin/v1/app/disable',
        method: 'POST',
        body: { appId: app.appId, status: type === 'disable' ? 2 : 1 },
      })
    }
    setConfirmLoading(false)
    if (!result.ok) {
      notify('error', adminFailure(result, settings.adminToken))
      return
    }

    if (type === 'reset' && result.data) {
      const data = result.data as ResetSecretData
      setReveal({
        title: '密钥已重置',
        appId: data.appId,
        secret: data.secret,
        signKey: data.signKey,
        hint: `旧密钥将在 ${formatDateTime(data.oldSecretExpireAt)} 失效，期间新旧密钥同时可用。`,
      })
    } else if (type === 'reveal' && result.data) {
      const data = result.data as RevealSignKeyData
      setReveal({
        title: '当前签名密钥',
        appId: data.appId,
        secret: '',
        signKey: data.signKey,
        hint: `查看时间 ${formatDateTime(data.revealedAt)}；该操作已记录审计日志，请勿转发到公开渠道。`,
      })
    }
    notify('success', type === 'reset' ? '密钥已重置' : type === 'reveal' ? '已获取签名密钥' : type === 'disable' ? '应用已停用' : '应用已启用')
    setConfirm(null)
    void loadApps()
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const canQuery = settings.adminToken.trim().length > 0

  return (
    <div className="page-stack">
      <section className="page-intro">
        <div>
          <span className="eyebrow">应用</span>
          <h1>应用管理</h1>
          <p>创建开放应用、启停状态与重置密钥。管理面使用 JWT 鉴权，与 Basic 凭据相互独立。</p>
        </div>
        <div className="intro-stats">
          <span><AppWindow size={15} /> {total} 个应用</span>
          <span><ShieldCheck size={15} /> JwtAuth</span>
        </div>
      </section>

      <div className="apps-top-grid">
        <section className="card admin-connection-card">
          <div className="card-heading compact">
            <div>
              <span className="eyebrow">管理鉴权</span>
              <h2>管理面连接</h2>
            </div>
            <span className={`health-pill ${canQuery ? 'ok' : ''}`}>
              <LockKeyhole size={13} /> {canQuery ? 'JWT 已填写' : '未配置 JWT'}
            </span>
          </div>
          <div className="field">
            <span className="field-label">管理面地址</span>
            <div className="static-field mono">{buildUrl(settings.baseUrl, '/open-admin/v1')}</div>
            <p className="field-hint">与开放面同端口，由 nginx 默认不对外转发。</p>
          </div>
          <div className="field">
            <label className="field-label" htmlFor="adminToken">JWT Token</label>
            <div className="secret-input-group">
              <input
                id="adminToken"
                className="text-input mono"
                type={showToken ? 'text' : 'password'}
                value={settings.adminToken}
                onChange={(event) => onChange({ adminToken: event.target.value.trim() })}
                placeholder="Bearer token（不要带 Bearer 前缀）"
                spellCheck={false}
                autoComplete="off"
              />
              <button type="button" className="input-action" onClick={() => setShowToken((value) => !value)} aria-label={showToken ? '隐藏 token' : '显示 token'}>
                {showToken ? <EyeOff size={15} /> : <Eye size={15} />}
              </button>
            </div>
            <p className="field-hint">JWT 只保存在当前浏览器，仅限测试环境。401 时请重新获取。</p>
          </div>
          <div className="connection-actions">
            <button type="button" className="button secondary" onClick={() => void loadApps(1)} disabled={!canQuery || loading}>
              {loading ? <LoaderCircle size={14} className="spin" /> : <RefreshCw size={14} />}
              验证并加载
            </button>
          </div>
          <div className="connection-note">
            <LockKeyhole size={15} />
            <span>管理面变更会立即刷新开放面凭据缓存；停用后请求立刻返回 403。</span>
          </div>
        </section>

        <section className="card create-app-card">
          <div className="card-heading compact">
            <div>
              <span className="eyebrow">新建</span>
              <h2>创建应用</h2>
            </div>
            <span className="method-icon"><Plus size={17} /></span>
          </div>
          <div className="field">
            <label className="field-label" htmlFor="appName">应用名称</label>
            <input
              id="appName"
              className="text-input"
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="例如：podman 试点 / Java 巡检服务"
              maxLength={64}
              autoComplete="off"
              spellCheck={false}
            />
          </div>
          <div className="field">
            <label className="field-label" htmlFor="appRemark">备注</label>
            <input
              id="appRemark"
              className="text-input"
              value={remark}
              onChange={(event) => setRemark(event.target.value)}
              placeholder="可选，说明调用方与用途"
              maxLength={255}
              autoComplete="off"
              spellCheck={false}
            />
          </div>
          <label className="sign-option">
            <input
              type="checkbox"
              checked={signRequired}
              onChange={(event) => setSignRequired(event.target.checked)}
              aria-label="启用请求签名"
            />
            <div>
              <b>启用请求签名</b>
              <small>HMAC-SHA256 · 创建后 signKey 仅展示一次，需要 Redis 防重放</small>
            </div>
            <ShieldCheck size={16} />
          </label>
          <button type="button" className="button primary wide" onClick={() => void handleCreate()} disabled={!canQuery || creating}>
            {creating ? <LoaderCircle size={15} className="spin" /> : <Plus size={15} />}
            {creating ? '创建中…' : '创建应用'}
          </button>
          <p className="field-hint center">appId、secret 由服务端生成；启用签名时 signKey 也仅返回一次。</p>
        </section>
      </div>

      {reveal && (
        <div ref={revealRef}>
          <SecretReveal
            title={reveal.title}
            appId={reveal.appId}
            secret={reveal.secret}
            signKey={reveal.signKey}
            hint={reveal.hint}
            onDismiss={() => setReveal(null)}
            notify={notify}
          />
        </div>
      )}

      {error && <div className="alert error"><LockKeyhole size={16} /><span>{error}</span></div>}

      <section className="card apps-list-card">
        <div className="card-heading">
          <div>
            <span className="eyebrow">应用</span>
            <h2>应用列表</h2>
          </div>
          <span className="count-badge">{total}</span>
        </div>

        <div className="apps-filters">
          <div className="search-box">
            <Search size={14} />
            <input
              value={keyword}
              onChange={(event) => setKeyword(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') void loadApps(1)
              }}
              placeholder="搜索 appId 或名称"
              aria-label="搜索应用"
            />
          </div>
          <select value={status} onChange={(event) => setStatus(Number(event.target.value))} aria-label="按状态过滤">
            <option value={0}>全部状态</option>
            <option value={1}>启用</option>
            <option value={2}>停用</option>
          </select>
          <button type="button" className="button secondary compact" onClick={() => void loadApps(1)} disabled={!canQuery || loading}>
            {loading ? <LoaderCircle size={14} className="spin" /> : <Search size={14} />}
            查询
          </button>
        </div>

        {!apps.length && !loading && (
          <div className="history-empty">
            <AppWindow size={22} />
            <span>{canQuery ? '暂无应用，先在上方创建一个。' : '填写 JWT 后点击「验证并加载」。'}</span>
          </div>
        )}

        {apps.length > 0 && (
          <div className="table-scroll">
            <table className="apps-table">
              <thead>
                <tr>
                  <th>应用</th>
                  <th>状态</th>
                  <th>最近调用</th>
                  <th>创建时间</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {apps.map((app) => (
                  <tr key={app.appId}>
                    <td>
                      <div className="app-cell">
                        <span className="app-avatar"><KeyRound size={15} /></span>
                        <div>
                          <b>{app.name}</b>
                          <span className="app-id-line">
                            <code title={app.appId}>{shortId(app.appId, 20)}</code>
                            <CopyButton text={app.appId} label="" className="compact-copy" />
                          </span>
                          {app.remark && <small>{app.remark}</small>}
                        </div>
                      </div>
                    </td>
                    <td>
                      <span className={`app-status ${app.status === 1 ? 'ok' : 'off'}`}>
                        {app.status === 1 ? '启用' : '停用'}
                      </span>
                      {app.signRequired && <span className="app-status sign">已签名</span>}
                    </td>
                    <td className="muted nowrap">{formatDateTime(app.lastUsedAt)}</td>
                    <td className="muted nowrap">{formatDateTime(app.createdAt)}</td>
                    <td>
                      <div className="row-actions">
                        <button
                          type="button"
                          className="text-button"
                          onClick={() => setConfirm({ type: app.status === 1 ? 'disable' : 'enable', app })}
                        >
                          <Power size={12} /> {app.status === 1 ? '停用' : '启用'}
                        </button>
                        <button type="button" className="text-button" onClick={() => setConfirm({ type: 'reset', app })}>
                          <RotateCcw size={12} /> 重置密钥
                        </button>
                        {app.signRequired && (
                          <button type="button" className="text-button" onClick={() => setConfirm({ type: 'reveal', app })}>
                            <KeyRound size={12} /> 查看 signKey
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <div className="apps-footer">
          <span className="muted">第 {pageNum} / {totalPages} 页 · 共 {total} 条</span>
          <div className="pagination">
            <select value={pageSize} onChange={(event) => { const size = Number(event.target.value); setPageSize(size); void loadApps(1, size) }} aria-label="每页数量">
              <option value={10}>10 条/页</option>
              <option value={20}>20 条/页</option>
              <option value={50}>50 条/页</option>
            </select>
            <button type="button" className="icon-page" disabled={pageNum <= 1 || loading} onClick={() => void loadApps(pageNum - 1)} aria-label="上一页">
              <ChevronLeft size={15} />
            </button>
            <button type="button" className="icon-page" disabled={pageNum >= totalPages || loading} onClick={() => void loadApps(pageNum + 1)} aria-label="下一页">
              <ChevronRight size={15} />
            </button>
          </div>
        </div>
      </section>

      <ConfirmDialog
        open={confirm !== null}
        title={confirm ? confirmTitle(confirm.type) : ''}
        message={confirm ? confirmMessage(confirm) : ''}
        confirmLabel={confirm ? confirmLabel(confirm.type) : '确认'}
        danger={confirm?.type === 'disable' || confirm?.type === 'reset'}
        loading={confirmLoading}
        onCancel={() => setConfirm(null)}
        onConfirm={() => void executeConfirm()}
      />
    </div>
  )
}

function adminFailure(result: AdminResult<unknown>, token: string): string {
  // 管理面信封恒为 HTTP 200，JWT 失效只能靠 code 识别，不能静默当成普通业务错误。
  if (result.code === 100401) {
    if (!token.trim()) return '请先填写管理面 JWT token'
    return '管理面 JWT 已失效，请重新获取并填写（当前输入已保留）'
  }
  const base = result.msg || `管理面请求失败（HTTP ${result.httpStatus}）`
  return result.requestId ? `${base} · rid ${shortId(result.requestId, 12)}` : base
}

function confirmTitle(type: ConfirmState['type']): string {
  if (type === 'reset') return '重置应用密钥？'
  if (type === 'reveal') return '查看签名密钥？'
  return type === 'disable' ? '停用应用？' : '启用应用？'
}

function confirmMessage(state: ConfirmState): string {
  if (state.type === 'reset') {
    return `将为「${state.app.name}」生成新密钥。旧密钥保留 24 小时后失效，期间新旧密钥都可用。`
  }
  if (state.type === 'reveal') {
    return `将查看「${state.app.name}」当前生效的 signKey。该操作会记录审计日志，请确认当前环境允许查看敏感凭据。`
  }
  if (state.type === 'disable') {
    return `停用「${state.app.name}」后，所有使用该应用凭据的请求会立即返回 403 / 130101。`
  }
  return `启用「${state.app.name}」后，现有有效密钥会立即恢复鉴权。`
}

function confirmLabel(type: ConfirmState['type']): string {
  if (type === 'reset') return '生成新密钥'
  if (type === 'reveal') return '确认查看'
  return type === 'disable' ? '确认停用' : '确认启用'
}
