import { Link } from 'react-router-dom'
import type { Product } from '../../features/product/types'

function formatPrice(price: number) {
  return `${price.toLocaleString('ru-RU')} ₽`
}

export function CatalogCard({ product }: { product: Product }) {
  const inStock = product.availableQuantity > 0
  const queueCount = product.queueCount ?? 0

  return (
    <Link
      to={`/product/${product.id}`}
      className="group flex flex-col overflow-hidden rounded-2xl bg-white no-underline transition duration-150 hover:shadow-card"
    >
      <div className="relative grid aspect-[4/3] place-items-center bg-[#f5f5f5] text-[72px] sm:text-[80px]">
        <span className="absolute top-3 left-3 rounded-lg bg-avito-ink px-2 py-1 text-[11px] font-extrabold tracking-wide text-white uppercase">
          Лимит
        </span>
        <span className="select-none" aria-hidden="true">
          {product.image ?? '🛒'}
        </span>
      </div>

      <div className="flex flex-1 flex-col p-3.5 sm:p-4">
        <div className="text-[20px] font-extrabold tracking-tight text-avito-ink sm:text-[22px]">
          {formatPrice(product.price)}
        </div>
        <div className="mt-1 line-clamp-2 text-[15px] leading-snug font-semibold text-[#1a1a1a] group-hover:text-avito-blue">
          {product.title}
        </div>

        <div className="mt-3 grid gap-1 text-[13px]">
          <div className="flex justify-between gap-3">
            <span className="text-avito-muted">В наличии</span>
            <strong className={inStock ? 'text-avito-green' : 'text-avito-red'}>
              {inStock ? `${product.availableQuantity} шт.` : 'нет'}
            </strong>
          </div>
          <div className="flex justify-between gap-3">
            <span className="text-avito-muted">В очереди</span>
            <strong className="text-avito-ink">{queueCount}</strong>
          </div>
        </div>

        {!inStock && (
          <div className="mt-3 rounded-xl bg-[#fff0f2] px-3 py-2 text-[12px] font-bold text-[#b82334]">
            Товар закончился
          </div>
        )}

        <div className="mt-auto pt-3.5 text-[13px] font-extrabold text-avito-blue">
          Смотреть объявление
        </div>
      </div>
    </Link>
  )
}
