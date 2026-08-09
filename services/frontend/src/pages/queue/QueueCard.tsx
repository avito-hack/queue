import { Link } from 'react-router-dom'
import { useCountdown } from '../../features/queue/useCountdown'
import { ticketAllows } from '../../features/ticket/types'
import { metaRows, statusLabel, statusStyles } from './queueTileLib'
import type { QueueTileView } from './types'

type QueueCardProps = {
  tile: QueueTileView
  onOpenTicket: () => void
  onLeaveQueue: () => void
}

export function QueueCard({
  tile,
  onOpenTicket,
  onLeaveQueue,
}: QueueCardProps) {
  const countdown = useCountdown(
    tile.kind === 'ticket' ? tile.expiresAt : undefined,
  )

  return (
    <article className="flex flex-col overflow-hidden rounded-2xl bg-white transition duration-150 hover:shadow-card">
      <div className="grid aspect-[16/10] place-items-center bg-[#f5f5f5] text-[72px] sm:text-[84px]">
        <span className="select-none" aria-hidden="true">
          {tile.image}
        </span>
      </div>

      <div className="flex flex-1 flex-col gap-3.5 p-4 sm:gap-4 sm:p-5">
        <div
          className={`self-start rounded-lg px-2.5 py-1.5 text-[12px] font-extrabold ${statusStyles[tile.kind]}`}
        >
          {statusLabel(tile, countdown)}
        </div>

        <div className="text-[17px] font-extrabold leading-snug text-avito-ink sm:text-lg">
          {tile.title}
        </div>

        <div className="grid gap-2 text-[13px] text-[#555] sm:text-sm">
          {metaRows(tile, countdown).map((row) => (
            <div key={row.label} className="flex justify-between gap-3">
              <span>{row.label}</span>
              <strong className="font-semibold tabular-nums text-avito-ink">
                {row.value}
              </strong>
            </div>
          ))}
        </div>

        <div className="mt-auto flex flex-col gap-2.5 pt-1">
          {tile.kind === 'ticket' &&
            (ticketAllows(tile, 'activate') ||
              ticketAllows(tile, 'checkout') ||
              ticketAllows(tile, 'decline')) && (
            <button
              type="button"
              className="min-h-11 w-full cursor-pointer rounded-2xl bg-avito-blue px-3 py-2.5 text-[14px] font-extrabold text-white transition duration-150 hover:bg-avito-blue-hover"
              onClick={onOpenTicket}
            >
              Перейти к покупке
            </button>
          )}

          {tile.kind === 'queued' && (
            <button
              type="button"
              className="min-h-11 w-full cursor-pointer rounded-2xl bg-[#fff0f2] px-3 py-2.5 text-[14px] font-extrabold text-avito-red"
              onClick={onLeaveQueue}
            >
              Выйти из очереди
            </button>
          )}

          <Link
            to={`/product/${tile.productId}`}
            className="py-1 text-center text-[13px] font-extrabold text-avito-blue no-underline sm:text-sm"
          >
            Страница товара
          </Link>
        </div>
      </div>
    </article>
  )
}
