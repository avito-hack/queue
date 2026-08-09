export type TicketStatus = 'issued' | 'redeemed' | 'closed'

export type TicketAvailableAction = 'activate' | 'decline' | 'checkout'

export type TicketEntry = {
  id: string
  productId: string
  expiresAt?: string
  status?: TicketStatus
  checkoutUrl?: string
  availableActions?: TicketAvailableAction[]
}

/** Если бэк не прислал actions — не блокируем демо-кнопки activate/decline. */
export function ticketAllows(
  ticket: Pick<TicketEntry, 'availableActions'> | null | undefined,
  action: TicketAvailableAction,
): boolean {
  const actions = ticket?.availableActions
  if (!actions || actions.length === 0) {
    return action === 'activate' || action === 'decline' || action === 'checkout'
  }
  return actions.includes(action)
}
