import { useEffect } from 'react'
import type { Toast } from '../../types/ui'

export function ToastMessage({ toast, onClose }: { toast: Toast | null; onClose: () => void }) {
  useEffect(() => {
    if (!toast) return undefined
    const timer = window.setTimeout(onClose, 3600)
    return () => window.clearTimeout(timer)
  }, [onClose, toast])

  if (!toast) return null
  return (
    <div className={`toast ${toast.type}`} role="status">
      <span>{toast.type === 'success' ? 'OK' : '!'}</span>
      <p>{toast.text}</p>
      <button type="button" onClick={onClose} aria-label="Fechar notificação">x</button>
    </div>
  )
}
