import { api } from '../../shared/api/client'
import {
  mapListingToProduct,
  type ListingListResponse,
} from './mapListing'
import type { Product } from './types'

const getListing = async (listingId: string): Promise<Product> => {
  const response = await api.get(`/v1/avito/listings/${listingId}`)
  const product = mapListingToProduct(response.data)
  if (!product) {
    throw new Error('Invalid listing payload')
  }
  return product
}

const getProducts = async (): Promise<Product[]> => {
  const response = await api.get<ListingListResponse>('/v1/avito/listings', {
    params: {
      status: 'active',
      limit: 100,
      offset: 0,
    },
  })
  const list = Array.isArray(response.data?.items) ? response.data.items : null
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
