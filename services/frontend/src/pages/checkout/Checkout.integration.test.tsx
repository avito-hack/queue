import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../../test/render'
import { Checkout } from './Checkout'

vi.mock('../../features/ticket/api', () => ({
  ticketApi: {
    listTickets: vi.fn().mockResolvedValue({ ticket: [] }),
    payOrder: vi.fn(),
    declineTicket: vi.fn(),
  },
}))

import { ticketApi } from '../../features/ticket/api'

const checkoutState = {
  tickets: {
    ticketItems: [
      {
        id: 't-1',
        productId: 'p-1',
        expiresAt: '2026-08-05T12:10:00.000Z',
      },
    ],
  },
  products: {
    productItems: [
      {
        id: 'p-1',
        name: 'Куртка для оплаты',
        description: 'mock',
        price: 12800,
        image: '🧥',
        count: 1,
        queueCount: 0,
      },
    ],
  },
}

describe('Checkout integration', () => {
  beforeEach(() => {
    vi.mocked(ticketApi.payOrder).mockReset()
  })

  it('shows empty state without a valid ticket', () => {
    renderWithProviders(<Checkout />, { route: '/checkout' })

    expect(screen.getByText('Нет права на покупку')).toBeInTheDocument()
  })

  it('pays order via API and removes ticket from store', async () => {
    const user = userEvent.setup()
    vi.mocked(ticketApi.payOrder).mockResolvedValue({ ok: true })

    const { store } = renderWithProviders(<Checkout />, {
      route: '/checkout?ticket=t-1',
      preloadedState: checkoutState,
    })

    expect(screen.getByText('Куртка для оплаты')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Оплатить заказ' }))

    await waitFor(() => {
      expect(ticketApi.payOrder).toHaveBeenCalledWith('t-1')
    })
    expect(store.getState().tickets.ticketItems).toEqual([])
    expect(screen.getByText('Заказ оформлен')).toBeInTheDocument()
  })

  it('completes demo pay when API fails', async () => {
    const user = userEvent.setup()
    vi.mocked(ticketApi.payOrder).mockRejectedValue(new Error('offline'))

    const { store } = renderWithProviders(<Checkout />, {
      route: '/checkout?ticket=t-1',
      preloadedState: checkoutState,
    })

    await user.click(screen.getByRole('button', { name: 'Оплатить заказ' }))

    await waitFor(() => {
      expect(ticketApi.payOrder).toHaveBeenCalledWith('t-1')
    })
    expect(store.getState().tickets.ticketItems).toEqual([])
    expect(screen.getByText('Заказ оформлен')).toBeInTheDocument()
  })
})
