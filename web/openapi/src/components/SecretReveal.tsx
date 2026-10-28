import { KeyRound, ShieldAlert, X } from 'lucide-react'
import type { ToastKind } from '../types'
import CopyButton from './CopyButton'

interface SecretRevealProps {
  title: string
  appId: string
  secret?: string
  signKey?: string
  hint?: string
  onDismiss: () => void
  notify: (kind: ToastKind, text: string) => void
}

/** 突出展示敏感凭据；secret/signKey 可按场景只展示其中一个。 */
export default function SecretReveal({ title, appId, secret, signKey, hint, onDismiss, notify }: SecretRevealProps) {
  return (
    <section className="secret-reveal">
      <div className="secret-heading">
        <div className="secret-title">
          <span className="secret-icon"><KeyRound size={18} /></span>
          <div>
            <b>{title}</b>
            <small>{hint ?? '请立即保存到安全位置，关闭后可能无法再次查看。'}</small>
          </div>
        </div>
        <button type="button" className="icon-button subtle" onClick={onDismiss} aria-label="关闭">
          <X size={16} />
        </button>
      </div>
      <div className="secret-warning">
        <ShieldAlert size={15} />
        <span>仅限测试环境使用；请立即保存到安全位置，不要提交到代码仓库。</span>
      </div>
      <div className="secret-row">
        <label>App ID</label>
        <code>{appId}</code>
        <CopyButton text={appId} onResult={(ok) => notify(ok ? 'success' : 'error', ok ? 'App ID 已复制' : '复制失败')} />
      </div>
      {signKey && (
        <div className="secret-row">
          <label>Sign Key</label>
          <code className="secret-value">{signKey}</code>
          <CopyButton text={signKey} label="复制签名密钥" onResult={(ok) => notify(ok ? 'success' : 'error', ok ? '签名密钥已复制' : '复制失败')} />
        </div>
      )}
      {secret && (
        <div className="secret-row">
          <label>Secret</label>
          <code className="secret-value">{secret}</code>
          <CopyButton
            text={secret}
            label="复制密钥"
            onResult={(ok) => notify(ok ? 'success' : 'error', ok ? '密钥已复制' : '复制失败')}
          />
        </div>
      )}
    </section>
  )
}
