import { createSlice } from '@reduxjs/toolkit'
import type { PayloadAction } from '@reduxjs/toolkit'
import type { Product } from './types'

interface ProductItemsState {
  productItems: Product[]
}

const initialState = { productItems: [] } satisfies ProductItemsState as ProductItemsState

const ProductItemsSlice = createSlice({
  name: 'productItems',
  initialState,
  reducers: {
    setProductItems(state, action: PayloadAction<Product[]>) {
      state.productItems = action.payload
    },
    upsertProduct(state, action: PayloadAction<Product>) {
      const next = action.payload
      const idx = state.productItems.findIndex((item) => item.id === next.id)
      if (idx >= 0) {
        state.productItems[idx] = next
        return
      }
      state.productItems.push(next)
    },
    patchProductQueueCount(
      state,
      action: PayloadAction<{ id: string; queueCount: number }>,
    ) {
      const item = state.productItems.find((p) => p.id === action.payload.id)
      if (item) {
        item.queueCount = action.payload.queueCount
      }
    },
  },
})

export const { setProductItems, upsertProduct, patchProductQueueCount } =
  ProductItemsSlice.actions
export default ProductItemsSlice.reducer