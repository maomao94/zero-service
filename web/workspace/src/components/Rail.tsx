import { Moon, Sun, X } from 'lucide-react'
import { modules } from '../data/modules'
import type { ModuleDefinition } from '../data/modules'
import { latencyText } from '../lib/format'
import type { ActivityEvent, ModuleHealth } from '../hooks/useHealth'
import type { Theme } from '../hooks/useTheme'
import { clockTime } from '../lib/format'
import { searchShortcut } from '../lib/platform'
import { BrandMark } from './BrandMark'

const moduleName = new Map(modules.map((module) => [module.id, module.name]))

interface RailProps {
  health: Record<string, ModuleHealth>
  open: boolean
  theme: Theme
  online: number
  total: number
  average: number | null
  checking: boolean
  nextCheckAt: number
  now: number
  events: ActivityEvent[]
  onOpenModule: (module: ModuleDefinition) => void
  onToggleTheme: () => void
  onClose: () => void
}

export function Rail({
  health,
  open,
  theme,
  online,
  total,
  average,
  checking,
  nextCheckAt,
  now,
  events,
  onOpenModule,
  onToggleTheme,
  onClose,
}: RailProps) {
  const seconds = Math.max(0, Math.ceil((nextCheckAt - now) / 1000))

  return (
    <aside className="rail" data-open={open} aria-label="工作台侧栏">
      <div className="rail-brand">
        <BrandMark />
        <div className="rail-brand-copy">
          <strong>Zero 工作台</strong>
          <span>统一入口 · 子系统导航</span>
        </div>
        <button type="button" className="rail-close" onClick={onClose} aria-label="关闭导航">
          <X size={16} />
        </button>
      </div>

      <div className="rail-block">
        <p className="rail-label">系统状态</p>
        <ul className="rail-modules">
          {modules.map((module) => {
            const state = health[module.id]
            return (
              <li key={module.id}>
                <button
                  type="button"
                  className="rail-module"
                  onClick={() => onOpenModule(module)}
                  title={`打开 ${module.name}`}
                >
                  <span className="rail-dot" data-status={state?.status ?? 'checking'} />
                  <span className="rail-module-name">{module.name}</span>
                  <span className="rail-module-meta">
                    {state?.status === 'offline' ? '未启动' : latencyText(state?.latency ?? null)}
                  </span>
                </button>
              </li>
            )
          })}
        </ul>
      </div>

      <div className="rail-block">
        <p className="rail-label">运行概况</p>
        <dl className="rail-stats">
          <div>
            <dt>在线模块</dt>
            <dd>
              {online}
              <span>/{total}</span>
            </dd>
          </div>
          <div>
            <dt>平均响应</dt>
            <dd>
              {average ?? '—'}
              {average !== null && <span> ms</span>}
            </dd>
          </div>
          <div>
            <dt>下次检测</dt>
            <dd>
              {checking ? '—' : seconds}
              {!checking && <span> s</span>}
            </dd>
          </div>
        </dl>
      </div>

      <div className="rail-block">
        <p className="rail-label">最近变化</p>
        {events.length === 0 ? (
          <p className="rail-empty">暂无状态变化</p>
        ) : (
          <ul className="rail-events">
            {events.slice(0, 3).map((event) => (
              <li key={event.id}>
                <time dateTime={new Date(event.at).toISOString()}>{clockTime(event.at)}</time>
                <span className="rail-event-dot" data-kind={event.kind} aria-hidden="true" />
                <span className="rail-event-text">
                  {event.kind === 'boot'
                    ? '工作台已就绪'
                    : `${moduleName.get(event.moduleId) ?? event.moduleId}${
                        event.kind === 'online' ? '恢复在线' : '连接中断'
                      }`}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="rail-block">
        <p className="rail-label">快捷键</p>
        <dl className="rail-stats">
          <div>
            <dt>搜索模块</dt>
            <dd>
              <kbd>{searchShortcut}</kbd>
            </dd>
          </div>
          <div>
            <dt>关闭面板</dt>
            <dd>
              <kbd>esc</kbd>
            </dd>
          </div>
        </dl>
      </div>

      <div className="rail-foot">
        <button
          type="button"
          className="rail-theme"
          onClick={onToggleTheme}
          aria-pressed={theme === 'dark'}
        >
          {theme === 'dark' ? <Sun size={14} /> : <Moon size={14} />}
          <span>{theme === 'dark' ? '切换浅色主题' : '切换深色主题'}</span>
        </button>
      </div>
    </aside>
  )
}
