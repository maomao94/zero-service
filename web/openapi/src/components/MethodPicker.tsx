import { Search, X } from 'lucide-react'
import { useState } from 'react'
import { METHOD_GROUPS } from '../lib/methods'
import type { MethodDefinition } from '../types'

interface MethodPickerProps {
  methods: MethodDefinition[]
  activeKey: string
  onSelect: (method: MethodDefinition) => void
}

export default function MethodPicker({ methods, activeKey, onSelect }: MethodPickerProps) {
  const [query, setQuery] = useState('')
  const [selection, setSelection] = useState<{ service: string; forKey: string }>()
  const [expanded, setExpanded] = useState<Record<string, boolean>>({})
  const activeGroup = methods.find((method) => method.key === activeKey)?.category
  const service = (selection?.forKey === activeKey ? selection?.service : undefined)
    ?? (activeGroup && METHOD_GROUPS[activeGroup].service)
    ?? 'Podman'
  const keyword = query.trim().toLowerCase()
  const groups = Object.entries(METHOD_GROUPS).map(([category, group]) => ({
    ...group, category, items: methods.filter((method) => method.category === category),
  }))
  const services = new Set(groups.filter((group) => group.items.length).map((group) => group.service))
  const visible = groups.map((group) => ({
    ...group,
    items: group.items.filter((method) => keyword
      ? [method.key, method.label, method.rpc, group.service, ...method.fields.map((field) => field.name)].join(' ').toLowerCase().includes(keyword)
      : group.service === service),
  })).filter((group) => group.items.length)

  return (
    <section className="method-card" aria-label="接口目录">
      <div className="catalog-heading"><h2>接口目录</h2><span>{methods.length} 个接口</span></div>
      <div className="method-search">
        <Search size={16} aria-hidden="true" />
        <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="搜索所有接口" aria-label="搜索所有接口" />
        {query && <button type="button" className="catalog-clear" onClick={() => setQuery('')} aria-label="清空搜索"><X size={14} /></button>}
      </div>
      <label className="catalog-service">
        <span>服务</span>
        <select value={service} onChange={(event) => { setSelection({ service: event.target.value, forKey: activeKey }); setQuery('') }}>
          {Array.from(services, (name) => <option key={name}>{name}</option>)}
        </select>
      </label>
      <div className="method-list grouped">
        {visible.map((group, index) => {
          const activeService = activeGroup && METHOD_GROUPS[activeGroup].service === service
          const defaultOpen = activeService ? group.category === activeGroup : index === 0
          return (
            <details
              key={`${service}:${activeKey}:${group.category}:${Boolean(keyword)}`}
              className="method-group"
              open={Boolean(keyword) || (expanded[group.category] ?? defaultOpen)}
              onToggle={(event) => {
                if (keyword) return
                const open = event.currentTarget.open
                setExpanded((previous) => (previous[group.category] ?? defaultOpen) === open
                  ? previous
                  : { ...previous, [group.category]: open })
              }}
            >
              <summary className="method-group-title"><span>{keyword ? `${group.service} / ` : ''}{group.label}</span><span>{group.items.length}</span></summary>
              {group.items.map((method) => (
                <button
                  type="button"
                  key={method.key}
                  className={`method-item${activeKey === method.key ? ' active' : ''}`}
                  aria-current={activeKey === method.key ? 'true' : undefined}
                  onClick={() => {
                    onSelect(method)
                    setQuery('')
                    setSelection(undefined)
                    setExpanded({})
                  }}
                >
                  <span className="method-copy"><b>{method.label}</b><small>{method.key}</small></span>
                </button>
              ))}
            </details>
          )
        })}
        {!visible.length && <p className="method-empty">没有匹配的接口。试试接口名或字段名。</p>}
      </div>
    </section>
  )
}
