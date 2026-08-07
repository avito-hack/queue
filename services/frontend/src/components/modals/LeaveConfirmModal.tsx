type LeaveConfirmModalProps = {
  open: boolean
  productTitle: string
  leaving?: boolean
  onClose: () => void
  onConfirm: () => void
}

export function LeaveConfirmModal({
  open,
  productTitle,
  leaving = false,
  onClose,
  onConfirm,
}: LeaveConfirmModalProps) {
  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 grid place-items-center bg-black/50 p-4 backdrop-blur-[3px]"
      onClick={leaving ? undefined : onClose}
      role="presentation"
    >
      <div
        className="w-full max-w-[480px] rounded-[22px] bg-white p-6 shadow-[0_24px_60px_rgba(0,0,0,0.22)]"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="leave-confirm-title"
      >
        <div className="mb-4 flex items-start justify-between gap-4">
          <h2
            id="leave-confirm-title"
            className="m-0 text-[25px] leading-tight tracking-tight"
          >
            Выйти из очереди?
          </h2>
          <button
            type="button"
            className="grid size-9 shrink-0 cursor-pointer place-items-center rounded-full bg-[#f2f2f2] text-xl leading-none disabled:cursor-default disabled:opacity-50"
            onClick={onClose}
            disabled={leaving}
            aria-label="Закрыть"
          >
            ×
          </button>
        </div>

        <p className="mb-5 leading-relaxed text-[#555]">
          Вы покинете очередь на «{productTitle}». Вернуться можно только в
          конец очереди.
        </p>

        <div className="grid gap-2.5">
          <button
            type="button"
            disabled={leaving}
            className="min-h-12 w-full cursor-pointer rounded-xl bg-[#fff0f2] px-[18px] py-3 font-extrabold text-avito-red disabled:cursor-default disabled:opacity-50"
            onClick={onConfirm}
          >
            {leaving ? 'Выходим…' : 'Да, выйти из очереди'}
          </button>
          <button
            type="button"
            disabled={leaving}
            className="min-h-12 w-full cursor-pointer rounded-xl bg-[#f1f1f1] px-[18px] py-3 font-extrabold text-avito-ink disabled:cursor-default disabled:opacity-50"
            onClick={onClose}
          >
            Остаться в очереди
          </button>
        </div>
      </div>
    </div>
  )
}
