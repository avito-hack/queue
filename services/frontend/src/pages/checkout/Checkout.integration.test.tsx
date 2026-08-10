import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it } from 'vitest'
import { makeProduct } from '../../features/product/testProduct'
import { isTicketPaid } from '../../features/ticket/paidTickets'
import { renderWithProviders } from '../../test/render'
import { Checkout } from './Checkout'

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
      makeProduct({
        id: 'p-1',
        title: 'Куртка для оплаты',
        availableQuantity: 1,
        quantity: 1,
        queueCount: 0,
      }),
    ],
  },
}

describe('Checkout integration', () => {
  beforeEach(() => {
    localStorage.clear()
    localStorage.setItem('userId', 'user-checkout')
  })

  it('shows empty state without a valid ticket', () => {
    renderWithProviders(<Checkout />, { route: '/checkout' })

    expect(screen.getByText('Нет права на покупку')).toBeInTheDocument()
  })

  it('completes demo pay locally without calling a pay API', async () => {
    const user = userEvent.setup()

    const { store } = renderWithProviders(<Checkout />, {
      route: '/checkout?ticket=t-1',
      preloadedState: checkoutState,
    })

    expect(screen.getByText('Куртка для оплаты')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Оплатить заказ' }))

    expect(store.getState().tickets.ticketItems).toEqual([])
    expect(isTicketPaid('t-1')).toBe(true)
    expect(screen.getByText('Заказ оформлен')).toBeInTheDocument()
  })
})
