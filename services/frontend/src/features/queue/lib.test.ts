import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  formatCountdown,
  getProductActionLabel,
  isTicketExpired,
} from './lib'
import type { QueueEntry } from './types'
import type { TicketEntry } from '../ticket/types'

const queued: QueueEntry = {
  id: 'e1',
  productId: 'p1',
  status: 'queued',
  position: 3,
}

const ticket: TicketEntry = {
  id: 't1',
  productId: 'p1',
  expiresAt: '2026-08-05T12:10:00.000Z',
}

describe('getProductActionLabel', () => {
  it('returns join label when in stock and not in queue', () => {
    expect(getProductActionLabel(true)).toBe('Встать в очередь')
  })

  it('returns notify label when out of stock and not in queue', () => {
    expect(getProductActionLabel(false)).toBe('Уведомить о поступлении')
  })

  it('returns queue label when already queued', () => {
    expect(getProductActionLabel(true, queued)).toBe('Перейти к моим очередям')
  })

  it('returns purchase label when ticket exists', () => {
    expect(getProductActionLabel(true, null, ticket)).toBe('Перейти к покупке')
  })
})

describe('isTicketExpired', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-05T12:00:00.000Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('returns false when expiresAt is missing', () => {
    expect(isTicketExpired()).toBe(false)
  })

  it('returns true when deadline already passed', () => {
    expect(isTicketExpired('2026-08-05T11:59:59.000Z')).toBe(true)
  })

  it('returns false when deadline is in the future', () => {
    expect(isTicketExpired('2026-08-05T12:00:01.000Z')).toBe(false)
  })
})

describe('formatCountdown', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-05T12:00:00.000Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('returns null when expiresAt is missing', () => {
    expect(formatCountdown()).toBeNull()
  })

  it('returns 00:00 when time already passed', () => {
    expect(formatCountdown('2026-08-05T11:59:00.000Z')).toBe('00:00')
  })

  it('formats remaining minutes and seconds', () => {
    expect(formatCountdown('2026-08-05T12:05:07.000Z')).toBe('05:07')
  })

  it('formats hours when more than 60 minutes left', () => {
    expect(formatCountdown('2026-08-05T14:05:07.000Z')).toBe('02:05:07')
  })

  it('formats days when more than 24 hours left', () => {
    expect(formatCountdown('2026-08-07T14:05:07.000Z')).toBe('2д 02:05:07')
  })
})
