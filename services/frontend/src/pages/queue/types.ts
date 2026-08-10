import type {
  TicketAvailableAction,
  TicketStatus,
} from '../../features/ticket/types'

export type TileKind = 'queued' | 'ticket' | 'soldout'

export type QueueTileView = {
  id: string
  productId: string
  kind: TileKind
  position?: number
  expiresAt?: string
  title: string
  image: string
  ticketStatus?: TicketStatus
  checkoutUrl?: string
  availableActions?: TicketAvailableAction[]
}
