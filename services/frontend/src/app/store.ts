import { configureStore } from '@reduxjs/toolkit'
import productsReducer from '../features/product/productSlice'
import queueReducer from '../features/queue/queueSlice'
import ticketsReducer from '../features/ticket/ticketSlice'

export const store = configureStore({
  reducer: {
    queue: queueReducer,
    tickets: ticketsReducer,
    products: productsReducer,
  },
})

export type RootState = ReturnType<typeof store.getState>
export type AppDispatch = typeof store.dispatch
