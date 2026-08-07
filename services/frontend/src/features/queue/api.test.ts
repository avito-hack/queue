import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../../shared/api/client', () => ({
  api: {
    post: vi.fn(),
    get: vi.fn(),
    delete: vi.fn(),
  },
}))

import { api } from '../../shared/api/client'
import { queueApi } from './api'

describe('queueApi.joinQueue', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.mocked(api.post).mockReset()
    vi.mocked(api.get).mockReset()
    vi.mocked(api.post).mockResolvedValue({ data: undefined })
    vi.mocked(api.get).mockResolvedValue({ data: { position: 3 } })
  })

  it('posts to /v1/queue/{itemID}/enqueue without body', async () => {
    const entry = await queueApi.joinQueue('listing-uuid-1')

    expect(api.post).toHaveBeenCalledWith('/v1/queue/listing-uuid-1/enqueue')
    expect(api.get).toHaveBeenCalledWith('/v1/queue/listing-uuid-1/position')
    expect(entry).toEqual({
      id: 'listing-uuid-1-00000000-0000-4000-8000-000000000001',
      productId: 'listing-uuid-1',
      status: 'queued',
      position: 3,
      memberStatus: 'waiting_in_line',
    })
  })
})

describe('queueApi.leaveQueue', () => {
  beforeEach(() => {
    vi.mocked(api.delete).mockReset()
    vi.mocked(api.delete).mockResolvedValue({ data: undefined })
  })

  it('deletes /v1/queue/{itemID}/dequeue without body', async () => {
    await queueApi.leaveQueue('listing-uuid-1')
    expect(api.delete).toHaveBeenCalledWith('/v1/queue/listing-uuid-1/dequeue')
  })
})

describe('queueApi.getItemQueueState', () => {
  beforeEach(() => {
    vi.mocked(api.get).mockReset()
    vi.mocked(api.get).mockResolvedValue({
      data: { state: 'tickets_partially_issued' },
    })
  })

  it('gets /v1/queue/{itemID}/state', async () => {
    await expect(queueApi.getItemQueueState('listing-uuid-1')).resolves.toEqual(
      { state: 'tickets_partially_issued' },
    )
    expect(api.get).toHaveBeenCalledWith('/v1/queue/listing-uuid-1/state')
  })
})
