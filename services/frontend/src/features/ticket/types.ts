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

/**
 * undefined — бэк не прислал actions, для демо разрешаем кнопки.
 * [] — явно пусто (например redeemed), действий нет.
 */
export function ticketAllows(
  ticket: Pick<TicketEntry, 'availableActions'> | null | undefined,
  action: TicketAvailableAction,
): boolean {
  const actions = ticket?.availableActions
  if (actions === undefined) {
    return action === 'activate' || action === 'decline' || action === 'checkout'
  }
  if (actions.length === 0) return false
  return actions.includes(action)
}
