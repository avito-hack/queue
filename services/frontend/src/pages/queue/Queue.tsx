import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { LeaveConfirmModal } from '../../components/modals/LeaveConfirmModal'
import { TicketPurchaseModal } from '../../components/modals/TicketPurchaseModal'
import { queueApi } from '../../features/queue/api'
import { isTicketExpired } from '../../features/queue/lib'
import { leaveQueue } from '../../features/queue/queueSlice'
import { ticketApi } from '../../features/ticket/api'
import { removeTicket } from '../../features/ticket/ticketSlice'
import { QueueCard } from './QueueCard'
import { QueueEmpty } from './QueueEmpty'
import { SummaryTile } from './SummaryTile'
import type { QueueTileView, TileKind } from './types'

export function Queue() {
  const navigate = useNavigate()
  const dispatch = useAppDispatch()
  const queueItems = useAppSelector((state) => state.queue.queueItems)
  const ticketItems = useAppSelector((state) => state.tickets.ticketItems)
  const productItems = useAppSelector((state) => state.products.productItems)
  const [ticketTile, setTicketTile] = useState<QueueTileView | null>(null)
  const [leaveTile, setLeaveTile] = useState<QueueTileView | null>(null)
  const [buying, setBuying] = useState(false)
  const [leaving, setLeaving] = useState(false)

  const closeTicketModal = () => {
    if (buying) return
    setTicketTile(null)
  }

  const handleActivateAndBuy = () => {
    if (!ticketTile || buying) return
    const id = ticketTile.id
    if (isTicketExpired(ticketTile.expiresAt)) {
      dispatch(removeTicket(id))
      setTicketTile(null)
      return
    }

    void (async () => {
      setBuying(true)
      try {
        await ticketApi.activateTicket(id)
        setTicketTile(null)
        navigate(`/checkout?ticket=${id}`)
      } catch (error) {
        console.error(error)
        // Пока бэк может быть недоступен — как у join: не блокируем демо-checkout
        setTicketTile(null)
        navigate(`/checkout?ticket=${id}`)
      } finally {
        setBuying(false)
      }
    })()
  }

  const handleConfirmLeave = () => {
    if (!leaveTile || leaving) return
    const entryId = leaveTile.id
    const productId = leaveTile.productId

    void (async () => {
      setLeaving(true)
      try {
        await queueApi.leaveQueue(productId)
      } catch (error) {
        console.error(error)
        // бэк может быть недоступен — убираем из store для демо
      } finally {
        dispatch(leaveQueue(entryId))
        setLeaveTile(null)
        setLeaving(false)
      }
    })()
  }

  const withProduct = (entry: {
    id: string
    productId: string
    kind: TileKind
    position?: number
    expiresAt?: string
  }): QueueTileView => {
    const product = productItems.find((p) => p.id === entry.productId)
    return {
      ...entry,
      name: product?.name ?? `Товар ${entry.productId}`,
      image: product?.image ?? '🛒',
    }
  }

  const queueTiles: QueueTileView[] = [
    ...queueItems.map((entry) =>
      withProduct({
        id: entry.id,
        productId: entry.productId,
        kind: entry.status,
        position: entry.position,
      }),
    ),
    ...ticketItems.map((ticket) =>
      withProduct({
        id: ticket.id,
        productId: ticket.productId,
        kind: 'ticket',
        expiresAt: ticket.expiresAt,
      }),
    ),
  ]

  const activeCount = queueTiles.filter((t) => t.kind === 'queued').length
  const ticketCount = queueTiles.filter((t) => t.kind === 'ticket').length
  const doneCount = queueTiles.filter((t) => t.kind === 'soldout').length

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
        <QueueEmpty />
      ) : (
        <div className="grid grid-cols-1 gap-[18px] sm:grid-cols-2 lg:grid-cols-3">
          {queueTiles.map((tile) => (
            <QueueCard
              key={`${tile.kind}-${tile.id}`}
              tile={tile}
              onOpenTicket={() => setTicketTile(tile)}
              onLeaveQueue={() => setLeaveTile(tile)}
            />
          ))}
        </div>
      )}

      <TicketPurchaseModal
        open={ticketTile !== null}
        productTitle={ticketTile?.name ?? ''}
        productImage={ticketTile?.image ?? '🛒'}
        expiresAt={ticketTile?.expiresAt}
        buying={buying}
        onClose={closeTicketModal}
        onBuy={handleActivateAndBuy}
        onDecline={() => {
          if (!ticketTile || buying) return
          const id = ticketTile.id
          void (async () => {
            try {
              await ticketApi.declineTicket(id)
            } catch (error) {
              console.error(error)
            } finally {
              dispatch(removeTicket(id))
              setTicketTile(null)
            }
          })()
        }}
      />

      <LeaveConfirmModal
        open={leaveTile !== null}
        productTitle={leaveTile?.name ?? ''}
        leaving={leaving}
        onClose={() => {
          if (leaving) return
          setLeaveTile(null)
        }}
        onConfirm={handleConfirmLeave}
      />
    </section>
  )
}
