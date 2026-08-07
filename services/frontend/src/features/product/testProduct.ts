import type { Product } from './types'

const now = '2026-08-06T12:00:00.000Z'

/** Короткий Listing-shaped fixture для тестов. */
export function makeProduct(overrides: Partial<Product> = {}): Product {
  return {
    id: '11111111-1111-4111-8111-111111111101',
    sellerId: '00000000-0000-4000-8000-000000000010',
    title: 'Куртка Northline Shell',
    description: 'Лимитированная коллекция',
    price: 12800,
    image: '🧥',
    quantity: 2,
    reservedQuantity: 0,
    availableQuantity: 2,
    queueEnabled: true,
    status: 'active',
    queueCount: 5,
    createdAt: now,
    updatedAt: now,
    ...overrides,
  }
}
