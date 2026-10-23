import { AlertTriangle, LoaderCircle } from 'lucide-react'

interface ConfirmDialogProps {
  open: boolean
  title: string
  message: string
  confirmLabel?: string
  danger?: boolean
  loading?: boolean
  onCancel: () => void
  onConfirm: () => void
}

export default function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = '确认',
  danger = false,
  loading = false,
  onCancel,
  onConfirm,
}: ConfirmDialogProps) {
  if (!open) return null

  return (
    <div className="modal-overlay" role="dialog" aria-modal="true" aria-label={title}>
      <div className="modal-card">
        <div className={`modal-icon ${danger ? 'danger' : ''}`}>
          <AlertTriangle size={22} />
        </div>
        <h3>{title}</h3>
        <p>{message}</p>
        <div className="modal-actions">
          <button type="button" className="button secondary" onClick={onCancel} disabled={loading}>
            取消
          </button>
          <button
            type="button"
            className={`button ${danger ? 'danger' : 'primary'}`}
            onClick={onConfirm}
            disabled={loading}
          >
            {loading && <LoaderCircle size={14} className="spin" />}
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
