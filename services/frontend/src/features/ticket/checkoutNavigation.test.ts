import { describe, expect, it } from 'vitest'
import { resolveCheckoutNavigation } from './checkoutNavigation'

describe('resolveCheckoutNavigation', () => {
  it('falls back to demo checkout when url is missing', () => {
    expect(resolveCheckoutNavigation('t-1')).toEqual({
      kind: 'internal',
      path: '/checkout?ticket=t-1',
    })
  })

  it('treats absolute url as external', () => {
    expect(
      resolveCheckoutNavigation('t-1', 'https://avito.example/checkout/1'),
    ).toEqual({
      kind: 'external',
      url: 'https://avito.example/checkout/1',
    })
  })

  it('treats relative url as internal navigate path', () => {
    expect(resolveCheckoutNavigation('t-1', '/orders/abc')).toEqual({
      kind: 'internal',
      path: '/orders/abc',
    })
  })
})
