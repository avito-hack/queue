import { screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '../../test/render'
import { Catalog } from './Catalog'

vi.mock('../../features/product/api', () => ({
  productApi: {
    getProducts: vi.fn(),
  },
}))

import { productApi } from '../../features/product/api'

const apiProducts = [
  {
    id: '10',
    name: 'Тестовые кроссовки',
    description: 'mock',
    price: 5000,
    image: '👟',
    count: 2,
    queueCount: 4,
  },
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
    expect(screen.getByText('1 объявлений')).toBeInTheDocument()
  })

  it('falls back to mock products when API fails', async () => {
    vi.mocked(productApi.getProducts).mockRejectedValue(new Error('offline'))

    renderWithProviders(<Catalog />, { route: '/catalog' })

    await waitFor(() => {
      expect(
        screen.getByText('Кроссовки Northline Drop 01'),
      ).toBeInTheDocument()
    })
  })
})
