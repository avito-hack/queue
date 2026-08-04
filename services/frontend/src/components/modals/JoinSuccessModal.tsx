type JoinSuccessModalProps = {
  open: boolean
  position?: number
  productTitle: string
  onClose: () => void
  onGoToQueues: () => void
}

export function JoinSuccessModal({
  open,
  position,
  productTitle,
  onClose,
  onGoToQueues,
}: JoinSuccessModalProps) {
  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 grid place-items-center bg-black/50 p-4 backdrop-blur-[3px]"
      onClick={onClose}
      role="presentation"
    >
      <div
        className="w-full max-w-[480px] rounded-[22px] bg-white p-6 shadow-[0_24px_60px_rgba(0,0,0,0.22)]"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="join-success-title"
      >
        <div className="mb-4 flex items-start justify-between gap-4">
          <h2
            id="join-success-title"
            className="m-0 text-[25px] leading-tight tracking-tight"
          >
            Вы в очереди
          </h2>
          <button
            type="button"
            className="grid size-9 shrink-0 cursor-pointer place-items-center rounded-full bg-[#f2f2f2] text-xl leading-none"
            onClick={onClose}
            aria-label="Закрыть"
          >
            ×
          </button>
        </div>

        <p className="mb-5 text-[#555] leading-relaxed">
          Вы успешно встали в очередь на «{productTitle}». Мы сообщим, когда
          подойдёт ваша очередь и появится право на покупку.
        </p>

        <div className="mb-5 rounded-[14px] bg-avito-blue-soft p-4">
          <div className="text-sm text-[#0078bd]">Ваше место</div>
          <div className="mt-1 text-[28px] font-extrabold text-avito-ink">
            {position ?? '—'}
          </div>
        </div>

        <div className="grid gap-2.5">
          <button
            type="button"
            className="min-h-12 w-full cursor-pointer rounded-xl bg-avito-blue px-[18px] py-3 font-extrabold text-white transition duration-150 hover:-translate-y-px hover:bg-avito-blue-hover"
            onClick={onGoToQueues}
          >
            Перейти к моим очередям
          </button>
          <button
            type="button"
            className="min-h-12 w-full cursor-pointer rounded-xl bg-avito-blue-soft px-[18px] py-3 font-extrabold text-[#007fc8]"
            onClick={onClose}
          >
            Остаться на странице товара
          </button>
        </div>
      </div>
    </div>
  )
}
