import { useCountdown } from '../../features/queue/useCountdown'

type TicketPurchaseModalProps = {
  open: boolean
  productTitle: string
  productImage: string
  expiresAt?: string
  buying?: boolean
  buyError?: string | null
  canActivate?: boolean
  canDecline?: boolean
  onClose: () => void
  onBuy: () => void
  onDecline: () => void
}

export function TicketPurchaseModal({
  open,
  productTitle,
  productImage,
  expiresAt,
  buying = false,
  buyError = null,
  canActivate = true,
  canDecline = true,
  onClose,
  onBuy,
  onDecline,
}: TicketPurchaseModalProps) {
  const countdown = useCountdown(expiresAt)
  const expired = countdown === '00:00'

  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-50 grid place-items-end bg-black/50 p-0 backdrop-blur-[3px] sm:place-items-center sm:p-4"
      onClick={buying ? undefined : onClose}
      role="presentation"
    >
      <div
        className="max-h-[92dvh] w-full max-w-[480px] overflow-y-auto rounded-t-3xl bg-white p-5 shadow-[0_24px_60px_rgba(0,0,0,0.22)] sm:rounded-[22px] sm:p-6"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="ticket-purchase-title"
      >
        <div className="mb-4 flex items-start justify-between gap-4">
          <div />
          <button
            type="button"
            className="grid size-9 shrink-0 cursor-pointer place-items-center rounded-full bg-[#f2f2f2] text-xl leading-none disabled:cursor-default disabled:opacity-50"
            onClick={onClose}
            disabled={buying}
            aria-label="Закрыть"
          >
            ×
          </button>
        </div>

        <div className="mb-3 grid place-items-center text-7xl" aria-hidden="true">
          {productImage}
        </div>

        <h2
          id="ticket-purchase-title"
          className="m-0 text-center text-[25px] leading-tight tracking-tight"
        >
          Товар доступен для вас
        </h2>
        <p className="mt-3 mb-5 text-center leading-relaxed text-[#555]">
          Очередь подошла для «{productTitle}». У вас есть ограниченное время,
          чтобы оформить заказ.
        </p>

        <div className="mb-5 rounded-[14px] bg-[#f0f9e7] p-4">
          <div className="flex justify-between gap-3 text-sm text-[#477b12]">
            <span>Право на покупку</span>
            <strong>1 шт.</strong>
          </div>
          <div className="mt-2 flex justify-between gap-3 text-sm text-[#477b12]">
            <span>Осталось времени</span>
            <strong className="font-mono text-[18px] text-avito-ink">
              {countdown ?? '—'}
            </strong>
          </div>
        </div>

        {expired && (
          <div className="mb-4 rounded-[14px] bg-[#fff0f2] px-3 py-2 text-sm font-bold text-[#b82334]">
            Время вышло. Право на покупку больше не действует.
          </div>
        )}

        {buyError && (
          <div className="mb-4 rounded-[14px] bg-[#fff0f2] px-3 py-2 text-sm font-bold text-[#b82334]">
            {buyError}
          </div>
        )}

        <div className="grid gap-2.5">
          {canActivate && (
            <button
              type="button"
              disabled={expired || buying}
              className="min-h-12 w-full cursor-pointer rounded-xl bg-avito-blue px-[18px] py-3 font-extrabold text-white transition duration-150 hover:-translate-y-px hover:bg-avito-blue-hover disabled:cursor-default disabled:opacity-50 disabled:hover:translate-y-0"
              onClick={onBuy}
            >
              {buying ? 'Активация…' : 'Перейти к покупке'}
            </button>
          )}
          {canDecline && (
            <button
              type="button"
              disabled={buying}
              className="min-h-12 w-full cursor-pointer rounded-xl bg-[#fff0f2] px-[18px] py-3 font-extrabold text-avito-red disabled:cursor-default disabled:opacity-50"
              onClick={onDecline}
            >
              Отказаться от покупки
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
