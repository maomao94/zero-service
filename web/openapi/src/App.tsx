import { AppWindow, BookOpen, Braces, FlaskConical, ShieldAlert, TerminalSquare } from 'lucide-react'
import { useCallback, useEffect, useReducer, useRef, useState } from 'react'
import ConnectionPanel from './components/ConnectionPanel'
import ConfirmDialog from './components/ConfirmDialog'
import HistoryPanel from './components/HistoryPanel'
import MethodPicker from './components/MethodPicker'
import RequestPanel from './components/RequestPanel'
import ResponsePanel, { type RequestPreview } from './components/ResponsePanel'
import Toast from './components/Toast'
import { checkHealth, callGateway, gatewayUrl, type GatewayCallInput, type HealthResult } from './lib/http'
import { generateTid, parseJsonObject } from './lib/format'
import { DEFAULT_METHOD, METHODS, findMethod } from './lib/methods'
import { exchangeReducer, type CallSnapshot } from './lib/workbench'
import {
  clearStoredCredentials,
  loadHistory,
  loadSettings,
  saveHistory,
  saveSettings,
} from './lib/storage'
import type {
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
  const [exchange, dispatchExchange] = useReducer(exchangeReducer, { phase: 'idle' })
  const [health, setHealth] = useState<HealthResult | null>(null)
  const [checking, setChecking] = useState(false)
  const [pane, setPane] = useState<'request' | 'response'>('request')
  const [connectionOpen, setConnectionOpen] = useState(false)
  const [errorCode, setErrorCode] = useState<number | null>(null)
  const healthSequence = useRef(0)
  const activeCall = useRef<AbortController | null>(null)
  const [inFlight, setInFlight] = useState<CallSnapshot | null>(null)
  const pendingWrite = exchange.phase === 'confirm' ? exchange.call : null
  const result = exchange.phase === 'done' ? exchange.result : null
  const visibleCall = exchange.phase === 'sending' || exchange.phase === 'done' ? exchange.call : null
  const lastRequest: RequestPreview | null = visibleCall && {
    method: visibleCall.input.method,
    tid: visibleCall.input.tid,
    dataText: visibleCall.dataText,
    signed: Boolean(visibleCall.input.signKey),
  }

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
    dispatchExchange({ type: 'reset' })
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
    const input: GatewayCallInput = {
      baseUrl: settings.baseUrl,
      appId: settings.appId.trim(),
      secret: settings.secret.trim(),
      signKey: settings.signKey.trim(),
      method: method.key,
      data: parsed.value,
      tid: requestTid,
    }
    const call = { id: generateTid(), startedAt: Date.now(), input, dataText }
    if (method.write) {
      dispatchExchange({ type: 'prepare', call })
      return
    }
    void sendCall(call)
  }

  const sendCall = async (prepared: CallSnapshot) => {
    if (activeCall.current) return
    const controller = new AbortController()
    activeCall.current = controller
    const call = { ...prepared, startedAt: Date.now() }
    const { input, startedAt } = call
    dispatchExchange({ type: 'send', call })
    setPane('response')
    setInFlight(call)
    const next = await callGateway({ ...input, signal: controller.signal })
    activeCall.current = null
    setInFlight(null)

    const entry: HistoryEntry = {
      id: call.id,
      at: new Date(startedAt).toISOString(),
      method: input.method,
      tid: input.tid,
      dataText: call.dataText,
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
    dispatchExchange({ type: 'complete', id: call.id, result: next })
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
    setTid(generateTid())
    resetDebugResult()
    updateSettings({ lastMethod: target.key })
    window.scrollTo({ top: 0, behavior: 'smooth' })
    const sameTarget = entry.baseUrl === settings.baseUrl && entry.appId === settings.appId
    notify('info', sameTarget ? '已回填参数并生成新 tid' : '已回填参数。当前网关或应用与历史记录不同，请核对后发送')
  }

  const clearCredentials = () => {
    clearStoredCredentials()
    updateSettings({ appId: '', secret: '', signKey: '', adminToken: '' })
    setConnectionOpen(true)
    notify('success', '本地凭据已清除')
  }

  const disabledReason = !settings.appId.trim()
    ? '填写 App ID 后启用发送'
    : !settings.secret.trim()
      ? '填写 Secret 后启用发送'
      : ''

  return (
    <div className="app-shell">
      <a className="skip-link" href="#main-content">跳到主要内容</a>
      <header className="app-header">
        <div className="brand">
          <span className="brand-mark"><Braces size={19} /></span>
          <div className="brand-copy">
            <b>开放平台控制台</b>
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
            data-state={checking ? 'checking' : health ? (health.ok ? 'ok' : 'bad') : 'idle'}
            onClick={() => void checkConnection()}
            title="重新检测网关连通性"
          >
            <i aria-hidden="true" />
            <span aria-live="polite">{checking ? '检测中…' : health ? (health.ok ? `网关可达 · ${Math.round(health.durationMs)} ms` : '网关不可达') : '未检测'}</span>
          </button>
          <span className="env-badge"><FlaskConical size={13} /> 测试环境</span>
          <span className="header-note"><ShieldAlert size={13} /> 仅限测试使用</span>
        </div>
      </header>

      <main className="app-main" id="main-content" tabIndex={-1}>
        {tab === 'debug' && (
          <div className="page-stack">
            <section className="debug-toolbar">
              <div>
                <h1>API 调试</h1>
                <div className="gateway-target"><b>POST</b><code>{gatewayUrl(settings.baseUrl || window.location.origin)}</code></div>
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
                  <button type="button" className="button secondary" aria-controls="request-pane" aria-pressed={pane === 'request'} onClick={() => setPane('request')}>请求</button>
                  <button type="button" className="button secondary" aria-controls="response-pane" aria-pressed={pane === 'response'} onClick={() => setPane('response')}>响应</button>
                </div>
                <div className="request-pane" id="request-pane">
                  <RequestPanel
                    method={method}
                    dataText={dataText}
                    onDataChange={updateCurrentData}
                    tid={tid}
                    onTidChange={setTid}
                    sending={exchange.phase === 'sending'}
                    busyReason={inFlight && exchange.phase !== 'sending' ? `正在等待 ${inFlight.input.method}` : ''}
                    onCancelWaiting={() => activeCall.current?.abort()}
                    disabledReason={disabledReason}
                    signed={settings.signKey.trim() !== ''}
                    onSend={handleSend}
                    notify={notify}
                  />
                </div>
                <div className="response-pane" id="response-pane">
                  <ResponsePanel result={result} sending={exchange.phase === 'sending'} request={lastRequest} notify={notify} onCancel={() => activeCall.current?.abort()} onLookupError={(code) => {
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

        {tab === 'errors' && <ErrorsPage key={errorCode ?? 'all'} initialCode={errorCode} onBack={errorCode === null ? undefined : () => setTab('debug')} />}
      </main>

      <footer className="app-footer">
        <span>Zero Service · 开放平台控制台 v1.0.0</span>
        <span>默认方法 {DEFAULT_METHOD.key} · 凭据仅存于本浏览器</span>
      </footer>

      <Toast message={toast} onClose={() => setToast(null)} />
      {pendingWrite && (
        <ConfirmDialog
          open
          title="确认执行接口测试"
          message="此接口可能修改数据或设备状态。请确认目标环境和请求参数。"
          confirmLabel="执行测试请求"
          danger
          onCancel={() => dispatchExchange({ type: 'reset' })}
          onConfirm={() => {
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
