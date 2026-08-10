import axios from 'axios'
import { ticketApi } from './api'
import type { TicketEntry } from './types'

export function isActivateConflictError(error: unknown): boolean {
  if (!axios.isAxiosError(error) || error.response?.status !== 409) {
    return false
  }
  const data = error.response.data
  if (!data || typeof data !== 'object') return true
  const record = data as Record<string, unknown>
  const code =
    (typeof record.error === 'string' && record.error) ||
    (typeof record.code === 'string' && record.code) ||
    ''
  return (
    code === '' ||
    code === 'ticket_not_activatable' ||
    code === 'idempotency_conflict' ||
    code === 'activation_in_progress' ||
    code === 'checkout_rejected' ||
    code === 'conflict'
  )
}

export async function resolveActivatedTicket(
  ticketId: string,
  productId: string,
): Promise<TicketEntry> {
  const fallbackUrl = `/checkout?ticket=${encodeURIComponent(ticketId)}`
  try {
    const dto = (await ticketApi.getTicket(ticketId)) as {
      id?: string
      listing_id?: string
      listingId?: string
      status?: string
      checkout_url?: string | null
      checkoutUrl?: string | null
    }
    const checkoutUrl =
      (typeof dto.checkout_url === 'string' && dto.checkout_url.trim()) ||
      (typeof dto.checkoutUrl === 'string' && dto.checkoutUrl.trim()) ||
      fallbackUrl
    return {
      id: dto.id || ticketId,
      productId: dto.listing_id || dto.listingId || productId,
      status: 'redeemed',
      checkoutUrl,
      availableActions: ['checkout'],
    }
  } catch {
    return {
      id: ticketId,
      productId,
      status: 'redeemed',
      checkoutUrl: fallbackUrl,
      availableActions: ['checkout'],
    }
  }
}
