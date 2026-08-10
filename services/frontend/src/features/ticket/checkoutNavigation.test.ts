import { afterEach, describe, expect, it, vi } from 'vitest'
import { resolveCheckoutNavigation } from './checkoutNavigation'

describe('resolveCheckoutNavigation', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('falls back to demo checkout when url is missing', () => {
    expect(resolveCheckoutNavigation('t-1')).toEqual({
      kind: 'internal',
      path: '/checkout?ticket=t-1',
    })
  })

  it('maps avito-adapter checkout.local mock to SPA checkout', () => {
    expect(
      resolveCheckoutNavigation(
        't-1',
        'https://checkout.local/orders/ord-1?sku_id=sku-1',
      ),
    ).toEqual({
      kind: 'internal',
      path: '/checkout?ticket=t-1',
    })
  })

  it('treats real absolute url as external', () => {
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

  it('ensures ticket query on relative checkout path', () => {
    expect(resolveCheckoutNavigation('t-1', '/checkout')).toEqual({
      kind: 'internal',
      path: '/checkout?ticket=t-1',
    })
  })

  it('keeps relative checkout path that already has ticket', () => {
    expect(
      resolveCheckoutNavigation('t-1', '/checkout?ticket=t-1'),
    ).toEqual({
      kind: 'internal',
      path: '/checkout?ticket=t-1',
    })
  })
})
