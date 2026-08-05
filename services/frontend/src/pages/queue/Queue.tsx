import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { TicketPurchaseModal } from '../../components/modals/TicketPurchaseModal'
import type { Product } from '../../features/product/types'
import { leaveQueue } from '../../features/queue/queueSlice'
import { useCountdown } from '../../features/queue/useCountdown'
import type { QueueEntry, QueueStatus } from '../../features/queue/types'
import { ticketApi } from '../../features/ticket/api'

type QueueTileView = QueueEntry & Pick<Product, 'name' | 'image'>

const statusStyles: Record<QueueStatus, string> = {
  queued: 'bg-avito-blue-soft text-[#0075b8]',
  ticket: 'bg-[#f0f9e7] text-[#4a7f14]',
  soldout: 'bg-[#fff0f2] text-[#b82334]',
}

export function Queue() {
  const navigate = useNavigate()
  const dispatch = useAppDispatch()
  const queueItems = useAppSelector((state) => state.queue.queueItems)
  const productItems = useAppSelector((state) => state.products.productItems)
  const [ticketTile, setTicketTile] = useState<QueueTileView | null>(null)

  const queueTiles: QueueTileView[] = queueItems.map((entry) => {
    const product = productItems.find((p) => p.id === entry.productId)
    return {
      ...entry,
      name: product?.name ?? `Товар ${entry.productId}`,
      image: product?.image ?? '🛒',
    }
  })

  const activeCount = queueTiles.filter((t) => t.status === 'queued').length
  const ticketCount = queueTiles.filter((t) => t.status === 'ticket').length
  const doneCount = queueTiles.filter((t) => t.status === 'soldout').length

  return (
    <section>
      <div className="mb-[22px] flex flex-col items-start justify-between gap-5 sm:flex-row sm:items-end">
        <div>
          <h1 className="m-0 text-2xl tracking-tight sm:text-[30px]">
            Мои очереди
          </h1>
          <p className="mt-2 max-w-[620px] leading-normal text-avito-muted">
            Каждая плитка — отдельная очередь. Откройте товар, чтобы посмотреть
            позицию, право на покупку или продолжить оформление.
          </p>
        </div>
        <Link
          to="/catalog"
          className="inline-flex min-h-10 items-center rounded-xl bg-[#f1f1f1] px-4 py-2 font-extrabold text-avito-ink no-underline"
        >
          ← В каталог
        </Link>
      </div>

      <div className="mb-[22px] grid grid-cols-1 gap-3.5 sm:grid-cols-3">
        <SummaryTile label="Активных очередей" value={activeCount} />
        <SummaryTile label="Доступно к покупке" value={ticketCount} />
        <SummaryTile label="Завершено" value={doneCount} />
      </div>

      {queueTiles.length === 0 ? (
        <div className="rounded-2xl bg-white p-8 text-center text-avito-muted">
          Вы ещё не вставали в очередь.{' '}
          <Link to="/catalog" className="font-extrabold text-[#008ed8]">
            Перейти в каталог
          </Link>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-[18px] sm:grid-cols-2 lg:grid-cols-3">
          {queueTiles.map((tile) => (
            <QueueCard
              key={tile.id}
              tile={tile}
              onOpenTicket={() => setTicketTile(tile)}
              onLeaveQueue={() => dispatch(leaveQueue(tile.id))}
            />
          ))}
        </div>
      )}

      <TicketPurchaseModal
        open={ticketTile !== null}
        productTitle={ticketTile?.name ?? ''}
        productImage={ticketTile?.image ?? '🛒'}
        expiresAt={ticketTile?.expiresAt}
        onClose={() => setTicketTile(null)}
        onBuy={() => {
          if (!ticketTile) return
          const id = ticketTile.id
          setTicketTile(null)
          navigate(`/checkout?ticket=${id}`)
        }}
        onDecline={() => {
          if (!ticketTile) return
          const id = ticketTile.id
          void (async () => {
            try {
              await ticketApi.declineTicket(id)
            } catch (error) {
              console.error(error)
            } finally {
              dispatch(leaveQueue(id))
              setTicketTile(null)
            }
          })()
        }}
      />
    </section>
  )
}

function SummaryTile({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-2xl bg-white p-[18px]">
      <div className="text-sm text-avito-muted">{label}</div>
      <div className="mt-2 text-[28px] font-extrabold">{value}</div>
    </div>
  )
}

function QueueCard({
  tile,
  onOpenTicket,
  onLeaveQueue,
}: {
  tile: QueueTileView
  onOpenTicket: () => void
  onLeaveQueue: () => void
}) {
  const countdown = useCountdown(
    tile.status === 'ticket' ? tile.expiresAt : undefined,
  )

  return (
    <article className="flex min-h-[365px] flex-col overflow-hidden rounded-[20px] border border-transparent bg-white transition duration-150 hover:-translate-y-0.5 hover:border-[#c9c9c9] hover:shadow-card">
      <div className="grid h-[170px] place-items-center bg-linear-to-br from-[#dff5ff] to-[#e7dcff] text-[92px]">
        {tile.image}
      </div>

      <div className="flex flex-1 flex-col gap-4 p-5 sm:p-6">
        <div
          className={`self-start rounded-full px-2.5 py-1.5 text-xs font-extrabold ${statusStyles[tile.status]}`}
        >
          {statusLabel(tile, countdown)}
        </div>

        <div className="text-lg font-extrabold leading-snug text-avito-ink">
          {tile.name}
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
          {tile.status === 'ticket' && (
            <button
              type="button"
              className="min-h-10 w-full cursor-pointer rounded-[10px] bg-avito-blue px-3 py-2 text-[13px] font-extrabold text-white transition duration-150 hover:-translate-y-px hover:bg-avito-blue-hover"
              onClick={onOpenTicket}
            >
              Перейти к покупке
            </button>
          )}

          {tile.status === 'queued' && (
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

function statusLabel(tile: QueueTileView, countdown: string | null): string {
  if (tile.status === 'queued') {
    return `В очереди · место ${tile.position ?? '—'}`
  }
  if (tile.status === 'ticket') {
    return countdown
      ? `Право на покупку · ${countdown}`
      : 'Право на покупку'
  }
  return 'Товар закончился'
}

function metaRows(
  tile: QueueTileView,
  countdown: string | null,
): { label: string; value: string }[] {
  if (tile.status === 'queued') {
    return [
      {
        label: 'Перед вами',
        value: String(Math.max(0, (tile.position ?? 1) - 1)),
      },
      { label: 'Тикет', value: 'ожидается' },
    ]
  }
  if (tile.status === 'ticket') {
    return [
      { label: 'Право на покупку', value: '1 шт.' },
      { label: 'Осталось', value: countdown ?? '—' },
    ]
  }
  return [
    { label: 'Поступление', value: 'можно подписаться' },
    { label: 'Тикет', value: 'нет' },
  ]
}
