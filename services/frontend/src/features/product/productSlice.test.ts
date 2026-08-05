import { describe, expect, it } from 'vitest'
import reducer, { setProductItems } from './productSlice'
import type { Product } from './types'

const products: Product[] = [
  {
    id: '1',
    name: 'Кроссовки',
    description: 'Drop',
    price: 1000,
    image: '👟',
    count: 2,
    queueCount: 5,
  },
]

describe('productSlice', () => {
  it('starts with an empty list', () => {
    const state = reducer(undefined, { type: 'unknown' })
    expect(state.productItems).toEqual([])
  })

  it('replaces product items', () => {
    const state = reducer(undefined, setProductItems(products))
    expect(state.productItems).toEqual(products)
  })
})
