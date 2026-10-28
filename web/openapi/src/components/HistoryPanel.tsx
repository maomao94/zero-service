import { History, Inbox, RotateCcw, Search, Trash2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { formatDateTime, formatDuration, shortId } from '../lib/format'
import type { HistoryEntry } from '../types'
import ConfirmDialog from './ConfirmDialog'
import CopyButton from './CopyButton'

interface HistoryPanelProps {
  entries: HistoryEntry[]
  onReplay: (entry: HistoryEntry) => void
  onClear: () => void
}

type StatusFilter = 'all' | 'ok' | 'fail'

export default function HistoryPanel({ entries, onReplay, onClear }: HistoryPanelProps) {
  const [query, setQuery] = useState('')
  const [method, setMethod] = useState('all')
  const [status, setStatus] = useState<StatusFilter>('all')
  const [confirmOpen, setConfirmOpen] = useState(false)

  const methods = useMemo(() => Array.from(new Set(entries.map((entry) => entry.method))), [entries])

  const filtered = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    return entries.filter((entry) => {
      if (method !== 'all' && entry.method !== method) return false
      // 信封场景 HTTP 恒为 200，成功与否只看业务码；非信封统一为 code=-1。
      const ok = entry.type === 'response' && entry.code === 0
      if (status === 'ok' && !ok) return false
      if (status === 'fail' && ok) return false
      if (!keyword) return true
      return [entry.method, entry.tid, entry.msg, String(entry.code), entry.appId]
        .join(' ')
        .toLowerCase()
        .includes(keyword)
    })
  }, [entries, method, query, status])

  return (
    <section className="card history-card">
      <div className="card-heading history-heading">
        <div>
          <span className="eyebrow">历史</span>
          <h2>请求历史</h2>
        </div>
        <div className="history-heading-actions">
          <span className="count-badge">{entries.length} / 50</span>
          <button type="button" className="button ghost danger-text" onClick={() => setConfirmOpen(true)} disabled={!entries.length}>
            <Trash2 size={14} /> 清空
          </button>
        </div>
      </div>

      <div className="history-filters">
        <div className="search-box">
          <Search size={14} />
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="搜索 method、tid、错误码或消息"
            aria-label="搜索请求历史"
          />
        </div>
        <select value={method} onChange={(event) => setMethod(event.target.value)} aria-label="按 method 过滤">
          <option value="all">全部 method</option>
          {methods.map((item) => <option key={item} value={item}>{item}</option>)}
        </select>
        <select value={status} onChange={(event) => setStatus(event.target.value as StatusFilter)} aria-label="按状态过滤">
          <option value="all">全部状态</option>
          <option value="ok">仅成功</option>
          <option value="fail">仅失败</option>
        </select>
      </div>

      {!entries.length && (
        <div className="history-empty"><Inbox size={20} /><span>还没有请求记录，发送一次调用后会出现在这里。</span></div>
      )}

      {entries.length > 0 && !filtered.length && (
        <div className="history-empty"><History size={20} /><span>没有符合当前筛选条件的记录。</span></div>
      )}

      {filtered.length > 0 && (
        <div className="table-scroll">
          <table className="history-table">
            <thead>
              <tr>
                <th>时间</th>
                <th>Method</th>
                <th>结果</th>
                <th>耗时</th>
                <th>tid</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {filtered.map((entry) => {
                const ok = entry.type === 'response' && entry.code === 0
                return (
                  <tr key={entry.id}>
                    <td className="nowrap muted">{formatDateTime(entry.at)}</td>
                    <td><code className="method-tag">{entry.method}</code></td>
                    <td>
                      <span className={`history-status ${ok ? 'ok' : 'fail'}`}>
                        {entry.type === 'network'
                          ? '网络错误'
                          : entry.code === -1
                            ? `HTTP ${entry.httpStatus} · 非信封`
                            : `${entry.httpStatus} / ${entry.code}`}
                      </span>
                      <small className="history-msg" title={entry.msg}>{entry.msg || '—'}</small>
                    </td>
                    <td className="nowrap">{formatDuration(entry.durationMs)}</td>
                    <td>
                      <span className="tid-cell" title={entry.tid}>{shortId(entry.tid, 18)}</span>
                      <CopyButton text={entry.tid} label="" className="compact-copy" />
                    </td>
                    <td className="align-right">
                      <button type="button" className="text-button" onClick={() => onReplay(entry)}>
                        <RotateCcw size={12} /> 回填
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      <ConfirmDialog
        open={confirmOpen}
        title="清空请求历史？"
        message="仅清除当前浏览器 localStorage 中的记录，不会影响服务端。"
        confirmLabel="清空"
        danger
        onCancel={() => setConfirmOpen(false)}
        onConfirm={() => {
          onClear()
          setConfirmOpen(false)
        }}
      />
    </section>
  )
}
