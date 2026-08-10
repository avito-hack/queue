import { beforeEach, describe, expect, it } from 'vitest'
import { ticketAllows, type TicketEntry } from './types'
import { toTicketEntry } from './useTicketPolling'

describe('ticketAllows', () => {
  it('defaults to activate/decline/checkout when actions missing', () => {
    expect(ticketAllows({}, 'activate')).toBe(true)
    expect(ticketAllows({ availableActions: undefined }, 'decline')).toBe(true)
  })

  it('denies all actions when available_actions is explicitly empty', () => {
    expect(ticketAllows({ availableActions: [] }, 'activate')).toBe(false)
    expect(ticketAllows({ availableActions: [] }, 'checkout')).toBe(false)
  })

  it('respects available_actions from API', () => {
    const ticket: TicketEntry = {
      id: 't1',
      productId: 'p1',
      availableActions: ['decline'],
    }
    expect(ticketAllows(ticket, 'activate')).toBe(false)
    expect(ticketAllows(ticket, 'decline')).toBe(true)
  })
})

describe('toTicketEntry', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('maps V1Ticket fields including available_actions', () => {
    expect(
      toTicketEntry({
        id: 't1',
        listing_id: 'p1',
        status: 'issued',
        activation_deadline: '2099-01-01T00:00:00.000Z',
        available_actions: ['activate', 'decline'],
      }),
    ).toEqual({
      id: 't1',
      productId: 'p1',
      status: 'issued',
      expiresAt: '2099-01-01T00:00:00.000Z',
      availableActions: ['activate', 'decline'],
    })
  })

  it('ignores closed tickets', () => {
    expect(
      toTicketEntry({
        id: 't1',
        listing_id: 'p1',
        status: 'closed',
      }),
    ).toBeNull()
  })

  it('maps redeemed tickets even without checkout_url', () => {
    expect(
      toTicketEntry({
        id: 't1',
        listing_id: 'p1',
        status: 'redeemed',
      }),
    ).toEqual({
      id: 't1',
      productId: 'p1',
      status: 'redeemed',
      checkoutUrl: '/checkout?ticket=t1',
      availableActions: ['checkout'],
      expiresAt: undefined,
    })
  })

  it('keeps redeemed tickets with checkout_url for purchase tile', () => {
    expect(
      toTicketEntry({
        id: 't1',
        listing_id: 'p1',
        status: 'redeemed',
        checkout_url: '/checkout?ticket=t1',
        activation_deadline: '2099-01-01T00:00:00.000Z',
      }),
    ).toEqual({
      id: 't1',
      productId: 'p1',
      status: 'redeemed',
      checkoutUrl: '/checkout?ticket=t1',
      availableActions: ['checkout'],
      expiresAt: undefined,
    })
  })

  it('hides locally paid tickets even if backend still returns redeemed', () => {
    localStorage.setItem('userId', 'user-a')
    localStorage.setItem(
      'paidTickets',
      JSON.stringify({ 'user-a': ['t-paid'] }),
    )

    expect(
      toTicketEntry({
        id: 't-paid',
        listing_id: 'p1',
        status: 'redeemed',
        checkout_url: '/checkout?ticket=t-paid',
      }),
    ).toBeNull()
  })
})
