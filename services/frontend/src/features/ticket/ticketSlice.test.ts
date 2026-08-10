import { describe, expect, it } from 'vitest'
import reducer, {
  removeTicket,
  setTicketItems,
  upsertTicket,
} from './ticketSlice'
import type { TicketEntry } from './types'

const ticket: TicketEntry = {
  id: 't-1',
  productId: 'p-1',
  expiresAt: '2026-08-05T12:10:00.000Z',
}

describe('ticketSlice', () => {
  it('adds a ticket', () => {
    const state = reducer(undefined, upsertTicket(ticket))
    expect(state.ticketItems).toEqual([ticket])
  })

  it('updates ticket with the same id', () => {
    const withOne = reducer(undefined, upsertTicket(ticket))
    const updated = { ...ticket, expiresAt: '2026-08-05T12:20:00.000Z' }
    const state = reducer(withOne, upsertTicket(updated))
    expect(state.ticketItems).toEqual([updated])
  })

  it('removes ticket by id', () => {
    const withOne = reducer(undefined, upsertTicket(ticket))
    const state = reducer(withOne, removeTicket('t-1'))
    expect(state.ticketItems).toEqual([])
  })

  it('replaces all items on setTicketItems (hydrate)', () => {
    const withOne = reducer(undefined, upsertTicket(ticket))
    const next = [{ ...ticket, id: 't-2' }]
    const state = reducer(withOne, setTicketItems(next))
    expect(state.ticketItems).toEqual(next)
  })
})
