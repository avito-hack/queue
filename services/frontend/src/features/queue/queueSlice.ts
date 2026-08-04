// import { createSlice } from '@reduxjs/toolkit'
// import type { PayloadAction } from '@reduxjs/toolkit'
// import type { QueueEntry } from './types'

// interface QueueItemsState {
//   queueItems: QueueEntry[]
// }

// const initialState = { value: 0 } satisfies CounterState as CounterState

// const counterSlice = createSlice({
//   name: 'counter',
//   initialState,
//   reducers: {
//     increment(state) {
//       state.value++
//     },
//     decrement(state) {
//       state.value--
//     },
//     incrementByAmount(state, action: PayloadAction<number>) {
//       state.value += action.payload
//     },
//   },
// })

// export const { increment, decrement, incrementByAmount } = counterSlice.actions
// export default counterSlice.reducer