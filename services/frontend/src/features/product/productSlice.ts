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
  },
})

export const { setProductItems } = ProductItemsSlice.actions
export default ProductItemsSlice.reducer