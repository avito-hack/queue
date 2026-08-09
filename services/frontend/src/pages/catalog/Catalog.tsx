import { useEffect } from 'react'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { CatalogCard } from './CatalogCard'
import { productApi } from '../../features/product/api'
import { setProductItems } from '../../features/product/productSlice'
import { reportApiError } from '../../shared/api/errors'

export function Catalog() {
  const dispatch = useAppDispatch()
  useEffect(() => {
    void (async () => {
      try {
        const products = await productApi.getProducts()
        dispatch(setProductItems(products))
      } catch (e) {
        reportApiError(e, 'Не удалось загрузить каталог')
        dispatch(setProductItems([]))
      }
    })()
  }, [dispatch])

  const products = useAppSelector((state) => state.products.productItems)

  return (
    <section>
      <div className="mb-5 sm:mb-6">
        <div className="mb-2 text-[13px] text-avito-muted sm:text-sm">
          Главная › Лимитированные товары
        </div>
        <h1 className="m-0 text-[26px] font-extrabold tracking-tight sm:text-[32px]">
          Лимитированные товары
        </h1>
        <p className="mt-2 max-w-[640px] text-[14px] leading-normal text-avito-muted sm:text-[15px]">
          Очередь даёт право на покупку. Место в очереди не гарантирует покупку —
          товар может закончиться раньше.
        </p>
      </div>

      {products.length === 0 ? (
        <div className="rounded-2xl bg-white px-5 py-10 text-center text-avito-muted">
          Пока нет активных объявлений.
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 min-[480px]:grid-cols-2 lg:grid-cols-3 lg:gap-4">
          {products.map((product) => (
            <CatalogCard key={product.id} product={product} />
          ))}
        </div>
      )}
    </section>
  )
}
