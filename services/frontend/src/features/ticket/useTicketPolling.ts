import { useEffect } from 'react'
import { useAppDispatch } from '../../app/hooks'
import { removeQueuedByProductId } from '../queue/queueSlice'
import { ticketApi } from './api'
import { mergeTicketLists } from './activatedTickets'
import { isTicketPaid } from './paidTickets'
import { setTicketItems } from './ticketSlice'
import type {
  TicketAvailableAction,
  TicketEntry,
  TicketStatus,
} from './types'

const POLL_MS = 5000

const ACTIONS = new Set<TicketAvailableAction>([
  'activate',
  'decline',
  'checkout',
])

function readCheckoutUrl(dto: {
  checkout_url?: string | null
  checkoutUrl?: string | null
}): string | undefined {
  const raw = dto.checkout_url ?? dto.checkoutUrl
  const value = typeof raw === 'string' ? raw.trim() : ''
  return value || undefined
}

export function toTicketEntry(dto: {
  id?: string
  listing_id?: string
  status?: string
  activation_deadline?: string
  available_actions?: string[]
  checkout_url?: string | null
  checkoutUrl?: string | null
  finished_at?: string | null
}): TicketEntry | null {
  if (!dto.id || !dto.listing_id) return null
  if (isTicketPaid(dto.id)) return null
  if (dto.finished_at) return null

  const status = dto.status as TicketStatus | undefined
  let checkoutUrl = readCheckoutUrl(dto)

  if (status === 'issued') {
    // ok
  } else if (status === 'redeemed') {
    checkoutUrl =
      checkoutUrl || `/checkout?ticket=${encodeURIComponent(dto.id)}`
  } else {
    return null
  }

  const parsedActions = Array.isArray(dto.available_actions)
    ? dto.available_actions.filter((a): a is TicketAvailableAction =>
        ACTIONS.has(a as TicketAvailableAction),
      )
    : undefined

  const availableActions =
    parsedActions && parsedActions.length > 0
      ? parsedActions
      : status === 'redeemed'
        ? (['checkout'] as TicketAvailableAction[])
        : parsedActions

  return {
    id: dto.id,
    productId: dto.listing_id,
    expiresAt: status === 'redeemed' ? undefined : dto.activation_deadline,
    status,
    checkoutUrl,
    availableActions,
  }
}

export function useTicketPolling(enabled = true) {
  const dispatch = useAppDispatch()

  useEffect(() => {
    if (!enabled) return

    let cancelled = false

    const sync = async () => {
      try {
        const data = await ticketApi.listTickets()
        if (cancelled) return

        const list = Array.isArray(data?.ticket) ? data.ticket : []
        const fromApi: TicketEntry[] = []
        for (const dto of list) {
          const ticket = toTicketEntry(dto)
          if (!ticket) continue
          dispatch(removeQueuedByProductId(ticket.productId))
          fromApi.push(ticket)
        }
        dispatch(setTicketItems(mergeTicketLists(fromApi)))
      } catch {
        if (cancelled) return
        const local = mergeTicketLists([])
        if (local.length > 0) {
          dispatch(setTicketItems(local))
        }
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
  }, [dispatch, enabled])
}
