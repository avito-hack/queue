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
      className="group flex flex-col overflow-hidden rounded-[20px] border border-transparent bg-white no-underline transition duration-150 hover:-translate-y-0.5 hover:border-[#c9c9c9] hover:shadow-card"
    >
      <div className="relative grid h-[200px] place-items-center bg-linear-to-br from-[#dff5ff] to-[#e7dcff] text-[88px]">
        <span className="absolute top-3.5 left-3.5 rounded-full bg-avito-ink px-2.5 py-1.5 text-[12px] font-extrabold text-white">
          Лимит
        </span>
        {product.image ?? '🛒'}
      </div>

      <div className="flex flex-1 flex-col p-4">
        <div className="text-[22px] font-extrabold tracking-tight text-avito-ink">
          {formatPrice(product.price)}
        </div>
        <div className="mt-1.5 text-[15px] leading-snug font-bold text-[#2a2a2a] group-hover:text-[#008ed8]">
          {product.title}
        </div>

        <div className="mt-3 grid gap-1.5 text-sm">
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
          <div className="mt-3 rounded-[12px] bg-[#fff0f2] px-3 py-2 text-[12px] font-bold text-[#b82334]">
            Товар закончился
          </div>
        )}

        <div className="mt-auto pt-4 text-sm font-extrabold text-[#008ed8]">
          Открыть объявление →
        </div>
      </div>
    </Link>
  )
}
