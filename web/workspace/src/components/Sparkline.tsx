import type { Snapshot } from '../hooks/useHealth'

interface SparklineProps {
  snapshots: Snapshot[]
}

/** 聚合响应趋势：仅展示有数据的检测点，缺失点按 0 处理，保持时间轴等距 */
export function Sparkline({ snapshots }: SparklineProps) {
  const width = 640
  const height = 96
  const values = snapshots.map((snapshot) => snapshot.latency)
  const series = values.length >= 2 ? values : [values[0] ?? 0, values[0] ?? 0]
  const max = Math.max(40, ...series.map((value) => value ?? 0)) * 1.2

  const points = series.map((value, index) => {
    const x = (index / (series.length - 1)) * width
    const y = height - ((value ?? 0) / max) * (height - 12) - 6
    return `${x.toFixed(1)} ${y.toFixed(1)}`
  })
  const line = `M${points.join(' L')}`
  const area = `${line} L${width} ${height} L0 ${height} Z`

  return (
    <svg className="board-spark" viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="none" aria-hidden="true">
      <defs>
        <linearGradient id="board-spark-fill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="currentColor" stopOpacity="0.32" />
          <stop offset="100%" stopColor="currentColor" stopOpacity="0" />
        </linearGradient>
      </defs>
      <path d={area} fill="url(#board-spark-fill)" />
      <path d={line} fill="none" stroke="currentColor" strokeWidth="1.5" vectorEffect="non-scaling-stroke" />
    </svg>
  )
}
