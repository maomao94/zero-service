import { AppWindow, BookOpen, Braces, FlaskConical, Info, ShieldAlert, TerminalSquare } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import ConnectionPanel from './components/ConnectionPanel'
import HistoryPanel from './components/HistoryPanel'
import MethodPicker from './components/MethodPicker'
import RequestPanel from './components/RequestPanel'
import ResponsePanel, { type RequestPreview } from './components/ResponsePanel'
import Toast from './components/Toast'
import { checkHealth, callGateway, type HealthResult } from './lib/http'
import { generateTid, parseJsonObject, shortId } from './lib/format'
import { DEFAULT_METHOD, METHODS, findMethod } from './lib/methods'
import {
  clearStoredCredentials,
  loadHistory,
  loadSettings,
  saveHistory,
  saveSettings,
} from './lib/storage'
import type {
  GatewayCallResult,
  HistoryEntry,
  MethodDefinition,
  Settings,
  TabKey,
  ToastKind,
  ToastMessage,
} from './types'
import AppsPage from './pages/AppsPage'
import ErrorsPage from './pages/ErrorsPage'

const TABS: Array<{ key: TabKey; label: string; icon: typeof TerminalSquare }> = [
  { key: 'debug', label: 'API 调试', icon: TerminalSquare },
  { key: 'apps', label: '应用管理', icon: AppWindow },
  { key: 'errors', label: '错误码速查', icon: BookOpen },
]

const initialSettings = loadSettings()
const initialMethod = findMethod(initialSettings.lastMethod)
const MODBUS_METHOD_COUNT = METHODS.filter((method) => method.category.startsWith('modbus')).length

export default function App() {
  const [tab, setTab] = useState<TabKey>('debug')
  const [settings, setSettings] = useState<Settings>(initialSettings)
  const [history, setHistory] = useState<HistoryEntry[]>(() => loadHistory())
  const [toast, setToast] = useState<ToastMessage | null>(null)
  const [methodKey, setMethodKey] = useState(initialMethod.key)
  const [drafts, setDrafts] = useState<Record<string, string>>(() =>
    Object.fromEntries(METHODS.map((method) => [method.key, method.example])),
  )
  const [tid, setTid] = useState(() => generateTid())
  const [sending, setSending] = useState(false)
  const [result, setResult] = useState<GatewayCallResult | null>(null)
  const [lastRequest, setLastRequest] = useState<RequestPreview | null>(null)
  const [health, setHealth] = useState<HealthResult | null>(null)
  const [checking, setChecking] = useState(false)

  const toastTimer = useRef<number | undefined>(undefined)

  const method = findMethod(methodKey)
  const dataText = drafts[method.key] ?? method.example

  useEffect(() => {
    saveSettings(settings)
  }, [settings])

  useEffect(() => {
    saveHistory(history)
  }, [history])

  const notify = useCallback((kind: ToastKind, text: string) => {
    if (toastTimer.current) window.clearTimeout(toastTimer.current)
    setToast({ id: Date.now(), kind, text })
    toastTimer.current = window.setTimeout(() => setToast(null), 3400)
  }, [])

  const updateSettings = useCallback((patch: Partial<Settings>) => {
    setSettings((previous) => ({ ...previous, ...patch }))
    if ('baseUrl' in patch) setHealth(null)
  }, [])

  const checkConnection = useCallback(async (announce = true) => {
    setChecking(true)
    const next = await checkHealth(settings.baseUrl)
    setHealth(next)
    setChecking(false)
    if (announce) {
      notify(next.ok ? 'success' : 'error', next.ok ? `连接正常 · ${Math.round(next.durationMs)}ms` : `连接失败：${next.message}`)
    }
  }, [notify, settings.baseUrl])

  // Base URL 变更后防抖探测，避免每次按键都请求；手动按钮仍会给出 toast。
  useEffect(() => {
    const timer = window.setTimeout(() => {
      void checkConnection(false)
    }, 600)
    return () => window.clearTimeout(timer)
  }, [checkConnection])

  const updateCurrentData = (value: string) => {
    setDrafts((previous) => ({ ...previous, [method.key]: value }))
  }

  const selectMethod = (next: MethodDefinition) => {
    if (next.key === method.key) return
    setMethodKey(next.key)
    setTid(generateTid())
    setResult(null)
    setLastRequest(null)
    updateSettings({ lastMethod: next.key })
  }

  const handleSend = async () => {
    if (!settings.appId || !settings.secret) {
      notify('error', '请先填写 App ID 与 Secret')
      return
    }
    const parsed = parseJsonObject(dataText)
    if (parsed.error) {
      notify('error', parsed.error)
      return
    }

    const requestTid = tid.trim() || generateTid()
    if (!tid.trim()) setTid(requestTid)
    const timestamp = Date.now()
    const preview: RequestPreview = {
      method: method.key,
      tid: requestTid,
      dataText,
      signed: settings.signKey.trim() !== '',
    }
    setLastRequest(preview)
    setSending(true)
    const next = await callGateway({
      baseUrl: settings.baseUrl,
      appId: settings.appId,
      secret: settings.secret,
      signKey: settings.signKey,
      method: method.key,
      data: parsed.value,
      tid: requestTid,
    })
    setSending(false)
    setResult(next)

    const entry: HistoryEntry = {
      id: `${timestamp}-${Math.random().toString(16).slice(2, 8)}`,
      at: new Date(timestamp).toISOString(),
      method: method.key,
      tid: requestTid,
      dataText,
      baseUrl: settings.baseUrl,
      appId: settings.appId,
      type: next.type,
      httpStatus: next.type === 'response' ? next.httpStatus : 0,
      code: next.type === 'response' ? next.code : -1,
      msg: next.type === 'response' ? next.msg : next.message,
      durationMs: next.durationMs,
      requestId: next.requestId,
    }
    setHistory((previous) => [entry, ...previous].slice(0, 50))

    if (next.type === 'network') {
      notify('error', next.message)
    } else if (next.code === 0) {
      notify('success', `调用成功 · ${Math.round(next.durationMs)}ms`)
    } else {
      // 信封失败时 HTTP 恒为 200，错误码与 requestId 是唯一排查依据。
      const detail = next.code === -1 ? next.msg : `${next.code} · ${next.msg || '调用失败'}`
      const rid = next.requestId ? ` · rid ${shortId(next.requestId, 12)}` : ''
      notify('info', `${detail}${rid}`)
    }
  }

  const replay = (entry: HistoryEntry) => {
    const target = METHODS.find((item) => item.key === entry.method)
    if (!target) {
      notify('error', `method ${entry.method} 已不在当前清单中`)
      return
    }
    setTab('debug')
    setMethodKey(target.key)
    setDrafts((previous) => ({ ...previous, [target.key]: entry.dataText }))
    setTid(entry.tid || generateTid())
    setResult(null)
    setLastRequest(null)
    updateSettings({ lastMethod: target.key })
    window.scrollTo({ top: 0, behavior: 'smooth' })
    notify('info', '已回填请求，可修改后重新发送')
  }

  const clearCredentials = () => {
    clearStoredCredentials()
    setSettings((previous) => ({ ...previous, appId: '', secret: '', signKey: '', adminToken: '' }))
    notify('success', '本地凭据已清除')
  }

  const disabledReason = !settings.appId
    ? '填写 App ID 后启用发送'
    : !settings.secret
      ? '填写 Secret 后启用发送'
      : ''

  return (
    <div className="app-shell">
      <header className="app-header">
        <div className="brand">
          <span className="brand-mark"><Braces size={19} /></span>
          <div className="brand-copy">
            <b>OpenAPI 控制台</b>
            <small>Zero Service · opengtw</small>
          </div>
        </div>

        <nav className="top-tabs" aria-label="主导航">
          {TABS.map((item) => {
            const Icon = item.icon
            return (
              <button
                type="button"
                key={item.key}
                className={tab === item.key ? 'active' : ''}
                onClick={() => {
                  setTab(item.key)
                  window.scrollTo({ top: 0, behavior: 'auto' })
                }}
              >
                <Icon size={15} />
                {item.label}
              </button>
            )
          })}
        </nav>

        <div className="header-right">
          <span className="env-badge"><FlaskConical size={13} /> 测试环境</span>
          <span className="header-note"><ShieldAlert size={13} /> 仅限测试使用</span>
        </div>
      </header>

      <main className="app-main">
        {tab === 'debug' && (
          <div className="page-stack">
            <section className="hero">
              <div>
                <span className="eyebrow">OPEN PLATFORM · PODMAN + MODBUS</span>
                <h1>开放平台调试台</h1>
                <p>统一转发 podman 与 Modbus 只读/写入接口，验证 Basic 鉴权、protojson 契约与统一错误信封。</p>
              </div>
              <div className="hero-badges">
                <span><TerminalSquare size={14} /> {METHODS.length} 个 method</span>
                <span><Info size={14} /> {MODBUS_METHOD_COUNT} Modbus I/O</span>
                <span><Info size={14} /> 120s 超时</span>
                <span><Braces size={14} /> protojson</span>
              </div>
            </section>

            <div className="debug-layout">
              <aside className="debug-sidebar">
                <ConnectionPanel
                  settings={settings}
                  onChange={updateSettings}
                  health={health}
                  checking={checking}
                  onCheck={() => void checkConnection()}
                  onClear={clearCredentials}
                />
                <MethodPicker methods={METHODS} activeKey={method.key} onSelect={selectMethod} />
              </aside>

              <div className="debug-main">
                <RequestPanel
                  method={method}
                  dataText={dataText}
                  onDataChange={updateCurrentData}
                  tid={tid}
                  onTidChange={setTid}
                  sending={sending}
                  disabledReason={disabledReason}
                  onSend={() => void handleSend()}
                  notify={notify}
                />
                <ResponsePanel result={result} sending={sending} request={lastRequest} notify={notify} />
                <HistoryPanel
                  entries={history}
                  onReplay={replay}
                  onClear={() => {
                    setHistory([])
                    notify('success', '请求历史已清空')
                  }}
                />
              </div>
            </div>
          </div>
        )}

        {tab === 'apps' && (
          <AppsPage settings={settings} onChange={updateSettings} notify={notify} />
        )}

        {tab === 'errors' && <ErrorsPage />}
      </main>

      <footer className="app-footer">
        <span>Zero Service · OpenAPI Console v1.0.0</span>
        <span>默认方法 {DEFAULT_METHOD.key} · 凭据仅存于本浏览器</span>
      </footer>

      <Toast message={toast} onClose={() => setToast(null)} />
    </div>
  )
}
