import { describe, expect, it } from 'vitest'
import reducer, {
  joinQueue,
  leaveQueue,
  updateQueueItem,
} from './queueSlice'
import type { QueueEntry } from './types'

const queued: QueueEntry = {
  id: 'e1',
  productId: 'p1',
  status: 'queued',
  position: 3,
}

const ticket: QueueEntry = {
  ...queued,
  status: 'ticket',
  expiresAt: '2026-08-05T12:10:00.000Z',
}

describe('queueSlice', () => {
  it('joins a new queue entry', () => {
    const state = reducer(undefined, joinQueue(queued))
    expect(state.queueItems).toEqual([queued])
  })

  it('does not add a duplicate entry by id', () => {
    const withOne = reducer(undefined, joinQueue(queued))
    const state = reducer(withOne, joinQueue({ ...queued, position: 99 }))
    expect(state.queueItems).toHaveLength(1)
    expect(state.queueItems[0]?.position).toBe(3)
  })

  it('removes entry on leaveQueue', () => {
    const withOne = reducer(undefined, joinQueue(queued))
    const state = reducer(withOne, leaveQueue('e1'))
    expect(state.queueItems).toEqual([])
  })

  it('updates an existing entry', () => {
    const withOne = reducer(undefined, joinQueue(queued))
    const state = reducer(withOne, updateQueueItem(ticket))
    expect(state.queueItems).toEqual([ticket])
  })
})
