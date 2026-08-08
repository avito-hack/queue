import { createSlice } from '@reduxjs/toolkit'
import type { PayloadAction } from '@reduxjs/toolkit'
import type { QueueEntry } from './types'

interface QueueItemsState {
  queueItems: QueueEntry[]
}

const initialState = { queueItems: [] } satisfies QueueItemsState as QueueItemsState

const QueueItemsSlice = createSlice({
  name: 'queueItems',
  initialState,
  reducers: {
    joinQueue(state, action: PayloadAction<QueueEntry>) {
      if (state.queueItems.find((item) => item.id === action.payload.id)) {
        return state
      }
      state.queueItems.push(action.payload)
    },
    leaveQueue(state, action: PayloadAction<string>) {
      state.queueItems = state.queueItems.filter(
        (item) => item.id !== action.payload,
      )
    },
    removeQueuedByProductId(state, action: PayloadAction<string>) {
      state.queueItems = state.queueItems.filter(
        (item) =>
          !(item.productId === action.payload && item.status === 'queued'),
      )
    },
    updateQueueItem(state, action: PayloadAction<QueueEntry>) {
      state.queueItems = state.queueItems.map((item) =>
        item.id === action.payload.id ? action.payload : item,
      )
    },
    setQueueItems(state, action: PayloadAction<QueueEntry[]>) {
      state.queueItems = action.payload
    },
  },
})

export const {
  joinQueue,
  leaveQueue,
  removeQueuedByProductId,
  updateQueueItem,
  setQueueItems,
} = QueueItemsSlice.actions
export default QueueItemsSlice.reducer
