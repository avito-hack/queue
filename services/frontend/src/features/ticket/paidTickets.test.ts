import { beforeEach, describe, expect, it } from 'vitest'
import { isTicketPaid, markTicketPaid } from './paidTickets'

describe('paidTickets', () => {
  beforeEach(() => {
    localStorage.clear()
    localStorage.setItem('userId', 'user-a')
  })

  it('marks ticket paid for current user', () => {
    // given
    markTicketPaid('t-1')

    // when / then
    expect(isTicketPaid('t-1')).toBe(true)
    expect(isTicketPaid('t-2')).toBe(false)
  })

  it('keeps paid tickets isolated per user', () => {
    // given
    markTicketPaid('t-1', 'user-a')
    markTicketPaid('t-2', 'user-b')

    // when / then
    expect(isTicketPaid('t-1', 'user-a')).toBe(true)
    expect(isTicketPaid('t-2', 'user-a')).toBe(false)
    expect(isTicketPaid('t-2', 'user-b')).toBe(true)
  })
})
