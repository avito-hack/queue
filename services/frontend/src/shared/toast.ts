export type ToastVariant = 'error' | 'info'

export type ToastDetail = {
  message: string
  variant: ToastVariant
}

export const TOAST_EVENT = 'avito-toast'

export function showToast(
  message: string,
  variant: ToastVariant = 'error',
): void {
  if (typeof window === 'undefined') return
  window.dispatchEvent(
    new CustomEvent<ToastDetail>(TOAST_EVENT, {
      detail: { message, variant },
    }),
  )
}
