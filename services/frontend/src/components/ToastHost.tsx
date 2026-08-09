import { useEffect, useState } from 'react'
import { TOAST_EVENT, type ToastDetail } from '../shared/toast'

/**
 * Глобальный toast для ошибок API (и др. коротких сообщений).
 * Не модалка: не блокирует экран, сам скрывается.
 */
export function ToastHost() {
  const [toast, setToast] = useState<ToastDetail | null>(null)

  useEffect(() => {
    let hideTimer = 0
    const onToast = (event: Event) => {
      const detail = (event as CustomEvent<ToastDetail>).detail
      if (!detail?.message) return
      setToast(detail)
      window.clearTimeout(hideTimer)
      hideTimer = window.setTimeout(() => setToast(null), 5000)
    }

    window.addEventListener(TOAST_EVENT, onToast)
    return () => {
      window.removeEventListener(TOAST_EVENT, onToast)
      window.clearTimeout(hideTimer)
    }
  }, [])

  if (!toast) return null

  const isError = toast.variant === 'error'

  return (
    <div
      role="alert"
      className="pointer-events-none fixed inset-x-0 bottom-[calc(1.25rem+env(safe-area-inset-bottom))] z-[100] flex justify-center px-4"
    >
      <div
        className={
          isError
            ? 'pointer-events-auto max-w-md rounded-xl border border-[#f0b4bc] bg-[#fff0f2] px-4 py-3 text-sm font-bold text-[#b82334] shadow-card'
            : 'pointer-events-auto max-w-md rounded-xl border border-[#c9c9c9] bg-white px-4 py-3 text-sm font-bold text-avito-ink shadow-card'
        }
      >
        {toast.message}
        <button
          type="button"
          className="ml-3 cursor-pointer align-middle text-lg leading-none opacity-60 hover:opacity-100"
          aria-label="Закрыть"
          onClick={() => setToast(null)}
        >
          ×
        </button>
      </div>
    </div>
  )
}
