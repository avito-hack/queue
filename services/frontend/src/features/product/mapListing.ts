import type { ListingStatus, Product } from './types'

/** Сырой Listing из avito-adapter (имена как в OpenAPI). */
export type ListingDto = {
  id?: string
  sellerId?: string
  title?: string
  price?: number
  quantity?: number
  reservedQuantity?: number
  availableQuantity?: number
  queueEnabled?: boolean
  status?: ListingStatus
  createdAt?: string
  updatedAt?: string
}

const FALLBACK_ISO = '1970-01-01T00:00:00.000Z'

/**
 * Listing (бэк) → Product (store).
 * UI-поля image/description/queueCount сюда не входят — их нет в схеме.
 */
export function mapListingToProduct(dto: ListingDto): Product | null {
  if (!dto.id || !dto.title || dto.price == null) return null
  if (dto.availableQuantity == null || dto.quantity == null) return null

  return {
    id: dto.id,
    sellerId: dto.sellerId ?? '',
    title: dto.title,
    price: dto.price,
    quantity: dto.quantity,
    reservedQuantity: dto.reservedQuantity ?? 0,
    availableQuantity: dto.availableQuantity,
    queueEnabled: dto.queueEnabled ?? false,
    status: dto.status ?? 'active',
    createdAt: dto.createdAt ?? FALLBACK_ISO,
    updatedAt: dto.updatedAt ?? FALLBACK_ISO,
  }
}
