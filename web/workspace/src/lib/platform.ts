/** 平台判定只影响快捷键提示的文案，不改变功能 */
const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform)

/** 搜索面板快捷键文案 */
export const searchShortcut = isMac ? '⌘K' : 'Ctrl K'
