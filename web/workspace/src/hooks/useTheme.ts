import { useCallback, useEffect, useState } from 'react'

export type Theme = 'light' | 'dark'

const STORAGE_KEY = 'zero-workspace-theme'

/** 首屏主题由 index.html 的内联脚本写入，这里直接读取，避免首帧闪烁 */
function readTheme(): Theme {
  return document.documentElement.dataset.theme === 'dark' ? 'dark' : 'light'
}

export function useTheme() {
  const [theme, setTheme] = useState<Theme>(readTheme)

  // 只应用主题，不主动写 localStorage：未手动切换前继续跟随系统偏好
  useEffect(() => {
    document.documentElement.dataset.theme = theme
  }, [theme])

  const toggleTheme = useCallback(() => {
    setTheme((current) => {
      const next = current === 'dark' ? 'light' : 'dark'
      try {
        localStorage.setItem(STORAGE_KEY, next)
      } catch {
        // 隐私模式下写入失败不影响当前主题
      }
      return next
    })
  }, [])

  return { theme, toggleTheme }
}
