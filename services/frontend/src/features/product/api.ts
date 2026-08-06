import { api } from '../../shared/api/client'
import { mapListingToProduct } from './mapListing'
import type { Product } from './types'

/**
 * Один listing по id.
 * Через nginx: /v1/avito/... → adapter (нужен rewrite на /v1/listings).
 * По OpenAPI сервиса путь: GET /v1/listings/{listingId}.
 */
const getListing = async (listingId: string): Promise<Product> => {
  const response = await api.get(`/v1/avito/listings/${listingId}`)
  const product = mapListingToProduct(response.data)
  if (!product) {
    throw new Error('Invalid listing payload')
  }
  return product
}

/**
 * Списка всех listings в OpenAPI нет.
 * Если бэк когда-нибудь отдаст массив — замапим; иначе Catalog уйдёт в mock.
 */
const getProducts = async (): Promise<Product[]> => {
  const response = await api.get('/v1/avito/listings')
  const raw = response.data
  const list = Array.isArray(raw) ? raw : Array.isArray(raw?.listings) ? raw.listings : null
  if (!list) {
    throw new Error('Listings list is not available')
  }
  return list
    .map(mapListingToProduct)
    .filter((item: Product | null): item is Product => item !== null)
}

export const productApi = {
  getListing,
  getProducts,
}
