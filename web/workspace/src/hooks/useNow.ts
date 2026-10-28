import { useEffect, useState } from 'react'

/** 轻量时钟：驱动倒计时与顶栏时间，避免整页每秒重渲染 */
export function useNow(interval = 1000) {
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), interval)
    return () => window.clearInterval(timer)
  }, [interval])

  return now
}
