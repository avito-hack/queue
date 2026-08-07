import { describe, expect, it } from 'vitest'
import reducer, {
  joinQueue,
  leaveQueue,
  removeQueuedByProductId,
  updateQueueItem,
} from './queueSlice'
import type { QueueEntry } from './types'

const queued: QueueEntry = {
  id: 'e1',
  productId: 'p1',
  status: 'queued',
  position: 3,
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
    const updated = { ...queued, position: 1 }
    const state = reducer(withOne, updateQueueItem(updated))
    expect(state.queueItems).toEqual([updated])
  })

  it('removes queued entry by productId', () => {
    const withOne = reducer(undefined, joinQueue(queued))
    const state = reducer(withOne, removeQueuedByProductId('p1'))
    expect(state.queueItems).toEqual([])
  })
})
