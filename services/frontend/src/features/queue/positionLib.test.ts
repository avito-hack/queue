import { describe, expect, it } from 'vitest'
import {
  queueItemWithPosition,
  queueItemsFromUserQueues,
} from './positionLib'
import type { QueueEntry } from './types'

const a: QueueEntry = {
  id: 'a',
  productId: 'listing-a',
  status: 'queued',
  position: 1,
}
const b: QueueEntry = {
  id: 'b',
  productId: 'listing-b',
  status: 'queued',
  position: 2,
}

describe('queueItemsFromUserQueues', () => {
  it('updates each tile by item_id from /v1/user/queues', () => {
    expect(
      queueItemsFromUserQueues([a, b], [
        { item_id: 'listing-a', position: 3, status: 'waiting_in_line' },
        { item_id: 'listing-b', position: 9, status: 'waiting_in_line' },
      ]),
    ).toEqual({
      updates: [
        { ...a, position: 3, memberStatus: 'waiting_in_line' },
        { ...b, position: 9, memberStatus: 'waiting_in_line' },
      ],
      removeProductIds: [],
    })
  })

  it('marks soldout when status is item_out_of_stock', () => {
    expect(
      queueItemsFromUserQueues([a], [
        {
          item_id: 'listing-a',
          position: 1,
          status: 'item_out_of_stock',
        },
      ]),
    ).toEqual({
      updates: [
        {
          ...a,
          status: 'soldout',
          position: undefined,
          memberStatus: 'item_out_of_stock',
        },
      ],
      removeProductIds: [],
    })
  })

  it('removes tile when user acquired purchase rights', () => {
    expect(
      queueItemsFromUserQueues([a], [
        {
          item_id: 'listing-a',
          position: 1,
          status: 'acquired_purchase_rights',
        },
      ]),
    ).toEqual({
      updates: [],
      removeProductIds: ['listing-a'],
    })
  })
})

describe('queueItemWithPosition', () => {
  it('updates only the matching itemId', () => {
    expect(queueItemWithPosition([a, b], 'listing-b', 7)).toEqual({
      ...b,
      position: 7,
    })
  })
})
