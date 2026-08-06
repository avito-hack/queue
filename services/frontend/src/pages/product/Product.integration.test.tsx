import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../../test/render'
import { Product } from './Product'

vi.mock('../../features/queue/api', () => ({
  queueApi: {
    joinQueue: vi.fn(),
    leaveQueue: vi.fn(),
  },
}))

import { queueApi } from '../../features/queue/api'

const product = {
  id: 'p-1',
  name: 'Куртка Northline Shell',
  description: 'Лимитированная коллекция',
  price: 12800,
  image: '🧥',
  count: 2,
  queueCount: 5,
}

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

  it('shows not found when product is missing from store', () => {
    renderWithProviders(
      <Routes>
        <Route path="/product/:id" element={<Product />} />
      </Routes>,
      { route: '/product/missing' },
    )

    expect(screen.getByText(/Товар не найден/i)).toBeInTheDocument()
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

  it('uses local fallback entry when join API fails', async () => {
    const user = userEvent.setup()
    vi.mocked(queueApi.joinQueue).mockRejectedValue(new Error('offline'))

    const { store } = renderProductPage()

    await user.click(screen.getByRole('button', { name: 'Встать в очередь' }))

    await waitFor(() => {
      expect(store.getState().queue.queueItems).toEqual([
        {
          id: 'p-1-entry',
          productId: 'p-1',
          status: 'queued',
          position: 8,
        },
      ])
    })
    expect(screen.getByRole('dialog')).toBeInTheDocument()
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
            productItems: [{ ...product, count: 0 }],
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
            productItems: [{ ...product, count: 0 }],
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
