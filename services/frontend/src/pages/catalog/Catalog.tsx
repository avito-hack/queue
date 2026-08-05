import { useEffect } from 'react'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import type { Product } from '../../features/product/types'
import { CatalogCard } from './CatalogCard'
import { productApi } from '../../features/product/api'
import { setProductItems } from '../../features/product/productSlice'

const mockProducts: Product[] = [
  {
    id: '1',
    name: 'Кроссовки Northline Drop 01',
    description: 'Лимитированная коллекция',
    price: 19990,
    image: '👟',
    count: 3,
    queueCount: 27,
  },
  {
    id: '2',
    name: 'Куртка Northline Shell',
    description: 'Лимитированная коллекция',
    price: 12800,
    image: '🧥',
    count: 1,
    queueCount: 14,
  },
  {
    id: '3',
    name: 'Кепка Drop 01',
    description: 'Лимитированная коллекция',
    price: 3900,
    image: '🧢',
    count: 0,
    queueCount: 5,
  },
  {
    id: '4',
    name: 'Рюкзак Northline City',
    description: 'Лимитированная коллекция',
    price: 6700,
    image: '🎒',
    count: 5,
    queueCount: 2,
  },
  {
    id: '5',
    name: 'Кроссовки Northline Base',
    description: 'Лимитированная коллекция',
    price: 14500,
    image: '👟',
    count: 2,
    queueCount: 9,
  },
  {
    id: '6',
    name: 'Худи Northline Soft',
    description: 'Лимитированная коллекция',
    price: 8900,
    image: '👕',
    count: 4,
    queueCount: 3,
  },
]

export function Catalog() {
  const dispatch = useAppDispatch()
  useEffect(() => {
    (async () => {
      try {
        const products = await productApi.getProducts()
        dispatch(setProductItems(products as Product[]))
      } catch (e) {
        console.error(e)
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
          Редкие позиции с очередью: сначала место, затем временное право на
          покупку. Выберите товар, чтобы встать в очередь.
        </p>
      </div>

      <div className="mb-5 flex flex-wrap items-center gap-2">
        <span className="rounded-full bg-avito-ink px-3 py-1.5 text-xs font-extrabold text-white">
          С очередью
        </span>
        <span className="rounded-full bg-[#f1f1f1] px-3 py-1.5 text-xs font-bold text-[#4d4d4d]">
          {products.length} объявлений
        </span>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {products.map((product) => (
          <CatalogCard key={product.id} product={product} />
        ))}
      </div>
    </section>
  )
}
