import { useEffect } from 'react'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { removeQueuedByProductId } from '../queue/queueSlice'
import { ticketApi } from './api'
import { upsertTicket } from './ticketSlice'
import type { TicketEntry } from './types'

const POLL_MS = 5000
const ACTIVE_STATUSES = new Set(['issued', 'active'])

function toTicketEntry(dto: {
  id?: string
  listing_id?: string
  status?: string
  activation_deadline?: string
}): TicketEntry | null {
  if (!dto.id || !dto.listing_id || !ACTIVE_STATUSES.has(dto.status ?? '')) {
    return null
  }
  return {
    id: dto.id,
    productId: dto.listing_id,
    expiresAt: dto.activation_deadline,
  }
}

export function useTicketPolling(enabled = true) {
  const dispatch = useAppDispatch()
  const hasQueued = useAppSelector((state) =>
    state.queue.queueItems.some((item) => item.status === 'queued'),
  )
  const hasTickets = useAppSelector(
    (state) => state.tickets.ticketItems.length > 0,
  )

  useEffect(() => {
    if (!enabled || (!hasQueued && !hasTickets)) return

    let cancelled = false

    const sync = async () => {
      try {
        const data = await ticketApi.listTickets()
        if (cancelled) return

        const list = Array.isArray(data?.ticket) ? data.ticket : []
        for (const dto of list) {
          const ticket = toTicketEntry(dto)
          if (!ticket) continue
          dispatch(removeQueuedByProductId(ticket.productId))
          dispatch(upsertTicket(ticket))
        }
      } catch {
        // бэк ещё не готов
      }
    }

    void sync()
    const id = window.setInterval(() => {
      void sync()
    }, POLL_MS)

    return () => {
      cancelled = true
      window.clearInterval(id)
    }
  }, [dispatch, enabled, hasQueued, hasTickets])
}
