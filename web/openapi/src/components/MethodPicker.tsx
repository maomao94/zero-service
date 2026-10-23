import { Activity, Box, Cpu, Layers, ListTree, PenLine, Search } from 'lucide-react'
import { useMemo, useState } from 'react'
import { METHOD_GROUP_LABELS, METHOD_GROUP_ORDER } from '../lib/methods'
import type { MethodDefinition } from '../types'

interface MethodPickerProps {
  methods: MethodDefinition[]
  activeKey: string
  onSelect: (method: MethodDefinition) => void
}

const ICONS = {
  'pod.list': ListTree,
  'pod.get': Box,
  'pod.stats': Activity,
  'image.list': Layers,
} as const

export default function MethodPicker({ methods, activeKey, onSelect }: MethodPickerProps) {
  const [query, setQuery] = useState('')

  const groups = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    return METHOD_GROUP_ORDER.map((category) => ({
      category,
      label: METHOD_GROUP_LABELS[category],
      items: methods.filter((method) => {
        if (method.category !== category) return false
        if (!keyword) return true
        return [method.key, method.label, method.description, method.rpc]
          .join(' ')
          .toLowerCase()
          .includes(keyword)
      }),
    })).filter((group) => group.items.length > 0)
  }, [methods, query])

  return (
    <section className="card method-card">
      <div className="card-heading compact">
        <div>
          <span className="eyebrow">METHODS</span>
          <h2>开放方法</h2>
        </div>
        <span className="count-badge">{methods.length}</span>
      </div>

      <div className="method-search">
        <Search size={14} />
        <input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="搜索 method 或能力"
          aria-label="搜索开放方法"
        />
      </div>

      <div className="method-list grouped">
        {groups.map((group) => (
          <div className="method-group" key={group.category}>
            <div className="method-group-title">
              <span>{group.label}</span>
              <span>{group.items.length}</span>
            </div>
            {group.items.map((method) => {
              const Icon = ICONS[method.key as keyof typeof ICONS]
                ?? (method.write ? PenLine : method.category === 'modbus-read' ? Cpu : Box)
              return (
                <button
                  type="button"
                  key={method.key}
                  className={`method-item ${activeKey === method.key ? 'active' : ''}`}
                  onClick={() => onSelect(method)}
                >
                  <span className={`method-icon ${method.write ? 'write' : ''}`}><Icon size={17} /></span>
                  <span className="method-copy">
                    <b>{method.key}</b>
                    <small>{method.label}</small>
                  </span>
                  {method.write && <span className="method-write-tag">写</span>}
                  <span className="method-arrow">→</span>
                </button>
              )
            })}
          </div>
        ))}
        {!groups.length && <div className="method-empty">没有匹配的 method。</div>}
      </div>
      <p className="method-footnote">Podman 4 个，Modbus I/O 15 个；配置管理与本地换算暂不开放。</p>
    </section>
  )
}
