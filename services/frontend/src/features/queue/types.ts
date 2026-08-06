export type QueueStatus = 'queued' | 'soldout'

export type QueueEntry = {
  id: string
  productId: string
  status: QueueStatus
  position?: number
}
