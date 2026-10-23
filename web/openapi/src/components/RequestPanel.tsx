import { AlertTriangle, Braces, CheckCircle2, Eraser, LoaderCircle, Play, RotateCcw, Wand2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { minifyJsonText, parseJsonObject, prettyJsonText } from '../lib/format'
import type { MethodDefinition, ToastKind } from '../types'
import CopyButton from './CopyButton'

interface RequestPanelProps {
  method: MethodDefinition
  dataText: string
  onDataChange: (value: string) => void
  tid: string
  onTidChange: (value: string) => void
  sending: boolean
  disabledReason: string
  onSend: () => void
  notify: (kind: ToastKind, text: string) => void
}

export default function RequestPanel({
  method,
  dataText,
  onDataChange,
  tid,
  onTidChange,
  sending,
  disabledReason,
  onSend,
  notify,
}: RequestPanelProps) {
  const [showFields, setShowFields] = useState(false)
  const parsed = useMemo(() => parseJsonObject(dataText), [dataText])
  const fieldCount = parsed.value && typeof parsed.value === 'object' ? Object.keys(parsed.value).length : 0
  const canSend = !sending && !parsed.error && !disabledReason

  const handleFormat = () => {
    try {
      onDataChange(prettyJsonText(dataText))
      notify('success', 'JSON 已格式化')
    } catch {
      notify('error', 'JSON 格式错误，无法格式化')
    }
  }

  const handleMinify = () => {
    try {
      onDataChange(minifyJsonText(dataText))
      notify('success', 'JSON 已压缩')
    } catch {
      notify('error', 'JSON 格式错误，无法压缩')
    }
  }

  return (
    <section className="card request-card">
      <div className="request-header">
        <div>
          <div className="request-title-row">
            <span className="method-key">{method.key}</span>
            <h2>{method.label}</h2>
            {method.write && <span className="write-badge">WRITE</span>}
          </div>
          <p className="request-desc">{method.description}</p>
          {method.write && (
            <div className="write-warning">
              <AlertTriangle size={15} />
              <span>这是写操作，会改变设备寄存器或线圈状态；请确认地址和值后再发送。</span>
            </div>
          )}
        </div>
        <span className="rpc-badge">{method.rpc}</span>
      </div>

      <div className="request-meta-grid">
        <div className="field compact-field">
          <label className="field-label" htmlFor="tid">请求 ID（tid）</label>
          <input
            id="tid"
            className="text-input mono"
            value={tid}
            onChange={(event) => onTidChange(event.target.value)}
            spellCheck={false}
          />
          <p className="field-hint">回显到响应头 X-Request-Id；留空时由网关生成。</p>
        </div>
        <div className="field compact-field">
          <span className="field-label">时间戳</span>
          <div className="static-field">发送时自动填充 Unix 毫秒</div>
          <p className="field-hint">本期只接收不校验，为后续签名预留。</p>
        </div>
      </div>

      <div className="editor-heading">
        <div className="editor-label">
          <Braces size={15} />
          <b>data</b>
          <span className="muted">protojson</span>
        </div>
        <div className="editor-actions">
          <button type="button" className="text-button" onClick={handleFormat}>格式化</button>
          <button type="button" className="text-button" onClick={handleMinify}>压缩</button>
          <button type="button" className="text-button" onClick={() => onDataChange(method.example)}>
            <RotateCcw size={12} /> 示例
          </button>
          <CopyButton text={dataText} label="复制" />
        </div>
      </div>

      <div className={`editor-shell ${parsed.error ? 'invalid' : ''}`}>
        <textarea
          className="json-editor"
          value={dataText}
          onChange={(event) => onDataChange(event.target.value)}
          onKeyDown={(event) => {
            if ((event.metaKey || event.ctrlKey) && event.key === 'Enter' && canSend) {
              event.preventDefault()
              onSend()
            }
          }}
          spellCheck={false}
          aria-label="data JSON 编辑器"
        />
        <div className="editor-status">
          <span className={parsed.error ? 'status-invalid' : 'status-valid'}>
            {parsed.error ? <Eraser size={12} /> : <CheckCircle2 size={12} />}
            {parsed.error ? 'JSON 无效' : `JSON 有效 · ${fieldCount} 个字段`}
          </span>
          <span className="muted">{dataText.length} 字符 · Ctrl/⌘ + Enter 发送</span>
        </div>
      </div>
      {parsed.error && <p className="inline-error">{parsed.error}</p>}

      <div className="request-tips">
        {method.tips.map((tip) => <span key={tip}>{tip}</span>)}
      </div>

      <div className="request-footer">
        <button type="button" className="text-button subtle" onClick={() => setShowFields((value) => !value)}>
          <Wand2 size={13} /> {showFields ? '收起字段说明' : '查看字段说明'}
        </button>
        <div className="send-area">
          {disabledReason && !sending && <span className="disabled-hint">{disabledReason}</span>}
          <button type="button" className="button primary send-button" onClick={onSend} disabled={!canSend}>
            {sending ? <LoaderCircle size={15} className="spin" /> : <Play size={15} />}
            {sending ? '请求中…' : method.write ? '发送写请求' : '发送请求'}
          </button>
        </div>
      </div>

      {showFields && (
        <div className="field-table">
          <div className="field-table-head">
            <span>字段</span><span>类型</span><span>说明</span>
          </div>
          {method.fields.map((field) => (
            <div className="field-table-row" key={field.name}>
              <code>{field.name}{field.required ? ' *' : ''}</code>
              <span>{field.type}</span>
              <span>{field.note}</span>
            </div>
          ))}
        </div>
      )}
    </section>
  )
}
