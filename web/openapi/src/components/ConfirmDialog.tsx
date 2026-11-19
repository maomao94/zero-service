import { AlertTriangle, LoaderCircle } from 'lucide-react'
import { useEffect, useRef, type ReactNode } from 'react'

interface ConfirmDialogProps {
  open: boolean
  title: string
  message: string
  children?: ReactNode
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
  children,
  confirmLabel = '确认',
  danger = false,
  loading = false,
  onCancel,
  onConfirm,
}: ConfirmDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    if (!open) return
    const dialog = dialogRef.current
    dialog?.showModal()
    return () => dialog?.close()
  }, [open])

  if (!open) return null

  return (
    <dialog ref={dialogRef} className="console-dialog" aria-label={title} onCancel={(event) => {
      event.preventDefault()
      if (!loading) onCancel()
    }}>
      <div className="modal-card">
        <div className={`modal-icon ${danger ? 'danger' : ''}`}>
          <AlertTriangle size={22} />
        </div>
        <h3>{title}</h3>
        <p>{message}</p>
        {children}
        <div className="modal-actions">
          <button type="button" className="button secondary" onClick={onCancel} disabled={loading} autoFocus>
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
    </dialog>
  )
}
