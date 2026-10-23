import {
  AlertTriangle,
  ArrowLeftRight,
  Braces,
  Clock3,
  FileJson,
  Gauge,
  Hash,
  LoaderCircle,
  Radio,
  ServerCrash,
  ShieldAlert,
  Sparkles,
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { errorTone, lookupError } from '../lib/errors'
import { formatBytes, formatDuration, prettyValue } from '../lib/format'
import type { GatewayCallResult, ToastKind } from '../types'
import CopyButton from './CopyButton'
import JsonHighlight from './JsonHighlight'

export interface RequestPreview {
  method: string
  tid: string
  dataText: string
  timestamp: number
}

interface ResponsePanelProps {
  result: GatewayCallResult | null
  sending: boolean
  request: RequestPreview | null
  notify: (kind: ToastKind, text: string) => void
}

type ResponseTab = 'data' | 'raw' | 'error' | 'request'

export default function ResponsePanel({ result, sending, request, notify }: ResponsePanelProps) {
  const [tab, setTab] = useState<ResponseTab>('data')

  useEffect(() => {
    // 信封场景 HTTP 恒为 200，失败与否只看业务码；非信封为 code=-1。
    if (result?.type === 'response' && result.code !== 0) {
      setTab('error')
      return
    }
    setTab('data')
  }, [result])

  const code = result?.type === 'response' ? result.code : -1
  const doc = lookupError(code)
  const requestJson = useMemo(() => request ? prettyValue({
    method: request.method,
    data: safeParse(request.dataText),
    tid: request.tid,
    timestamp: request.timestamp,
  }) : '', [request])

  const copyValue = result?.type === 'response'
    ? tab === 'raw'
      ? result.raw
      : tab === 'request'
        ? requestJson
        : tab === 'error'
          ? result.code === -1
            ? result.msg
            : [doc?.title, doc?.description, doc?.suggestion].filter(Boolean).join('\n')
          : prettyValue(result.data)
    : ''

  return (
    <section className="card response-card">
      <div className="card-heading response-heading">
        <div>
          <span className="eyebrow">RESPONSE</span>
          <h2>响应结果</h2>
        </div>
        {result?.type === 'response' && (
          <CopyButton
            text={copyValue}
            onResult={(ok) => notify(ok ? 'success' : 'error', ok ? '已复制' : '复制失败')}
          />
        )}
      </div>

      {sending && (
        <div className="response-loading">
          <div className="loading-orbit"><LoaderCircle size={26} className="spin" /></div>
          <b>正在等待 openapi 返回…</b>
          <span>最长等待 120 秒，慢请求会占用连接。</span>
        </div>
      )}

      {!sending && !result && (
        <div className="response-empty">
          <div className="empty-icon"><ArrowLeftRight size={24} /></div>
          <b>发起第一次调用</b>
          <p>选择左侧 method，确认 data 后发送。成功与失败都会保留在请求历史中。</p>
          <div className="empty-hints">
            <span><Radio size={13} /> 响应头 X-Request-Id</span>
            <span><Gauge size={13} /> HTTP 状态 / 业务错误码 / 耗时</span>
            <span><FileJson size={13} /> protojson 原始响应</span>
          </div>
        </div>
      )}

      {!sending && result?.type === 'network' && (
        <div className="network-error">
          <div className="network-error-head">
            <span className="error-icon"><ServerCrash size={20} /></span>
            <div>
              <b>请求未到达网关</b>
              <p>{result.message}</p>
            </div>
          </div>
          <div className="network-error-grid">
            <span>1. 确认 opengtw 已在 11005 端口启动</span>
            <span>2. 本地开发建议 Base URL 留空，走 vite 代理</span>
            <span>3. 跨域直连时确认网关已开启 CORS</span>
            <span>4. 生产环境确认 nginx 已转发 /v1/</span>
          </div>
          <div className="network-error-foot"><Clock3 size={13} /> 耗时 {formatDuration(result.durationMs)}</div>
        </div>
      )}

      {!sending && result?.type === 'response' && (
        <>
          <div className="result-strip">
            <span className={`result-chip http-${toneOfHttp(result.httpStatus)}`}>
              <b>HTTP</b> {result.httpStatus}
            </span>
            <span className={`result-chip code-${errorTone(result.code)}`}>
              <b>CODE</b> {result.code}
            </span>
            <span className="result-chip neutral"><Clock3 size={13} /> {formatDuration(result.durationMs)}</span>
            <span className="result-chip neutral"><Hash size={13} /> {formatBytes(result.sizeBytes)}</span>
            {result.requestId && (
              <span className="result-chip request-id" title={result.requestId}>
                <b>Request ID</b> {result.requestId}
              </span>
            )}
          </div>

          <div className="response-tabs" role="tablist">
            <button type="button" role="tab" aria-selected={tab === 'data'} className={tab === 'data' ? 'active' : ''} onClick={() => setTab('data')}>
              <Braces size={13} /> 数据
            </button>
            <button type="button" role="tab" aria-selected={tab === 'raw'} className={tab === 'raw' ? 'active' : ''} onClick={() => setTab('raw')}>
              <FileJson size={13} /> 原始响应
            </button>
            {result.code !== 0 && (
              <button type="button" role="tab" aria-selected={tab === 'error'} className={tab === 'error' ? 'active' : ''} onClick={() => setTab('error')}>
                <ShieldAlert size={13} /> 错误说明
              </button>
            )}
            <button type="button" role="tab" aria-selected={tab === 'request'} className={tab === 'request' ? 'active' : ''} onClick={() => setTab('request')}>
              <ArrowLeftRight size={13} /> 请求
            </button>
          </div>

          <div className="response-body">
            {tab === 'data' && (
              result.data === undefined
                ? <div className="json-empty">响应中没有 data 字段</div>
                : <JsonHighlight text={prettyValue(result.data)} />
            )}
            {tab === 'raw' && <JsonHighlight text={result.raw} emptyText="响应体为空" />}
            {tab === 'error' && (
              <div className="error-detail">
                <div className="error-detail-head">
                  <span className={`error-detail-icon ${errorTone(result.code)}`}><AlertTriangle size={19} /></span>
                  <div>
                    <b>
                      {result.code === -1
                        ? `HTTP ${result.httpStatus} · 非信封响应`
                        : doc
                          ? `${result.code} · ${doc.title}`
                          : `未收录错误码 ${result.code}`}
                    </b>
                    <p>
                      {result.code === -1
                        ? result.msg || '网关返回了无法解析的响应，未获得业务错误码。'
                        : doc?.description ?? result.msg ?? '服务端返回了未收录在控制台错误码表中的错误。'}
                    </p>
                  </div>
                </div>
                {doc?.suggestion && (
                  <div className="error-suggestion">
                    <Sparkles size={15} />
                    <span>{doc.suggestion}</span>
                  </div>
                )}
                {doc?.folded && (
                  <div className="error-note">该错误属于基础设施类，开放面已折叠为 100999；真实原因请查看服务端日志。</div>
                )}
                <div className="error-raw">
                  <label>服务端消息</label>
                  <pre>{result.msg || '—'}</pre>
                </div>
              </div>
            )}
            {tab === 'request' && (
              request
                ? <JsonHighlight text={requestJson} />
                : <div className="json-empty">本次没有可回显的请求数据</div>
            )}
          </div>
        </>
      )}
    </section>
  )
}

function safeParse(text: string): unknown {
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}

function toneOfHttp(status: number): 'ok' | 'warn' | 'error' {
  if (status >= 200 && status < 300) return 'ok'
  if (status >= 400 && status < 500) return 'warn'
  return 'error'
}
