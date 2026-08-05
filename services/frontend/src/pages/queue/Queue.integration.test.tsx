import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../../test/render'
import { Queue } from './Queue'

vi.mock('../../features/ticket/api', () => ({
  ticketApi: {
    payOrder: vi.fn(),
    declineTicket: vi.fn(),
  },
}))

import { ticketApi } from '../../features/ticket/api'

const product = {
  id: 'p-1',
  name: 'Кроссовки Northline Drop 01',
  description: 'mock',
  price: 19990,
  image: '👟',
  count: 3,
  queueCount: 10,
}

describe('Queue integration', () => {
  beforeEach(() => {
    vi.mocked(ticketApi.declineTicket).mockReset()
  })

  it('shows empty state when user has no queues', () => {
    renderWithProviders(<Queue />, { route: '/queue' })

    expect(
      screen.getByText(/Вы ещё не вставали в очередь/i),
    ).toBeInTheDocument()
  })

  it('renders tiles and summary counts from store', () => {
    renderWithProviders(<Queue />, {
      route: '/queue',
      preloadedState: {
        products: { productItems: [product] },
        queue: {
          queueItems: [
            {
              id: 'e-queued',
              productId: 'p-1',
              status: 'queued',
              position: 3,
            },
            {
              id: 'e-ticket',
              productId: 'p-1',
              status: 'ticket',
              expiresAt: '2099-01-01T00:00:00.000Z',
            },
            {
              id: 'e-done',
              productId: 'p-1',
              status: 'soldout',
            },
          ],
        },
      },
    })

    expect(screen.getByText('Активных очередей').parentElement).toHaveTextContent(
      '1',
    )
    expect(screen.getByText('Доступно к покупке').parentElement).toHaveTextContent(
      '1',
    )
    expect(screen.getByText('Завершено').parentElement).toHaveTextContent('1')
    expect(screen.getByText(/В очереди · место 3/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Перейти к покупке' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Выйти из очереди' })).toBeInTheDocument()
  })

  it('leaves queue and removes tile from store', async () => {
    const user = userEvent.setup()
    const { store } = renderWithProviders(<Queue />, {
      route: '/queue',
      preloadedState: {
        products: { productItems: [product] },
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
    })

    await user.click(screen.getByRole('button', { name: 'Выйти из очереди' }))

    expect(store.getState().queue.queueItems).toEqual([])
    expect(
      screen.getByText(/Вы ещё не вставали в очередь/i),
    ).toBeInTheDocument()
  })

  it('declines ticket via API and removes it from store', async () => {
    const user = userEvent.setup()
    vi.mocked(ticketApi.declineTicket).mockResolvedValue({ ok: true })

    const { store } = renderWithProviders(<Queue />, {
      route: '/queue',
      preloadedState: {
        products: { productItems: [product] },
        queue: {
          queueItems: [
            {
              id: 'e-ticket',
              productId: 'p-1',
              status: 'ticket',
              expiresAt: '2099-01-01T00:00:00.000Z',
            },
          ],
        },
      },
    })

    await user.click(screen.getByRole('button', { name: 'Перейти к покупке' }))
    await user.click(screen.getByRole('button', { name: 'Отказаться от покупки' }))

    await waitFor(() => {
      expect(ticketApi.declineTicket).toHaveBeenCalledWith('e-ticket')
    })
    expect(store.getState().queue.queueItems).toEqual([])
    expect(
      screen.getByText(/Вы ещё не вставали в очередь/i),
    ).toBeInTheDocument()
  })
})
