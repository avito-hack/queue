import { useEffect } from 'react'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import type { Product } from '../../features/product/types'
import { CatalogCard } from './CatalogCard'
import { productApi } from '../../features/product/api'
import { setProductItems } from '../../features/product/productSlice'
import { reportApiError } from '../../shared/api/errors'

const DEMO_SELLER = '00000000-0000-4000-8000-000000000010'
const now = '2026-08-06T12:00:00.000Z'

/** Демо-каталог в форме Listing (+ UI image/description/queueCount). */
const mockProducts: Product[] = [
  {
    id: '11111111-1111-4111-8111-111111111101',
    sellerId: DEMO_SELLER,
    title: 'Кроссовки Northline Drop 01',
    description: 'Лимитированная коллекция',
    price: 19990,
    image: '👟',
    quantity: 3,
    reservedQuantity: 0,
    availableQuantity: 3,
    queueEnabled: true,
    status: 'active',
    queueCount: 27,
    createdAt: now,
    updatedAt: now,
  },
  {
    id: '11111111-1111-4111-8111-111111111102',
    sellerId: DEMO_SELLER,
    title: 'Куртка Northline Shell',
    description: 'Лимитированная коллекция',
    price: 12800,
    image: '🧥',
    quantity: 1,
    reservedQuantity: 0,
    availableQuantity: 1,
    queueEnabled: true,
    status: 'active',
    queueCount: 14,
    createdAt: now,
    updatedAt: now,
  },
  {
    id: '11111111-1111-4111-8111-111111111103',
    sellerId: DEMO_SELLER,
    title: 'Кепка Drop 01',
    description: 'Лимитированная коллекция',
    price: 3900,
    image: '🧢',
    quantity: 0,
    reservedQuantity: 0,
    availableQuantity: 0,
    queueEnabled: true,
    status: 'active',
    queueCount: 5,
    createdAt: now,
    updatedAt: now,
  },
  {
    id: '11111111-1111-4111-8111-111111111104',
    sellerId: DEMO_SELLER,
    title: 'Рюкзак Northline City',
    description: 'Лимитированная коллекция',
    price: 6700,
    image: '🎒',
    quantity: 5,
    reservedQuantity: 0,
    availableQuantity: 5,
    queueEnabled: true,
    status: 'active',
    queueCount: 2,
    createdAt: now,
    updatedAt: now,
  },
  {
    id: '11111111-1111-4111-8111-111111111105',
    sellerId: DEMO_SELLER,
    title: 'Кроссовки Northline Base',
    description: 'Лимитированная коллекция',
    price: 14500,
    image: '👟',
    quantity: 2,
    reservedQuantity: 0,
    availableQuantity: 2,
    queueEnabled: true,
    status: 'active',
    queueCount: 9,
    createdAt: now,
    updatedAt: now,
  },
  {
    id: '11111111-1111-4111-8111-111111111106',
    sellerId: DEMO_SELLER,
    title: 'Худи Northline Soft',
    description: 'Лимитированная коллекция',
    price: 8900,
    image: '👕',
    quantity: 4,
    reservedQuantity: 0,
    availableQuantity: 4,
    queueEnabled: true,
    status: 'active',
    queueCount: 3,
    createdAt: now,
    updatedAt: now,
  },
]

export function Catalog() {
  const dispatch = useAppDispatch()
  useEffect(() => {
    void (async () => {
      try {
        const products = await productApi.getProducts()
        dispatch(setProductItems(products))
      } catch (e) {
        reportApiError(e, 'Не удалось загрузить каталог')
        dispatch(setProductItems(mockProducts))
      }
    })()
  }, [dispatch])

  const products = useAppSelector((state) => state.products.productItems)

  return (
    <section>
      <div className="mb-[22px]">
        <div className="mb-2 text-sm text-avito-muted">
          Главная › Лимитированные товары
        </div>
        <h1 className="m-0 text-2xl tracking-tight sm:text-[30px]">
          Лимитированные товары
        </h1>
        <p className="mt-2 max-w-[640px] leading-normal text-avito-muted">
          Очередь даёт право на покупку. Место в очереди не гарантирует покупку —
          товар может закончиться раньше.
        </p>
      </div>

      {products.length === 0 ? (
        <div className="rounded-2xl bg-white p-8 text-center text-avito-muted">
          Пока нет активных объявлений.
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {products.map((product) => (
            <CatalogCard key={product.id} product={product} />
          ))}
        </div>
      )}
    </section>
  )
}
