import { configureStore } from '@reduxjs/toolkit'
import queueReducer from '../features/queue/queueSlice'
import productsReducer from '../features/product/productSlice'

export const store = configureStore({
  reducer: {
    queue: queueReducer,
    products: productsReducer,
  },
})

export type RootState = ReturnType<typeof store.getState>
export type AppDispatch = typeof store.dispatch
