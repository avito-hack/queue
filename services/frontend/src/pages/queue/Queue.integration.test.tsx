import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '../../test/render'
import { Queue } from './Queue'

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
})
