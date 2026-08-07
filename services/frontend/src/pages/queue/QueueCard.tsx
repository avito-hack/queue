import { Link } from 'react-router-dom'
import { useCountdown } from '../../features/queue/useCountdown'
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
    <article className="flex min-h-[365px] flex-col overflow-hidden rounded-[20px] border border-transparent bg-white transition duration-150 hover:-translate-y-0.5 hover:border-[#c9c9c9] hover:shadow-card">
      <div className="grid h-[170px] place-items-center bg-linear-to-br from-[#dff5ff] to-[#e7dcff] text-[92px]">
        {tile.image}
      </div>

      <div className="flex flex-1 flex-col gap-4 p-5 sm:p-6">
        <div
          className={`self-start rounded-full px-2.5 py-1.5 text-xs font-extrabold ${statusStyles[tile.kind]}`}
        >
          {statusLabel(tile, countdown)}
        </div>

        <div className="text-lg font-extrabold leading-snug text-avito-ink">
          {tile.title}
        </div>

        <div className="grid gap-2.5 text-sm text-[#555]">
          {metaRows(tile, countdown).map((row) => (
            <div key={row.label} className="flex justify-between gap-3">
              <span>{row.label}</span>
              <strong className="font-mono text-avito-ink">{row.value}</strong>
            </div>
          ))}
        </div>

        <div className="mt-auto flex flex-col gap-3 pt-2">
          {tile.kind === 'ticket' && (
            <button
              type="button"
              className="min-h-10 w-full cursor-pointer rounded-[10px] bg-avito-blue px-3 py-2 text-[13px] font-extrabold text-white transition duration-150 hover:-translate-y-px hover:bg-avito-blue-hover"
              onClick={onOpenTicket}
            >
              Перейти к покупке
            </button>
          )}

          {tile.kind === 'queued' && (
            <button
              type="button"
              className="min-h-10 w-full cursor-pointer rounded-[10px] bg-[#fff0f2] px-3 py-2 text-[13px] font-extrabold text-avito-red"
              onClick={onLeaveQueue}
            >
              Выйти из очереди
            </button>
          )}

          <Link
            to={`/product/${tile.productId}`}
            className="text-center text-sm font-extrabold text-[#008ed8] no-underline"
          >
            Перейти на страницу товара →
          </Link>
        </div>
      </div>
    </article>
  )
}
