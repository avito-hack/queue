import { useEffect } from 'react'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { removeQueuedByProductId } from '../queue/queueSlice'
import { ticketApi } from './api'
import { upsertTicket } from './ticketSlice'
import type {
  TicketAvailableAction,
  TicketEntry,
  TicketStatus,
} from './types'

const POLL_MS = 5000
const ACTIVE_STATUSES = new Set<TicketStatus>(['issued', 'active'])

const ACTIONS = new Set<TicketAvailableAction>([
  'activate',
  'decline',
  'checkout',
])

export function toTicketEntry(dto: {
  id?: string
  listing_id?: string
  status?: string
  activation_deadline?: string
  available_actions?: string[]
}): TicketEntry | null {
  if (!dto.id || !dto.listing_id || !ACTIVE_STATUSES.has(dto.status as TicketStatus)) {
    return null
  }

  const availableActions = Array.isArray(dto.available_actions)
    ? dto.available_actions.filter((a): a is TicketAvailableAction =>
        ACTIONS.has(a as TicketAvailableAction),
      )
    : undefined

  return {
    id: dto.id,
    productId: dto.listing_id,
    expiresAt: dto.activation_deadline,
    status: dto.status as TicketStatus,
    availableActions,
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
