import { configureStore } from '@reduxjs/toolkit'
import { render } from '@testing-library/react'
import type { ReactElement, ReactNode } from 'react'
import { Provider } from 'react-redux'
import { MemoryRouter } from 'react-router-dom'
import type { RootState } from '../app/store'
import productsReducer from '../features/product/productSlice'
import queueReducer from '../features/queue/queueSlice'
import ticketsReducer from '../features/ticket/ticketSlice'

type RenderOptions = {
  route?: string
  preloadedState?: Partial<RootState>
}

export function createTestStore(preloadedState?: Partial<RootState>) {
  return configureStore({
    reducer: {
      queue: queueReducer,
      tickets: ticketsReducer,
      products: productsReducer,
    },
    preloadedState: preloadedState as RootState | undefined,
  })
}

export function renderWithProviders(
  ui: ReactElement,
  { route = '/', preloadedState }: RenderOptions = {},
) {
  const store = createTestStore(preloadedState)

  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <Provider store={store}>
        <MemoryRouter initialEntries={[route]}>{children}</MemoryRouter>
      </Provider>
    )
  }

  return {
    store,
    ...render(ui, { wrapper: Wrapper }),
  }
}
