import { AlertCircle, CheckCircle2, Info, X } from 'lucide-react'
import type { ToastMessage } from '../types'

interface ToastProps {
  message: ToastMessage | null
  onClose: () => void
}

const ICONS = {
  success: CheckCircle2,
  error: AlertCircle,
  info: Info,
}

export default function Toast({ message, onClose }: ToastProps) {
  if (!message) return null
  const Icon = ICONS[message.kind]
  return (
    <div className={`toast toast-${message.kind}`} role="status">
      <Icon size={18} />
      <span>{message.text}</span>
      <button type="button" className="toast-close" onClick={onClose} aria-label="关闭提示">
        <X size={15} />
      </button>
    </div>
  )
}
