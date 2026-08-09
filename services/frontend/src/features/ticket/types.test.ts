import { describe, expect, it } from 'vitest'
import { ticketAllows, type TicketEntry } from './types'
import { toTicketEntry } from './useTicketPolling'

describe('ticketAllows', () => {
  it('defaults to activate/decline/checkout when actions missing', () => {
    expect(ticketAllows({}, 'activate')).toBe(true)
    expect(ticketAllows({ availableActions: [] }, 'decline')).toBe(true)
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

  it('ignores redeemed tickets after activate', () => {
    expect(
      toTicketEntry({
        id: 't1',
        listing_id: 'p1',
        status: 'redeemed',
      }),
    ).toBeNull()
  })
})
