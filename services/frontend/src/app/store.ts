import { configureStore } from '@reduxjs/toolkit'
import queueReducer from '../features/queue/queueSlice'

export const store = configureStore({
  reducer: {
    queue: queueReducer,
  },
})

export type RootState = ReturnType<typeof store.getState>
export type AppDispatch = typeof store.dispatch
