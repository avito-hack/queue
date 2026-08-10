import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { makeProduct } from '../../features/product/testProduct'
import { renderWithProviders } from '../../test/render'
import { Product } from './Product'

vi.mock('../../features/queue/api', () => ({
  queueApi: {
    joinQueue: vi.fn(),
    leaveQueue: vi.fn(),
    getItemQueueState: vi.fn().mockResolvedValue({
      state: 'tickets_available',
      waiting_count: 0,
    }),
    listUserQueues: vi.fn().mockResolvedValue([]),
    getPosition: vi.fn(),
  },
}))

vi.mock('../../features/product/api', () => ({
  productApi: {
    getProducts: vi.fn(),
    getListing: vi.fn().mockRejectedValue(new Error('not found')),
  },
}))

import { queueApi } from '../../features/queue/api'

const product = makeProduct({ id: 'p-1' })

function renderProductPage() {
  return renderWithProviders(
    <Routes>
      <Route path="/product/:id" element={<Product />} />
    </Routes>,
    {
      route: '/product/p-1',
      preloadedState: {
        products: { productItems: [product] },
      },
    },
  )
}

describe('Product integration', () => {
  beforeEach(() => {
    vi.mocked(queueApi.joinQueue).mockReset()
  })

  it('shows not found when product is missing from store', async () => {
    renderWithProviders(
      <Routes>
        <Route path="/product/:id" element={<Product />} />
      </Routes>,
      { route: '/product/missing' },
    )

    expect(await screen.findByText(/Товар не найден/i)).toBeInTheDocument()
  })

  it('joins queue via API and opens success modal', async () => {
    const user = userEvent.setup()
    vi.mocked(queueApi.joinQueue).mockResolvedValue({
      id: 'entry-1',
      productId: 'p-1',
      status: 'queued',
      position: 4,
    })

    const { store } = renderProductPage()

    await user.click(screen.getByRole('button', { name: 'Встать в очередь' }))

    await waitFor(() => {
      expect(queueApi.joinQueue).toHaveBeenCalledWith('p-1')
    })
    expect(store.getState().queue.queueItems).toEqual([
      {
        id: 'entry-1',
        productId: 'p-1',
        status: 'queued',
        position: 4,
      },
    ])
    expect(
      screen.getByRole('dialog', { name: 'Вы в очереди' }),
    ).toBeInTheDocument()
  })

  it('does not join queue when API fails', async () => {
    const user = userEvent.setup()
    vi.mocked(queueApi.joinQueue).mockRejectedValue(new Error('offline'))

    const { store } = renderProductPage()

    await user.click(screen.getByRole('button', { name: 'Встать в очередь' }))

    await waitFor(() => {
      expect(queueApi.joinQueue).toHaveBeenCalledWith('p-1')
    })
    expect(store.getState().queue.queueItems).toEqual([])
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('notifies about restock when product is sold out', async () => {
    const user = userEvent.setup()
    localStorage.clear()

    renderWithProviders(
      <Routes>
        <Route path="/product/:id" element={<Product />} />
      </Routes>,
      {
        route: '/product/p-1',
        preloadedState: {
          products: {
            productItems: [{ ...product, availableQuantity: 0, quantity: 0 }],
          },
        },
      },
    )

    await user.click(
      screen.getByRole('button', { name: 'Уведомить о поступлении' }),
    )

    expect(localStorage.getItem('notify:p-1')).toBe('1')
    expect(
      screen.getByRole('button', { name: 'Подписка оформлена' }),
    ).toBeDisabled()
  })

  it('marks queue as sold out and opens modal when stock hits zero', async () => {
    renderWithProviders(
      <Routes>
        <Route path="/product/:id" element={<Product />} />
      </Routes>,
      {
        route: '/product/p-1',
        preloadedState: {
          products: {
            productItems: [{ ...product, availableQuantity: 0, quantity: 0 }],
          },
          queue: {
            queueItems: [
              {
                id: 'e-queued',
                productId: 'p-1',
                status: 'queued',
                position: 2,
              },
            ],
          },
        },
      },
    )

    expect(
      await screen.findByRole('dialog', { name: 'Упс, товар закончился' }),
    ).toBeInTheDocument()
  })
})
