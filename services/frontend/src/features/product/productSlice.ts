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
  },
})

export const { setProductItems, upsertProduct } = ProductItemsSlice.actions
export default ProductItemsSlice.reducer