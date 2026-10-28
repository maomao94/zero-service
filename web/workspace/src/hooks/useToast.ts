import { useCallback, useEffect, useRef, useState } from 'react'

export interface ToastState {
  id: number
  text: string
}

/** 单条轻提示：复制成功等即时反馈，自动消失 */
export function useToast(duration = 2400) {
  const [toast, setToast] = useState<ToastState | null>(null)
  const timer = useRef<number | undefined>(undefined)

  const show = useCallback(
    (text: string) => {
      setToast({ id: Date.now(), text })
      window.clearTimeout(timer.current)
      timer.current = window.setTimeout(() => setToast(null), duration)
    },
    [duration],
  )

  useEffect(() => () => window.clearTimeout(timer.current), [])

  return { toast, show }
}
