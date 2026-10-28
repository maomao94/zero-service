import { CornerDownLeft, Search, X } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { Fragment, useEffect, useMemo, useRef, useState } from 'react'

export interface CommandItem {
  id: string
  label: string
  hint: string
  group: string
  icon: LucideIcon
  run: () => void
}

interface CommandPaletteProps {
  open: boolean
  items: CommandItem[]
  onClose: () => void
}

export function CommandPalette({ open, items, onClose }: CommandPaletteProps) {
  const [query, setQuery] = useState('')
  const [active, setActive] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)

  const filtered = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    if (!keyword) return items
    return items.filter((item) =>
      `${item.label} ${item.hint} ${item.group}`.toLowerCase().includes(keyword),
    )
  }, [items, query])

  useEffect(() => {
    if (!open) return
    const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null
    setQuery('')
    setActive(0)
    const frame = requestAnimationFrame(() => inputRef.current?.focus())
    return () => {
      cancelAnimationFrame(frame)
      previous?.focus()
    }
  }, [open])

  useEffect(() => {
    setActive(0)
  }, [query])

  useEffect(() => {
    if (!open) return
    const node = listRef.current?.querySelector<HTMLElement>(`[data-index="${active}"]`)
    node?.scrollIntoView({ block: 'nearest' })
  }, [active, open])

  useEffect(() => {
    if (!open) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        onClose()
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [open, onClose])

  if (!open) return null

  const run = (item: CommandItem | undefined) => {
    if (!item) return
    onClose()
    item.run()
  }

  const onInputKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      setActive((index) => (filtered.length ? (index + 1) % filtered.length : 0))
      return
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault()
      setActive((index) => (filtered.length ? (index - 1 + filtered.length) % filtered.length : 0))
      return
    }
    if (event.key === 'Enter') {
      event.preventDefault()
      run(filtered[active])
    }
  }

  return (
    <div
      className="palette-scrim"
      onMouseDown={(event) => {
        if (event.target === event.currentTarget) onClose()
      }}
    >
      <div className="palette" role="dialog" aria-modal="true" aria-label="搜索模块与端点">
        <div className="palette-search">
          <Search size={16} aria-hidden="true" />
          <input
            ref={inputRef}
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            onKeyDown={onInputKeyDown}
            placeholder="搜索模块、端点或操作"
            aria-label="搜索模块、端点或操作"
            role="combobox"
            aria-expanded="true"
            aria-controls="palette-list"
            aria-activedescendant={filtered[active] ? `palette-item-${filtered[active].id}` : undefined}
            aria-autocomplete="list"
          />
          <kbd>esc</kbd>
          <button type="button" className="palette-close" onClick={onClose} aria-label="关闭搜索">
            <X size={15} />
          </button>
        </div>

        <div className="palette-list" id="palette-list" role="listbox" aria-label="搜索结果" ref={listRef}>
          {filtered.length === 0 && <p className="palette-empty">没有匹配项，换个关键词试试</p>}
          {filtered.map((item, index) => {
            const Icon = item.icon
            const showGroup = index === 0 || filtered[index - 1].group !== item.group
            return (
              <Fragment key={item.id}>
                {showGroup && <p className="palette-group">{item.group}</p>}
                <button
                  type="button"
                  role="option"
                  id={`palette-item-${item.id}`}
                  data-index={index}
                  aria-selected={index === active}
                  className="palette-item"
                  onMouseEnter={() => setActive(index)}
                  onClick={() => run(item)}
                >
                  <Icon size={15} aria-hidden="true" />
                  <span className="palette-label">{item.label}</span>
                  <span className="palette-hint mono">{item.hint}</span>
                </button>
              </Fragment>
            )
          })}
        </div>

        <footer className="palette-foot">
          <span>
            <kbd>↑</kbd>
            <kbd>↓</kbd> 选择
          </span>
          <span>
            <CornerDownLeft size={12} aria-hidden="true" /> 打开
          </span>
          <span>
            <kbd>esc</kbd> 关闭
          </span>
        </footer>
      </div>
    </div>
  )
}
