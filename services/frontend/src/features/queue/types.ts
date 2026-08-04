export type QueueStatus = 'queued' | 'ticket' | 'soldout'

export type QueueEntry = {
  id: string
  productId: string
  status: QueueStatus
  position?: number
}
