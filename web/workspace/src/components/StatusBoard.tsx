import { clockTime } from '../lib/format'
import type { Snapshot } from '../hooks/useHealth'
import { Sparkline } from './Sparkline'

interface StatusBoardProps {
  online: number
  total: number
  average: number | null
  checking: boolean
  lastCheckAt: number | null
  nextCheckAt: number
  now: number
  snapshots: Snapshot[]
}

type Tone = 'ok' | 'warn' | 'bad' | 'checking'

const TONE_TEXT: Record<Tone, string> = {
  ok: '运行正常',
  warn: '部分不可用',
  bad: '全部不可用',
  checking: '检测中',
}

function resolveTone(online: number, total: number, checking: boolean): Tone {
  if (checking && online === 0) return 'checking'
  if (online === total) return 'ok'
  if (online === 0) return 'bad'
  return 'warn'
}

export function StatusBoard({
  online,
  total,
  average,
  checking,
  lastCheckAt,
  nextCheckAt,
  now,
  snapshots,
}: StatusBoardProps) {
  const tone = resolveTone(online, total, checking)
  const seconds = Math.max(0, Math.ceil((nextCheckAt - now) / 1000))
  const offline = total - online

  return (
    <section className="board" data-tone={tone} aria-label="系统状态">
      <div className="board-grid" aria-hidden="true" />

      <div className="board-main">
        <div className="board-state">
          <p className="board-label">系统状态</p>
          <p className="board-title">
            <span className="board-dot" aria-hidden="true" />
            {TONE_TEXT[tone]}
          </p>
          <p className="board-sub">
            {offline > 0
              ? `${offline} 个模块未就绪，可在下方单独重试`
              : `全部 ${total} 个模块可达，健康检查通过`}
            <span className="board-sub-sep" aria-hidden="true" />
            最近检测 {clockTime(lastCheckAt)}
          </p>
        </div>

        <dl className="board-stats">
          <div className="board-stat">
            <dt>在线模块</dt>
            <dd>
              {online}
              <span className="board-stat-total">/{total}</span>
            </dd>
          </div>
          <div className="board-stat">
            <dt>平均响应</dt>
            <dd>
              {average === null ? '—' : average}
              {average !== null && <span className="board-stat-unit">ms</span>}
            </dd>
          </div>
          <div className="board-stat">
            <dt>下次检测</dt>
            <dd>
              {checking ? '—' : seconds}
              {!checking && <span className="board-stat-unit">s</span>}
            </dd>
          </div>
        </dl>
      </div>

      <div className="board-trend">
        <span className="board-trend-label">响应趋势 · 最近 {snapshots.length} 次</span>
        <Sparkline snapshots={snapshots} />
      </div>
    </section>
  )
}
