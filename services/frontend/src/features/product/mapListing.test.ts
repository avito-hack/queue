import { describe, expect, it } from 'vitest'
import { mapListingToProduct } from './mapListing'

describe('mapListingToProduct', () => {
  it('maps OpenAPI Listing fields into Product', () => {
    expect(
      mapListingToProduct({
        id: '11111111-1111-4111-8111-111111111111',
        sellerId: '22222222-2222-4222-8222-222222222222',
        title: 'Куртка',
        price: 12800,
        quantity: 5,
        reservedQuantity: 1,
        availableQuantity: 4,
        queueEnabled: true,
        status: 'active',
        createdAt: '2026-08-06T10:00:00.000Z',
        updatedAt: '2026-08-06T11:00:00.000Z',
      }),
    ).toEqual({
      id: '11111111-1111-4111-8111-111111111111',
      sellerId: '22222222-2222-4222-8222-222222222222',
      title: 'Куртка',
      price: 12800,
      quantity: 5,
      reservedQuantity: 1,
      availableQuantity: 4,
      queueEnabled: true,
      status: 'active',
      createdAt: '2026-08-06T10:00:00.000Z',
      updatedAt: '2026-08-06T11:00:00.000Z',
    })
  })

  it('returns null when required listing fields are missing', () => {
    expect(mapListingToProduct({ title: 'x' })).toBeNull()
  })
})
