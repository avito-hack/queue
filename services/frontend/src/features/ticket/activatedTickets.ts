import { currentUserId } from '../../shared/auth/currentUser'
import { isTicketPaid } from './paidTickets'
import type { TicketEntry } from './types'

const STORAGE_KEY = 'activatedTickets'

type Store = Record<string, TicketEntry[]>

function readAll(): Store {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as Store
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function writeAll(value: Store) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(value))
}

export function rememberActivatedTicket(
  ticket: TicketEntry,
  userId: string = currentUserId(),
): void {
  if (!ticket.id || isTicketPaid(ticket.id, userId)) return
  const entry: TicketEntry = {
    id: ticket.id,
    productId: ticket.productId,
    status: 'redeemed',
    checkoutUrl:
      ticket.checkoutUrl?.trim() ||
      `/checkout?ticket=${encodeURIComponent(ticket.id)}`,
    availableActions: ['checkout'],
  }
  const all = readAll()
  const current = all[userId] ?? []
  const next = current.filter((item) => item.id !== entry.id)
  next.push(entry)
  all[userId] = next
  writeAll(all)
}

export function forgetActivatedTicket(
  ticketId: string,
  userId: string = currentUserId(),
): void {
  const id = ticketId.trim()
  if (!id) return
  const all = readAll()
  const current = all[userId] ?? []
  all[userId] = current.filter((item) => item.id !== id)
  writeAll(all)
}

export function readActivatedTickets(
  userId: string = currentUserId(),
): TicketEntry[] {
  return (readAll()[userId] ?? []).filter(
    (item) => item.id && item.productId && !isTicketPaid(item.id, userId),
  )
}

export function mergeTicketLists(
  fromApi: TicketEntry[],
  fromLocal: TicketEntry[] = readActivatedTickets(),
): TicketEntry[] {
  const byId = new Map<string, TicketEntry>()
  for (const ticket of fromLocal) {
    byId.set(ticket.id, ticket)
  }
  for (const ticket of fromApi) {
    const prev = byId.get(ticket.id)
    byId.set(ticket.id, {
      ...prev,
      ...ticket,
      checkoutUrl: ticket.checkoutUrl || prev?.checkoutUrl,
      status: ticket.status ?? prev?.status,
      availableActions: ticket.availableActions?.length
        ? ticket.availableActions
        : prev?.availableActions,
    })
  }
  return [...byId.values()].filter((ticket) => !isTicketPaid(ticket.id))
}
