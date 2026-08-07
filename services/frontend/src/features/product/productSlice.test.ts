import { describe, expect, it } from 'vitest'
import reducer, { setProductItems } from './productSlice'
import { makeProduct } from './testProduct'

const products = [makeProduct({ title: 'Кроссовки' })]

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
