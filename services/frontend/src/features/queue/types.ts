import type { Product } from "../product/types"

export type QueueStatus = 'queued' | 'ticket' | 'soldout'

export type QueueEntry = {
    id: string
    product: Product
    status: QueueStatus
    position?: number
}