import { useEffect } from 'react'
import { useAppDispatch } from '../../app/hooks'
import { productApi } from '../product/api'
import { upsertProduct } from '../product/productSlice'
import { queueApi } from '../queue/api'
import { queueEntriesFromUserQueues } from '../queue/positionLib'
import { setQueueItems } from '../queue/queueSlice'
import { mergeTicketLists } from '../ticket/activatedTickets'
import { ticketApi } from '../ticket/api'
import { setTicketItems } from '../ticket/ticketSlice'
import type { TicketEntry } from '../ticket/types'
import { toTicketEntry } from '../ticket/useTicketPolling'

async function hydrateProducts(
  productIds: string[],
  dispatch: ReturnType<typeof useAppDispatch>,
  cancelled: () => boolean,
) {
  const uniqueIds = [...new Set(productIds.filter(Boolean))]
  if (uniqueIds.length === 0) return

  const results = await Promise.allSettled(
    uniqueIds.map((id) => productApi.getListing(id)),
  )

  if (cancelled()) return

  for (const result of results) {
    if (result.status === 'fulfilled') {
      dispatch(upsertProduct(result.value))
    }
  }
}

export function useBootstrapUserState() {
  const dispatch = useAppDispatch()

  useEffect(() => {
    let cancelled = false

    void (async () => {
      const [queuesResult, ticketsResult] = await Promise.allSettled([
        queueApi.listUserQueues(),
        ticketApi.listTickets(),
      ])

      if (cancelled) return

      const tickets: TicketEntry[] =
        ticketsResult.status === 'fulfilled'
          ? mergeTicketLists(
              (ticketsResult.value.ticket ?? [])
                .map(toTicketEntry)
                .filter(
                  (item: TicketEntry | null): item is TicketEntry =>
                    item !== null,
                ),
            )
          : mergeTicketLists([])

      if (ticketsResult.status === 'fulfilled' || tickets.length > 0) {
        dispatch(setTicketItems(tickets))
      }

      let queueProductIds: string[] = []
      if (queuesResult.status === 'fulfilled') {
        const ticketProductIds = new Set(tickets.map((t) => t.productId))
        const queues = queueEntriesFromUserQueues(queuesResult.value).filter(
          (q) => !ticketProductIds.has(q.productId),
        )
        queueProductIds = queues.map((q) => q.productId)
        dispatch(setQueueItems(queues))
      }

      await hydrateProducts(
        [...queueProductIds, ...tickets.map((t) => t.productId)],
        dispatch,
        () => cancelled,
      )
    })()

    return () => {
      cancelled = true
    }
  }, [dispatch])
}
