import { Check } from 'lucide-react'
import type { ToastState } from '../hooks/useToast'

/** 常驻 live region：内容变化时读屏才会可靠播报 */
export function Toast({ toast }: { toast: ToastState | null }) {
  return (
    <div className="toast-region" role="status" aria-live="polite">
      {toast && (
        <div className="toast" key={toast.id}>
          <Check size={15} aria-hidden="true" />
          {toast.text}
        </div>
      )}
    </div>
  )
}
