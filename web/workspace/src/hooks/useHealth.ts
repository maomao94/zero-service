import { useCallback, useEffect, useRef, useState } from 'react'
import { modules, POLL_INTERVAL_MS } from '../data/modules'
import { probe } from '../lib/health'

/** 每张卡片保留的最近检测样本数（用于响应条） */
export const HISTORY_LIMIT = 24
/** 状态板保留的聚合快照数（用于响应趋势线） */
const SNAPSHOT_LIMIT = 40

export type HealthStatus = 'checking' | 'online' | 'offline'

export interface Sample {
  at: number
  ok: boolean
  ms: number | null
}

export interface ModuleHealth {
  status: HealthStatus
  latency: number | null
  checkedAt: number | null
  error: string | null
  samples: Sample[]
}

export interface Snapshot {
  at: number
  online: number
  total: number
  latency: number | null
}

export interface ActivityEvent {
  id: string
  at: number
  /** 启动事件没有关联模块，moduleId 为空字符串 */
  moduleId: string
  kind: 'online' | 'offline' | 'boot'
  latency: number | null
}

function initialHealth(): Record<string, ModuleHealth> {
  return Object.fromEntries(
    modules.map((module) => [
      module.id,
      { status: 'checking', latency: null, checkedAt: null, error: null, samples: [] } satisfies ModuleHealth,
    ]),
  )
}

/**
 * 工作台运行状态：并发探测全部子系统，记录逐次样本、聚合快照与上下线事件。
 * 页面不可见时跳过定时探测，回到前台立即补检，避免后台标签页刷日志。
 */
export function useHealth() {
  const [health, setHealth] = useState<Record<string, ModuleHealth>>(initialHealth)
  const [events, setEvents] = useState<ActivityEvent[]>([])
  const [snapshots, setSnapshots] = useState<Snapshot[]>([])
  const [checking, setChecking] = useState(true)
  const [lastCheckAt, setLastCheckAt] = useState<number | null>(null)
  const [nextCheckAt, setNextCheckAt] = useState(() => Date.now() + POLL_INTERVAL_MS)

  const healthRef = useRef(health)
  const inFlight = useRef(false)
  const booted = useRef(false)

  const checkAll = useCallback(async () => {
    if (inFlight.current) return
    inFlight.current = true
    setChecking(true)

    const at = Date.now()
    const results = await Promise.all(
      modules.map(async (module) => ({ id: module.id, ...(await probe(module.healthUrl)) })),
    )

    const previous = healthRef.current
    const next: Record<string, ModuleHealth> = { ...previous }
    const transitions: ActivityEvent[] = []

    for (const result of results) {
      const status: HealthStatus = result.ok ? 'online' : 'offline'
      const before = previous[result.id]
      next[result.id] = {
        status,
        latency: result.ms,
        checkedAt: at,
        error: result.error ?? null,
        samples: [...before.samples, { at, ok: result.ok, ms: result.ms }].slice(-HISTORY_LIMIT),
      }
      // 首次检测（checking → online/offline）不算状态变化，避免开局刷屏
      if (before.status !== 'checking' && before.status !== status) {
        transitions.push({
          id: `${result.id}-${at}`,
          at,
          moduleId: result.id,
          kind: status,
          latency: result.ms,
        })
      }
    }

    const onlineResults = results.filter((result) => result.ok)
    const average = onlineResults.length
      ? Math.round(onlineResults.reduce((sum, result) => sum + (result.ms ?? 0), 0) / onlineResults.length)
      : null

    if (!booted.current) {
      booted.current = true
      transitions.push({
        id: `boot-${at}`,
        at,
        moduleId: '',
        kind: 'boot',
        latency: null,
      })
    }

    healthRef.current = next
    setHealth(next)
    setSnapshots((list) =>
      [...list, { at, online: onlineResults.length, total: results.length, latency: average }].slice(-SNAPSHOT_LIMIT),
    )
    if (transitions.length) {
      setEvents((list) => [...transitions, ...list].slice(0, 60))
    }
    setLastCheckAt(at)
    setNextCheckAt(Date.now() + POLL_INTERVAL_MS)
    setChecking(false)
    inFlight.current = false
  }, [])

  useEffect(() => {
    void checkAll()

    const timer = window.setInterval(() => {
      if (document.visibilityState === 'visible') void checkAll()
    }, POLL_INTERVAL_MS)

    const onVisibilityChange = () => {
      if (document.visibilityState === 'visible') void checkAll()
    }
    document.addEventListener('visibilitychange', onVisibilityChange)

    return () => {
      window.clearInterval(timer)
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
  }, [checkAll])

  const clearEvents = useCallback(() => setEvents([]), [])

  return { health, events, snapshots, checking, lastCheckAt, nextCheckAt, checkAll, clearEvents }
}
