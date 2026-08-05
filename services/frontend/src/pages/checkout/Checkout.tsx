import { Link, useSearchParams } from 'react-router-dom'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { leaveQueue } from '../../features/queue/queueSlice'
import { ticketApi } from '../../features/ticket/api'

export function Checkout() {
  const [params] = useSearchParams()
  const ticketId = params.get('ticket')
  const dispatch = useAppDispatch()

  const entry = useAppSelector((state) =>
    state.queue.queueItems.find(
      (item) => item.id === ticketId && item.status === 'ticket',
    ),
  )
  const product = useAppSelector((state) =>
    state.products.productItems.find((p) => p.id === entry?.productId),
  )

  const handlePay = async () => {
    if (!ticketId) return

    try {
      await ticketApi.payOrder(ticketId)
      dispatch(leaveQueue(ticketId))
    } catch (error) {
      console.error(error)
    }
  }


  if (!ticketId || !entry) {
    return (
      <section className="rounded-2xl bg-white p-8 text-center">
        <h1 className="m-0 text-2xl tracking-tight">Нет права на покупку</h1>
        <p className="mt-3 text-avito-muted">
          Ссылка недействительна или тикет уже неактивен. Пройдите очередь
          заново.
        </p>
        <Link
          to="/catalog"
          className="mt-5 inline-flex min-h-12 items-center rounded-xl bg-avito-blue px-5 font-extrabold text-white no-underline"
        >
          В каталог
        </Link>
      </section>
    )
  }

  return (
    <section className="mx-auto max-w-[520px]">
      <Link
        to="/queue"
        className="mb-4 inline-flex min-h-10 items-center rounded-xl bg-[#f1f1f1] px-4 py-2 font-extrabold text-avito-ink no-underline"
      >
        ← Назад к очередям
      </Link>

      <div className="rounded-[22px] bg-white p-6 shadow-card">
        <div className="text-sm font-bold text-avito-muted">
          Оформление заказа
        </div>
        <h1 className="mt-2 text-2xl tracking-tight">
          {product?.name ?? `Товар ${entry.productId}`}
        </h1>
        <div className="mt-4 flex items-center gap-4 rounded-[14px] bg-[#f7f7f7] p-4">
          <div className="text-5xl" aria-hidden="true">
            {product?.image ?? '🛒'}
          </div>
          <div>
            <div className="font-extrabold">
              {product ? `${product.price.toLocaleString('ru-RU')} ₽` : '—'}
            </div>
            <div className="mt-1 text-sm text-avito-muted">
              Тикет: {entry.id}
            </div>
          </div>
        </div>
        <p className="mt-4 text-sm leading-relaxed text-[#555]">
          Это упрощённый чекаут для демо. После оплаты тикет сгорит, а товар
          спишется на бэкенде.
        </p>
        <button
          type="button"
          onClick={handlePay}
          className="mt-5 min-h-12 w-full cursor-pointer rounded-xl bg-avito-blue px-[18px] py-3 font-extrabold text-white"
        >
          Оплатить заказ
        </button>
      </div>
    </section>
  )
}
