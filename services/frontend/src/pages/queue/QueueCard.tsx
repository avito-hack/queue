import { Link } from 'react-router-dom'
import { useCountdown } from '../../features/queue/useCountdown'
import { ticketAllows } from '../../features/ticket/types'
import {
  isRedeemedTile,
  metaRows,
  statusLabel,
  tileStatusClass,
} from './queueTileLib'
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
  const redeemed = isRedeemedTile(tile)
  const countdown = useCountdown(
    tile.kind === 'ticket' && !redeemed ? tile.expiresAt : undefined,
  )

  return (
    <article
      className={[
        'flex flex-col overflow-hidden rounded-2xl bg-white transition duration-150 hover:shadow-card',
        redeemed
          ? 'ring-2 ring-[#ff9f1a]/80 shadow-[0_8px_24px_rgba(178,92,0,0.12)]'
          : '',
      ].join(' ')}
    >
      <div
        className={[
          'relative grid aspect-[16/10] place-items-center text-[72px] sm:text-[84px]',
          redeemed
            ? 'bg-[linear-gradient(160deg,#fff8ee_0%,#ffe7c2_100%)]'
            : 'bg-[#f5f5f5]',
        ].join(' ')}
      >
        {redeemed && (
          <div className="absolute top-3 left-3 rounded-lg bg-[#b25c00] px-2.5 py-1 text-[11px] font-extrabold tracking-wide text-white uppercase">
            К оплате
          </div>
        )}
        <span className="select-none" aria-hidden="true">
          {tile.image}
        </span>
      </div>

      <div
        className={[
          'flex flex-1 flex-col gap-3.5 p-4 sm:gap-4 sm:p-5',
          redeemed ? 'bg-[#fffaf3]' : '',
        ].join(' ')}
      >
        <div
          className={`self-start rounded-lg px-2.5 py-1.5 text-[12px] font-extrabold ${tileStatusClass(tile)}`}
        >
          {statusLabel(tile, countdown)}
        </div>

        <div className="text-[17px] font-extrabold leading-snug text-avito-ink sm:text-lg">
          {tile.title}
        </div>

        {redeemed && (
          <p className="m-0 rounded-xl bg-[#fff4e5] px-3 py-2 text-[13px] leading-snug font-semibold text-[#8a4700]">
            Тикет уже активирован. Осталось оплатить заказ — место в очереди
            больше не нужно.
          </p>
        )}

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
              className={[
                'min-h-11 w-full cursor-pointer rounded-2xl px-3 py-2.5 text-[14px] font-extrabold text-white transition duration-150',
                redeemed
                  ? 'bg-[#e8890c] hover:bg-[#d47b08]'
                  : 'bg-avito-blue hover:bg-avito-blue-hover',
              ].join(' ')}
              onClick={onOpenTicket}
            >
              {redeemed ? 'Перейти к оплате' : 'Перейти к покупке'}
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
