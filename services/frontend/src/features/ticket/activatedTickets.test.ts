import { beforeEach, describe, expect, it } from 'vitest'
import {
  forgetActivatedTicket,
  mergeTicketLists,
  readActivatedTickets,
  rememberActivatedTicket,
} from './activatedTickets'

describe('activatedTickets', () => {
  beforeEach(() => {
    localStorage.clear()
    localStorage.setItem('userId', 'user-a')
  })

  it('remembers activated ticket across reads', () => {
    // given
    rememberActivatedTicket({
      id: 't-1',
      productId: 'p-1',
      status: 'redeemed',
      checkoutUrl: '/checkout?ticket=t-1',
    })

    // when / then
    expect(readActivatedTickets()).toEqual([
      {
        id: 't-1',
        productId: 'p-1',
        status: 'redeemed',
        checkoutUrl: '/checkout?ticket=t-1',
        availableActions: ['checkout'],
      },
    ])
  })

  it('keeps local activated ticket when api list is empty', () => {
    // given
    rememberActivatedTicket({
      id: 't-1',
      productId: 'p-1',
      status: 'redeemed',
    })

    // when
    const merged = mergeTicketLists([])

    // then
    expect(merged).toHaveLength(1)
    expect(merged[0]?.id).toBe('t-1')
  })

  it('forgets activated ticket after pay', () => {
    // given
    rememberActivatedTicket({
      id: 't-1',
      productId: 'p-1',
      status: 'redeemed',
    })

    // when
    forgetActivatedTicket('t-1')

    // then
    expect(readActivatedTickets()).toEqual([])
  })
})
