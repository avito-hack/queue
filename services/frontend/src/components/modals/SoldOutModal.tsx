type SoldOutModalProps = {
  open: boolean
  productTitle: string
  notified?: boolean
  onClose: () => void
  onNotify: () => void
  onSimilar?: () => void
}

export function SoldOutModal({
  open,
  productTitle,
  notified = false,
  onClose,
  onNotify,
  onSimilar,
}: SoldOutModalProps) {
  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 grid place-items-end bg-black/50 p-0 backdrop-blur-[3px] sm:place-items-center sm:p-4"
      onClick={onClose}
      role="presentation"
    >
      <div
        className="max-h-[92dvh] w-full max-w-[480px] overflow-y-auto rounded-t-3xl bg-white p-5 shadow-[0_24px_60px_rgba(0,0,0,0.22)] sm:rounded-[22px] sm:p-6"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="sold-out-title"
      >
        <div className="mb-4 flex items-start justify-between gap-4">
          <h2
            id="sold-out-title"
            className="m-0 text-[25px] leading-tight tracking-tight"
          >
            Упс, товар закончился
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

        <p className="mb-5 leading-relaxed text-[#555]">
          «{productTitle}» больше нет в наличии. Очередь по этому товару
          завершена. Можно подписаться на уведомление о поступлении или
          посмотреть похожие товары ниже на странице.
        </p>

        <div className="grid gap-2.5">
          <button
            type="button"
            disabled={notified}
            className="min-h-12 w-full cursor-pointer rounded-xl bg-avito-blue px-[18px] py-3 font-extrabold text-white transition duration-150 hover:-translate-y-px hover:bg-avito-blue-hover disabled:cursor-default disabled:opacity-70 disabled:hover:translate-y-0"
            onClick={onNotify}
          >
            {notified ? 'Подписка оформлена' : 'Уведомить о поступлении'}
          </button>
          {onSimilar && (
            <button
              type="button"
              className="min-h-12 w-full cursor-pointer rounded-xl bg-[#f1f1f1] px-[18px] py-3 font-extrabold text-avito-ink"
              onClick={onSimilar}
            >
              Посмотреть похожие товары
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
