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
    vi.mocked(api.post).mockResolvedValue({ data: { status: 'enqueued' } })
    vi.mocked(api.get).mockResolvedValue({ data: { position: 3 } })
  })

  it('posts EnqueueRequest { id, user } to /v1/queue/enqueue', async () => {
    const entry = await queueApi.joinQueue('listing-uuid-1')

    expect(api.post).toHaveBeenCalledWith('/v1/queue/enqueue', {
      id: 'listing-uuid-1',
      user: {
        id: '00000000-0000-4000-8000-000000000001',
        first_name: 'Demo',
        last_name: 'User',
        has_ticket: false,
      },
    })
    expect(api.get).toHaveBeenCalledWith('/v1/queue/listing-uuid-1/position')
    expect(entry).toEqual({
      id: 'listing-uuid-1-00000000-0000-4000-8000-000000000001',
      productId: 'listing-uuid-1',
      status: 'queued',
      position: 3,
    })
  })
})
