import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Route, Routes } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { makeProduct } from '../../features/product/testProduct'
import { renderWithProviders } from '../../test/render'
import { Checkout } from '../checkout/Checkout'
import { Queue } from './Queue'

vi.mock('../../features/ticket/api', () => ({
  ticketApi: {
    listTickets: vi.fn().mockResolvedValue({ ticket: [] }),
    activateTicket: vi.fn(),
    declineTicket: vi.fn(),
  },
}))

vi.mock('../../features/queue/api', () => ({
  queueApi: {
    joinQueue: vi.fn(),
    leaveQueue: vi.fn(),
    getPosition: vi.fn(),
  },
}))

import { queueApi } from '../../features/queue/api'
import { ticketApi } from '../../features/ticket/api'

const product = makeProduct({
  id: 'p-1',
  title: 'Кроссовки Northline Drop 01',
  price: 19990,
  image: '👟',
  availableQuantity: 3,
  quantity: 3,
  queueCount: 10,
})

describe('Queue integration', () => {
  beforeEach(() => {
    vi.mocked(ticketApi.activateTicket).mockReset()
    vi.mocked(ticketApi.declineTicket).mockReset()
    vi.mocked(queueApi.leaveQueue).mockReset()
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
              id: 'e-done',
              productId: 'p-1',
              status: 'soldout',
            },
          ],
        },
        tickets: {
          ticketItems: [
            {
              id: 'e-ticket',
              productId: 'p-1',
              expiresAt: '2099-01-01T00:00:00.000Z',
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

  it('confirms leave, calls dequeue API and removes tile', async () => {
    const user = userEvent.setup()
    vi.mocked(queueApi.leaveQueue).mockResolvedValue(undefined)

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

    const dialog = screen.getByRole('dialog', { name: 'Выйти из очереди?' })
    expect(dialog).toBeInTheDocument()

    await user.click(
      within(dialog).getByRole('button', { name: 'Да, выйти из очереди' }),
    )

    await waitFor(() => {
      expect(queueApi.leaveQueue).toHaveBeenCalledWith('p-1')
    })
    expect(store.getState().queue.queueItems).toEqual([])
    expect(
      screen.getByText(/Вы ещё не вставали в очередь/i),
    ).toBeInTheDocument()
  })

  it('activates ticket and follows checkout_url', async () => {
    const user = userEvent.setup()
    vi.mocked(ticketApi.activateTicket).mockResolvedValue({
      ticket_id: 'e-ticket',
      status: 'activated',
      order_id: 'order-1',
      checkout_url: '/checkout?ticket=e-ticket',
    })

    renderWithProviders(
      <Routes>
        <Route path="/queue" element={<Queue />} />
        <Route path="/checkout" element={<Checkout />} />
      </Routes>,
      {
        route: '/queue',
        preloadedState: {
          products: { productItems: [product] },
          tickets: {
            ticketItems: [
              {
                id: 'e-ticket',
                productId: 'p-1',
                expiresAt: '2099-01-01T00:00:00.000Z',
              },
            ],
          },
        },
      },
    )

    await user.click(screen.getByRole('button', { name: 'Перейти к покупке' }))

    const dialog = screen.getByRole('dialog', { name: 'Товар доступен для вас' })
    await user.click(
      within(dialog).getByRole('button', { name: 'Перейти к покупке' }),
    )

    await waitFor(() => {
      expect(ticketApi.activateTicket).toHaveBeenCalledWith('e-ticket')
    })
    expect(screen.getByText('Оформление заказа')).toBeInTheDocument()
    expect(screen.getByText(/Тикет: e-ticket/)).toBeInTheDocument()
    expect(screen.queryByText('Перейти к покупке')).not.toBeInTheDocument()
  })

  it('keeps ticket modal open when activate fails', async () => {
    const user = userEvent.setup()
    vi.mocked(ticketApi.activateTicket).mockRejectedValue(new Error('offline'))

    renderWithProviders(
      <Routes>
        <Route path="/queue" element={<Queue />} />
        <Route path="/checkout" element={<Checkout />} />
      </Routes>,
      {
        route: '/queue',
        preloadedState: {
          products: { productItems: [product] },
          tickets: {
            ticketItems: [
              {
                id: 'e-ticket',
                productId: 'p-1',
                expiresAt: '2099-01-01T00:00:00.000Z',
              },
            ],
          },
        },
      },
    )

    await user.click(screen.getByRole('button', { name: 'Перейти к покупке' }))
    const dialog = screen.getByRole('dialog', { name: 'Товар доступен для вас' })
    await user.click(
      within(dialog).getByRole('button', { name: 'Перейти к покупке' }),
    )

    await waitFor(() => {
      expect(ticketApi.activateTicket).toHaveBeenCalledWith('e-ticket')
    })
    expect(
      screen.getByRole('dialog', { name: 'Товар доступен для вас' }),
    ).toBeInTheDocument()
    expect(screen.queryByText('Оформление заказа')).not.toBeInTheDocument()
  })

  it('declines ticket via API and removes it from store', async () => {
    const user = userEvent.setup()
    vi.mocked(ticketApi.declineTicket).mockResolvedValue({ ok: true })

    const { store } = renderWithProviders(<Queue />, {
      route: '/queue',
      preloadedState: {
        products: { productItems: [product] },
        tickets: {
          ticketItems: [
            {
              id: 'e-ticket',
              productId: 'p-1',
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
    expect(store.getState().tickets.ticketItems).toEqual([])
    expect(
      screen.getByText(/Вы ещё не вставали в очередь/i),
    ).toBeInTheDocument()
  })
})
