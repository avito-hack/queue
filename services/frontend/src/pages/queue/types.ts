import type { Product } from '../../features/product/types'

export type TileKind = 'queued' | 'ticket' | 'soldout'

export type QueueTileView = {
  id: string
  productId: string
  kind: TileKind
  position?: number
  expiresAt?: string
} & Pick<Product, 'name' | 'image'>
