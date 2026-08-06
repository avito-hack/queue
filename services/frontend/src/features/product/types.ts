/** Статус объявления — как в avito-adapter ListingStatus. */
export type ListingStatus = 'active' | 'paused' | 'removed'

/**
 * Товар в store = Listing из OpenAPI
 * + опциональные UI-поля, которых в схеме нет.
 */
export type Product = {
  id: string
  sellerId: string
  title: string
  price: number
  quantity: number
  reservedQuantity: number
  availableQuantity: number
  queueEnabled: boolean
  status: ListingStatus
  createdAt: string
  updatedAt: string
  /** Нет в Listing — только для демо-UI */
  image?: string
  description?: string
  /** Нет в Listing — число желающих; пока мок/позже другой API */
  queueCount?: number
}
