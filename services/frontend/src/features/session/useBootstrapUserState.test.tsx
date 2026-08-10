import { renderHook, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { Provider } from 'react-redux'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createTestStore } from '../../test/render'
import { useBootstrapUserState } from './useBootstrapUserState'

vi.mock('../queue/api', () => ({
  queueApi: {
    listUserQueues: vi.fn(),
  },
}))

vi.mock('../ticket/api', () => ({
  ticketApi: {
    listTickets: vi.fn(),
  },
}))

vi.mock('../product/api', () => ({
  productApi: {
    getListing: vi.fn(),
  },
}))

import { productApi } from '../product/api'
import { queueApi } from '../queue/api'
import { ticketApi } from '../ticket/api'

describe('useBootstrapUserState', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.mocked(queueApi.listUserQueues).mockReset()
    vi.mocked(ticketApi.listTickets).mockReset()
    vi.mocked(productApi.getListing).mockReset()
  })

  it('hydrates queues, tickets and related products into the store on mount', async () => {
    vi.mocked(queueApi.listUserQueues).mockResolvedValue([
      {
        item_id: 'listing-a',
        position: 2,
        status: 'waiting_in_line',
      },
    ])
    vi.mocked(ticketApi.listTickets).mockResolvedValue({
      ticket: [
        {
          id: 't-1',
          listing_id: 'listing-b',
          status: 'issued',
          activation_deadline: '2099-01-01T00:00:00.000Z',
          available_actions: ['activate', 'decline'],
        },
      ],
    })
    vi.mocked(productApi.getListing).mockImplementation(async (id) => ({
      id,
      title: id === 'listing-a' ? 'Кепка Drop 01' : 'Худи Soft',
      price: 1000,
      quantity: 1,
      availableQuantity: 1,
      reservedQuantity: 0,
      sellerId: 'seller',
      queueEnabled: true,
      status: 'active',
      createdAt: '2026-01-01T00:00:00.000Z',
      updatedAt: '2026-01-01T00:00:00.000Z',
      image: '🧢',
    }))

    const store = createTestStore()
    const wrapper = ({ children }: { children: ReactNode }) => (
      <Provider store={store}>{children}</Provider>
    )

    renderHook(() => useBootstrapUserState(), { wrapper })

    await waitFor(() => {
      expect(store.getState().queue.queueItems).toHaveLength(1)
      expect(store.getState().tickets.ticketItems).toHaveLength(1)
      expect(store.getState().products.productItems).toHaveLength(2)
    })

    expect(store.getState().queue.queueItems[0]?.productId).toBe('listing-a')
    expect(store.getState().tickets.ticketItems[0]?.id).toBe('t-1')
    expect(store.getState().products.productItems.map((p) => p.title)).toEqual(
      expect.arrayContaining(['Кепка Drop 01', 'Худи Soft']),
    )
    expect(productApi.getListing).toHaveBeenCalledWith('listing-a')
    expect(productApi.getListing).toHaveBeenCalledWith('listing-b')
  })
})
