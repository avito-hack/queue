import { screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { makeProduct } from '../../features/product/testProduct'
import { renderWithProviders } from '../../test/render'
import { Catalog } from './Catalog'

vi.mock('../../features/product/api', () => ({
  productApi: {
    getProducts: vi.fn(),
  },
}))

import { productApi } from '../../features/product/api'

const apiProducts = [
  makeProduct({
    id: '10',
    title: 'Тестовые кроссовки',
    price: 5000,
    image: '👟',
    availableQuantity: 2,
    quantity: 2,
    queueCount: 4,
  }),
]

describe('Catalog integration', () => {
  beforeEach(() => {
    vi.mocked(productApi.getProducts).mockReset()
  })

  it('loads products from API into the page', async () => {
    vi.mocked(productApi.getProducts).mockResolvedValue(apiProducts)

    renderWithProviders(<Catalog />, { route: '/catalog' })

    await waitFor(() => {
      expect(screen.getByText('Тестовые кроссовки')).toBeInTheDocument()
    })
  })

  it('shows empty state when API fails', async () => {
    vi.mocked(productApi.getProducts).mockRejectedValue(new Error('offline'))

    renderWithProviders(<Catalog />, { route: '/catalog' })

    await waitFor(() => {
      expect(
        screen.getByText('Пока нет активных объявлений.'),
      ).toBeInTheDocument()
    })
  })
})
