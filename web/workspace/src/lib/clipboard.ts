/**
 * 复制文本。内网常见 http 源站不属于安全上下文，navigator.clipboard 不可用，
 * 因此保留 execCommand 兜底；同时对 clipboard 写入加超时，避免权限弹窗挂起时
 * 用户点了“复制”却没有任何反馈。
 */
export async function copyText(text: string): Promise<boolean> {
  if (navigator.clipboard?.writeText) {
    try {
      const written = await Promise.race([
        navigator.clipboard.writeText(text).then(() => true),
        new Promise<boolean>((resolve) => {
          window.setTimeout(() => resolve(false), 900)
        }),
      ])
      if (written) return true
    } catch {
      // 继续走兜底方案
    }
  }

  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.top = '-1000px'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)

  try {
    textarea.select()
    return document.execCommand('copy')
  } catch {
    return false
  } finally {
    textarea.remove()
  }
}
