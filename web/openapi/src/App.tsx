import { AppWindow, BookOpen, Braces, FlaskConical, ShieldAlert, TerminalSquare } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import ConnectionPanel from './components/ConnectionPanel'
import ConfirmDialog from './components/ConfirmDialog'
import HistoryPanel from './components/HistoryPanel'
import MethodPicker from './components/MethodPicker'
import RequestPanel from './components/RequestPanel'
import ResponsePanel, { type RequestPreview } from './components/ResponsePanel'
import Toast from './components/Toast'
import { checkHealth, callGateway, gatewayUrl, type GatewayCallInput, type HealthResult } from './lib/http'
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

interface PendingCall {
  input: GatewayCallInput
  preview: RequestPreview
}

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
  const [pane, setPane] = useState<'request' | 'response'>('request')
  const [connectionOpen, setConnectionOpen] = useState(!initialSettings.appId || !initialSettings.secret)
  const [pendingWrite, setPendingWrite] = useState<PendingCall | null>(null)
  const [errorCode, setErrorCode] = useState<number | null>(null)
  const debugContext = useRef(0)
  const healthSequence = useRef(0)
  const activeCall = useRef(false)

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

  const resetDebugResult = useCallback(() => {
    debugContext.current++
    setResult(null)
    setLastRequest(null)
    setPendingWrite(null)
    setPane('request')
  }, [])

  const updateSettings = useCallback((patch: Partial<Settings>) => {
    setSettings((previous) => ({ ...previous, ...patch }))
    if ('baseUrl' in patch) {
      healthSequence.current++
      setHealth(null)
      setChecking(false)
    }
    if ('baseUrl' in patch || 'appId' in patch || 'secret' in patch || 'signKey' in patch) resetDebugResult()
  }, [resetDebugResult])

  const checkConnection = useCallback(async (announce = true) => {
    const sequence = ++healthSequence.current
    setChecking(true)
    const next = await checkHealth(settings.baseUrl)
    if (sequence !== healthSequence.current) return
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
    resetDebugResult()
    updateSettings({ lastMethod: next.key })
  }

  const handleSend = () => {
    if (activeCall.current || pendingWrite) return
    if (!settings.appId.trim() || !settings.secret.trim()) {
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
    const preview: RequestPreview = {
      method: method.key,
      tid: requestTid,
      dataText,
      signed: settings.signKey.trim() !== '',
    }
    const input: GatewayCallInput = {
      baseUrl: settings.baseUrl,
      appId: settings.appId.trim(),
      secret: settings.secret.trim(),
      signKey: settings.signKey.trim(),
      method: method.key,
      data: parsed.value,
      tid: requestTid,
    }
    if (method.write) {
      setPendingWrite({ input, preview })
      return
    }
    void sendCall({ input, preview })
  }

  const sendCall = async ({ input, preview }: PendingCall) => {
    if (activeCall.current) return
    activeCall.current = true
    const context = debugContext.current
    const timestamp = Date.now()
    setResult(null)
    setLastRequest(preview)
    setPane('response')
    setSending(true)
    const next = await callGateway(input)
    activeCall.current = false
    setSending(false)

    const entry: HistoryEntry = {
      id: `${timestamp}-${Math.random().toString(16).slice(2, 8)}`,
      at: new Date(timestamp).toISOString(),
      method: input.method,
      tid: input.tid,
      dataText: preview.dataText,
      baseUrl: input.baseUrl,
      appId: input.appId,
      type: next.type,
      httpStatus: next.type === 'response' ? next.httpStatus : 0,
      code: next.type === 'response' ? next.code : -1,
      msg: next.type === 'response' ? next.msg : next.message,
      durationMs: next.durationMs,
      requestId: next.requestId,
    }
    setHistory((previous) => [entry, ...previous].slice(0, 50))
    if (context !== debugContext.current) return
    setResult(next)

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
    resetDebugResult()
    updateSettings({ lastMethod: target.key })
    window.scrollTo({ top: 0, behavior: 'smooth' })
    notify('info', '已回填请求，可修改后重新发送')
  }

  const clearCredentials = () => {
    clearStoredCredentials()
    updateSettings({ appId: '', secret: '', signKey: '', adminToken: '' })
    notify('success', '本地凭据已清除')
  }

  const disabledReason = !settings.appId.trim()
    ? '填写 App ID 后启用发送'
    : !settings.secret.trim()
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
                aria-current={tab === item.key ? 'page' : undefined}
                onClick={() => {
                  setErrorCode(null)
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
          <button
            type="button"
            className="header-health"
            data-state={health ? (health.ok ? 'ok' : 'bad') : 'checking'}
            onClick={() => void checkConnection()}
            title="重新检测网关连通性"
          >
            <i aria-hidden="true" />
            {health ? (health.ok ? `网关可达 · ${Math.round(health.durationMs)}ms` : '网关不可达') : '检测中'}
          </button>
          <span className="env-badge"><FlaskConical size={13} /> 测试环境</span>
          <span className="header-note"><ShieldAlert size={13} /> 仅限测试使用</span>
        </div>
      </header>

      <main className="app-main">
        {tab === 'debug' && (
          <div className="page-stack">
            <section className="debug-toolbar">
              <div>
                <h1>开放平台调试台</h1>
                <code>{gatewayUrl(settings.baseUrl || window.location.origin)}</code>
              </div>
              <details className="console-disclosure connection-disclosure" open={connectionOpen} onToggle={(event) => setConnectionOpen(event.currentTarget.open)}>
                <summary>连接设置 · {disabledReason ? '凭据未填写' : settings.signKey.trim() ? 'Basic + 签名' : 'Basic'}</summary>
                <ConnectionPanel
                  settings={settings}
                  onChange={updateSettings}
                  health={health}
                  checking={checking}
                  onCheck={() => void checkConnection()}
                  onClear={clearCredentials}
                />
              </details>
            </section>

            <div className="debug-layout">
              <aside className="debug-sidebar">
                <MethodPicker methods={METHODS} activeKey={method.key} onSelect={selectMethod} />
              </aside>

              <div className="debug-main" data-pane={pane}>
                <div className="pane-switch" role="group" aria-label="调试面板">
                  <button type="button" className="button secondary" aria-pressed={pane === 'request'} onClick={() => setPane('request')}>请求</button>
                  <button type="button" className="button secondary" aria-pressed={pane === 'response'} onClick={() => setPane('response')}>响应</button>
                </div>
                <div className="request-pane">
                  <RequestPanel
                    method={method}
                    dataText={dataText}
                    onDataChange={updateCurrentData}
                    tid={tid}
                    onTidChange={setTid}
                    sending={sending}
                    disabledReason={disabledReason}
                    signed={settings.signKey.trim() !== ''}
                    onSend={handleSend}
                    notify={notify}
                  />
                </div>
                <div className="response-pane">
                  <ResponsePanel result={result} sending={sending && lastRequest !== null} request={lastRequest} notify={notify} onLookupError={(code) => {
                    setErrorCode(code)
                    setTab('errors')
                    window.scrollTo(0, 0)
                  }} />
                </div>
              </div>
            </div>
            <details className="console-disclosure history-disclosure">
              <summary>请求历史（{history.length}）</summary>
              <HistoryPanel
                entries={history}
                onReplay={replay}
                onClear={() => {
                  setHistory([])
                  notify('success', '请求历史已清空')
                }}
              />
            </details>
          </div>
        )}

        {tab === 'apps' && (
          <AppsPage settings={settings} onChange={updateSettings} notify={notify} />
        )}

        {tab === 'errors' && <ErrorsPage key={errorCode ?? 'all'} initialCode={errorCode} />}
      </main>

      <footer className="app-footer">
        <span>Zero Service · OpenAPI Console v1.0.0</span>
        <span>默认方法 {DEFAULT_METHOD.key} · 凭据仅存于本浏览器</span>
      </footer>

      <Toast message={toast} onClose={() => setToast(null)} />
      {pendingWrite && (
        <ConfirmDialog
          open
          title="确认发送写请求"
          message="请求可能改变设备状态。核对目标和参数后再发送。"
          confirmLabel="确认写入"
          danger
          onCancel={() => setPendingWrite(null)}
          onConfirm={() => {
            setPendingWrite(null)
            void sendCall(pendingWrite)
          }}
        >
          <dl className="write-target">
            <dt>网关</dt><dd>{gatewayUrl(pendingWrite.input.baseUrl || window.location.origin)}</dd>
            <dt>应用</dt><dd>{pendingWrite.input.appId}</dd>
            <dt>方法</dt><dd>{pendingWrite.input.method}</dd>
          </dl>
          <pre className="write-payload">{JSON.stringify(pendingWrite.input.data, null, 2)}</pre>
        </ConfirmDialog>
      )}
    </div>
  )
}
