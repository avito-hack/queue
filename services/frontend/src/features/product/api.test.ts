import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '../../shared/api/client'
import { productApi } from './api'

vi.mock('../../shared/api/client', () => ({
  api: {
    get: vi.fn(),
  },
}))

describe('productApi', () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset()
  })

  it('getProducts maps ListingListResponse items', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        items: [
          {
            id: '11111111-1111-4111-8111-111111111111',
            sellerId: '22222222-2222-4222-8222-222222222222',
            title: 'Куртка',
            price: 12800,
            quantity: 5,
            queueEnabled: true,
            status: 'active',
            createdAt: '2026-08-06T10:00:00.000Z',
            updatedAt: '2026-08-06T11:00:00.000Z',
          },
        ],
        total: 1,
        limit: 100,
        offset: 0,
      },
    })

    const products = await productApi.getProducts()

    expect(api.get).toHaveBeenCalledWith('/v1/avito/listings', {
      params: { status: 'active', limit: 100, offset: 0 },
    })
    expect(products).toHaveLength(1)
    expect(products[0]?.title).toBe('Куртка')
    expect(products[0]?.availableQuantity).toBe(5)
  })

  it('getListing maps a single Listing', async () => {
    vi.mocked(api.get).mockResolvedValue({
      data: {
        id: '11111111-1111-4111-8111-111111111111',
        sellerId: '22222222-2222-4222-8222-222222222222',
        title: 'Куртка',
        price: 12800,
        quantity: 5,
        queueEnabled: true,
        status: 'active',
        createdAt: '2026-08-06T10:00:00.000Z',
        updatedAt: '2026-08-06T11:00:00.000Z',
      },
    })

    const product = await productApi.getListing(
      '11111111-1111-4111-8111-111111111111',
    )

    expect(api.get).toHaveBeenCalledWith(
      '/v1/avito/listings/11111111-1111-4111-8111-111111111111',
    )
    expect(product.id).toBe('11111111-1111-4111-8111-111111111111')
  })
})
