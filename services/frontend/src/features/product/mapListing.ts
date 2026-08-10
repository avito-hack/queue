import type { ListingStatus, Product } from './types'

/** Сырой Listing из avito-adapter (имена как в OpenAPI). */
export type ListingDto = {
  id?: string
  sellerId?: string
  title?: string
  price?: number
  quantity?: number
  /** Может отсутствовать в актуальной схеме — тогда = quantity. */
  reservedQuantity?: number
  availableQuantity?: number
  queueEnabled?: boolean
  status?: ListingStatus
  createdAt?: string
  updatedAt?: string
}

export type ListingListResponse = {
  items?: ListingDto[]
  total?: number
  limit?: number
  offset?: number
}

const FALLBACK_ISO = '1970-01-01T00:00:00.000Z'

const DEMO_IMAGES = ['👟', '🧥', '🧢', '🎒', '👕', '🕶️']

/** Стабильная демо-картинка по id (в Listing её нет). */
export function demoImageForListing(id: string): string {
  let hash = 0
  for (let i = 0; i < id.length; i += 1) {
    hash = (hash + id.charCodeAt(i)) % DEMO_IMAGES.length
  }
  return DEMO_IMAGES[hash] ?? '🛒'
}

/**
 * Listing (бэк) → Product (store).
 * UI-поля image/description/queueCount в схеме нет.
 */
export function mapListingToProduct(dto: ListingDto): Product | null {
  if (!dto.id || !dto.title || dto.price == null || dto.quantity == null) {
    return null
  }

  const quantity = dto.quantity
  const availableQuantity = dto.availableQuantity ?? quantity
  const reservedQuantity =
    dto.reservedQuantity ?? Math.max(0, quantity - availableQuantity)

  return {
    id: dto.id,
    sellerId: dto.sellerId ?? '',
    title: dto.title,
    price: dto.price,
    quantity,
    reservedQuantity,
    availableQuantity,
    queueEnabled: dto.queueEnabled ?? false,
    status: dto.status ?? 'active',
    createdAt: dto.createdAt ?? FALLBACK_ISO,
    updatedAt: dto.updatedAt ?? FALLBACK_ISO,
    image: demoImageForListing(dto.id),
  }
}
