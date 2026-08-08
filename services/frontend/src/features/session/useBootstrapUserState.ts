import { useEffect } from 'react'
import { useAppDispatch } from '../../app/hooks'
import { queueApi } from '../queue/api'
import { queueEntriesFromUserQueues } from '../queue/positionLib'
import { setQueueItems } from '../queue/queueSlice'
import { ticketApi } from '../ticket/api'
import { setTicketItems } from '../ticket/ticketSlice'
import type { TicketEntry } from '../ticket/types'
import { toTicketEntry } from '../ticket/useTicketPolling'


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

      let tickets: TicketEntry[] = []
      if (ticketsResult.status === 'fulfilled') {
        const list = ticketsResult.value.ticket ?? []
        tickets = list
          .map(toTicketEntry)
          .filter((item: TicketEntry | null): item is TicketEntry => item !== null)
        dispatch(setTicketItems(tickets))
      }

      if (queuesResult.status === 'fulfilled') {
        const ticketProductIds = new Set(tickets.map((t) => t.productId))
        const queues = queueEntriesFromUserQueues(queuesResult.value).filter(
          (q) => !ticketProductIds.has(q.productId),
        )
        dispatch(setQueueItems(queues))
      }
    })()

    return () => {
      cancelled = true
    }
  }, [dispatch])
}
