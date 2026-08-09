import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { LeaveConfirmModal } from '../../components/modals/LeaveConfirmModal'
import { TicketPurchaseModal } from '../../components/modals/TicketPurchaseModal'
import { queueApi } from '../../features/queue/api'
import { isTicketExpired } from '../../features/queue/lib'
import { leaveQueue } from '../../features/queue/queueSlice'
import { ticketApi } from '../../features/ticket/api'
import { resolveCheckoutNavigation } from '../../features/ticket/checkoutNavigation'
import { removeTicket, upsertTicket } from '../../features/ticket/ticketSlice'
import { ticketAllows } from '../../features/ticket/types'
import { reportApiError } from '../../shared/api/errors'
import { showToast } from '../../shared/toast'
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

  const goToCheckout = (ticketId: string, productId: string, checkoutUrl?: string) => {
    const target = resolveCheckoutNavigation(ticketId, checkoutUrl)
    if (target.kind === 'external') {
      window.location.assign(target.url)
      return
    }
    navigate(target.path, { state: { productId } })
  }

  const handleActivateAndBuy = () => {
    if (!ticketTile || buying) return
    if (!ticketAllows(ticketTile, 'activate') && !ticketAllows(ticketTile, 'checkout')) {
      return
    }
    const id = ticketTile.id
    const productId = ticketTile.productId

    if (
      ticketTile.ticketStatus === 'redeemed' ||
      (ticketTile.checkoutUrl && !ticketAllows(ticketTile, 'activate'))
    ) {
      setTicketTile(null)
      goToCheckout(id, productId, ticketTile.checkoutUrl)
      return
    }

    if (isTicketExpired(ticketTile.expiresAt)) {
      dispatch(removeTicket(id))
      setTicketTile(null)
      return
    }

    void (async () => {
      setBuying(true)
      try {
        const result = await ticketApi.activateTicket(id)
        const checkoutUrl = result.checkout_url?.trim() || undefined
        dispatch(
          upsertTicket({
            id,
            productId,
            status: 'redeemed',
            checkoutUrl,
            availableActions: ['checkout'],
          }),
        )
        setTicketTile(null)
        goToCheckout(id, productId, checkoutUrl)
      } catch (error) {
        reportApiError(error, 'Не удалось активировать тикет')
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
        dispatch(leaveQueue(entryId))
        setLeaveTile(null)
        showToast('Вы вышли из очереди', 'info')
      } catch (error) {
        reportApiError(error, 'Не удалось выйти из очереди')
      } finally {
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
    ticketStatus?: QueueTileView['ticketStatus']
    checkoutUrl?: string
    availableActions?: QueueTileView['availableActions']
  }): QueueTileView => {
    const product = productItems.find((p) => p.id === entry.productId)
    return {
      ...entry,
      title: product?.title ?? 'Загрузка товара…',
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
        ticketStatus: ticket.status,
        checkoutUrl: ticket.checkoutUrl,
        availableActions: ticket.availableActions,
      }),
    ),
  ]

  const activeCount = queueTiles.filter((t) => t.kind === 'queued').length
  const ticketCount = queueTiles.filter((t) => t.kind === 'ticket').length
  const doneCount = queueTiles.filter((t) => t.kind === 'soldout').length

  return (
    <section>
      <div className="mb-5 flex flex-col items-start justify-between gap-4 sm:mb-6 sm:flex-row sm:items-end">
        <div className="min-w-0">
          <h1 className="m-0 text-[26px] font-extrabold tracking-tight sm:text-[32px]">
            Мои очереди
          </h1>
          <p className="mt-2 max-w-[620px] text-[14px] leading-normal text-avito-muted sm:text-[15px]">
            Каждая плитка — отдельная очередь. Откройте товар, чтобы посмотреть
            позицию, право на покупку или продолжить оформление.
          </p>
        </div>
        <Link
          to="/catalog"
          className="inline-flex min-h-10 shrink-0 items-center rounded-2xl bg-[#ebebeb] px-4 py-2 text-sm font-extrabold text-avito-ink no-underline"
        >
          ← В каталог
        </Link>
      </div>

      <div className="mb-5 grid grid-cols-3 gap-2 sm:mb-6 sm:gap-3.5">
        <SummaryTile label="Активных очередей" value={activeCount} />
        <SummaryTile label="Доступно к покупке" value={ticketCount} />
        <SummaryTile label="Завершено" value={doneCount} />
      </div>

      {queueTiles.length === 0 ? (
        <QueueEmpty />
      ) : (
        <div className="grid grid-cols-1 gap-3 min-[560px]:grid-cols-2 lg:grid-cols-3 lg:gap-4">
          {queueTiles.map((tile) => (
            <QueueCard
              key={`${tile.kind}-${tile.id}`}
              tile={tile}
              onOpenTicket={() => {
                if (tile.ticketStatus === 'redeemed' || tile.checkoutUrl) {
                  goToCheckout(tile.id, tile.productId, tile.checkoutUrl)
                  return
                }
                setTicketTile(tile)
              }}
              onLeaveQueue={() => setLeaveTile(tile)}
            />
          ))}
        </div>
      )}

      <TicketPurchaseModal
        open={ticketTile !== null}
        productTitle={ticketTile?.title ?? ''}
        productImage={ticketTile?.image ?? '🛒'}
        expiresAt={ticketTile?.expiresAt}
        buying={buying}
        canActivate={ticketAllows(ticketTile, 'activate')}
        canDecline={ticketAllows(ticketTile, 'decline')}
        onClose={closeTicketModal}
        onBuy={handleActivateAndBuy}
        onDecline={() => {
          if (!ticketTile || buying) return
          if (!ticketAllows(ticketTile, 'decline')) return
          const id = ticketTile.id
          void (async () => {
            try {
              await ticketApi.declineTicket(id)
              dispatch(removeTicket(id))
              setTicketTile(null)
              showToast('Вы отказались от тикета', 'info')
            } catch (error) {
              reportApiError(error, 'Не удалось отказаться от тикета')
            }
          })()
        }}
      />

      <LeaveConfirmModal
        open={leaveTile !== null}
        productTitle={leaveTile?.title ?? ''}
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
