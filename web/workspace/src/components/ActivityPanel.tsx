import { Activity, Trash2 } from 'lucide-react'
import { modules } from '../data/modules'
import type { ActivityEvent } from '../hooks/useHealth'
import { clockTime, latencyText } from '../lib/format'

interface ActivityPanelProps {
  events: ActivityEvent[]
  onClear: () => void
}

const moduleName = new Map(modules.map((module) => [module.id, module.name]))

export function ActivityPanel({ events, onClear }: ActivityPanelProps) {
  return (
    <section className="panel" aria-labelledby="activity-title">
      <header className="panel-head">
        <div>
          <h2 id="activity-title">检测记录</h2>
          <p>本次会话内的启动与上下线变化</p>
        </div>
        {events.length > 0 && (
          <button type="button" className="link-btn" onClick={onClear}>
            <Trash2 size={13} />
            清空
          </button>
        )}
      </header>

      {events.length === 0 ? (
        <div className="panel-empty">
          <Activity size={17} />
          <div>
            <b>暂无状态变化</b>
            <span>模块上线或断开时会记录在这里</span>
          </div>
        </div>
      ) : (
        <ol className="activity-list">
          {events.map((event) => (
            <li key={event.id} className="activity-row">
              <time className="activity-time" dateTime={new Date(event.at).toISOString()}>
                {clockTime(event.at)}
              </time>
              <span className="activity-dot" data-kind={event.kind} aria-hidden="true" />
              <span className="activity-text">
                {event.kind === 'boot' ? (
                  <>
                    <b>工作台</b>已就绪，开始状态检测
                  </>
                ) : (
                  <>
                    <b>{moduleName.get(event.moduleId) ?? event.moduleId}</b>
                    {event.kind === 'online' ? '恢复在线' : '连接中断'}
                  </>
                )}
              </span>
              {event.kind !== 'boot' && (
                <span className="activity-latency">{latencyText(event.latency)}</span>
              )}
            </li>
          ))}
        </ol>
      )}
    </section>
  )
}
