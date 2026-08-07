import { createSlice } from '@reduxjs/toolkit'
import type { PayloadAction } from '@reduxjs/toolkit'
import type { TicketEntry } from './types'

interface TicketItemsState {
  ticketItems: TicketEntry[]
}

const initialState = {
  ticketItems: [],
} satisfies TicketItemsState as TicketItemsState

const TicketItemsSlice = createSlice({
  name: 'ticketItems',
  initialState,
  reducers: {
    upsertTicket(state, action: PayloadAction<TicketEntry>) {
      const ticket = action.payload
      const idx = state.ticketItems.findIndex((item) => item.id === ticket.id)
      if (idx >= 0) {
        state.ticketItems[idx] = ticket
        return
      }
      const byProduct = state.ticketItems.findIndex(
        (item) => item.productId === ticket.productId,
      )
      if (byProduct >= 0) {
        state.ticketItems[byProduct] = ticket
        return
      }
      state.ticketItems.push(ticket)
    },
    removeTicket(state, action: PayloadAction<string>) {
      state.ticketItems = state.ticketItems.filter(
        (item) => item.id !== action.payload,
      )
    },
  },
})

export const { upsertTicket, removeTicket } = TicketItemsSlice.actions
export default TicketItemsSlice.reducer
