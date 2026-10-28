import { ArrowUpRight, Copy } from 'lucide-react'
import type { CSSProperties } from 'react'
import type { ModuleDefinition } from '../data/modules'
import { HISTORY_LIMIT } from '../hooks/useHealth'
import type { HealthStatus, ModuleHealth, Sample } from '../hooks/useHealth'
import { clockTime, latencyText } from '../lib/format'

interface ModuleCardProps {
  module: ModuleDefinition
  health: ModuleHealth
  onOpen: () => void
  onCopy: () => void
}

const STATUS_TEXT: Record<HealthStatus, string> = {
  checking: '检测中',
  online: '运行中',
  offline: '未启动',
}

function barHeight(sample: Sample): number {
  if (!sample.ok) return 20
  if (sample.ms === null) return 8
  return Math.min(20, 6 + (sample.ms / 1200) * 14)
}

export function ModuleCard({ module, health, onOpen, onCopy }: ModuleCardProps) {
  const Icon = module.icon
  const recent = health.samples.slice(-HISTORY_LIMIT)
  const placeholders = Math.max(0, HISTORY_LIMIT - recent.length)
  const bars: (Sample | null)[] = [...Array<Sample | null>(placeholders).fill(null), ...recent]
  const successCount = recent.filter((sample) => sample.ok).length

  return (
    <article className="module" style={{ '--module-accent': module.accent } as CSSProperties}>
      <header className="module-head">
        <span className="module-icon" aria-hidden="true">
          <Icon size={19} strokeWidth={1.8} />
        </span>
        <div className="module-title">
          <h3>{module.name}</h3>
          <p className="module-source">
            {module.source} · {module.endpoint}
          </p>
        </div>
        <span className="status-pill" data-status={health.status}>
          <i aria-hidden="true" />
          {STATUS_TEXT[health.status]}
        </span>
      </header>

      <p className="module-desc">{module.description}</p>

      <ul className="module-features">
        {module.features.map((feature) => (
          <li key={feature}>{feature}</li>
        ))}
      </ul>

      <div className="module-telemetry">
        <div className="telemetry-cell">
          <span>延迟</span>
          <b data-empty={health.latency === null}>{latencyText(health.latency)}</b>
        </div>
        <div className="telemetry-cell">
          <span>最近检测</span>
          <b>{clockTime(health.checkedAt)}</b>
        </div>
        <div
          className="telemetry-bars"
          role="img"
          aria-label={
            recent.length
              ? `最近 ${recent.length} 次检测，${successCount} 次成功`
              : '暂无检测样本'
          }
        >
          {bars.map((sample, index) => (
            <i
              key={sample ? sample.at : `slot-${index}`}
              data-ok={sample ? sample.ok : undefined}
              data-empty={sample === null || undefined}
              style={{ height: sample ? barHeight(sample) : 2 }}
            />
          ))}
        </div>
      </div>

      {health.status === 'offline' && (
        <p className="module-alert">
          健康检查未通过{health.error ? `（${health.error}）` : ''}，确认子系统已启动后重试。
        </p>
      )}

      <div className="module-actions">
        <button type="button" className="btn btn-solid" onClick={onOpen}>
          打开
          <ArrowUpRight size={15} />
        </button>
        <button type="button" className="btn btn-ghost" onClick={onCopy}>
          <Copy size={14} />
          复制地址
        </button>
      </div>
    </article>
  )
}
