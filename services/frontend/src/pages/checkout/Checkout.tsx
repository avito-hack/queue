import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { removeTicket } from '../../features/ticket/ticketSlice'

export function Checkout() {
  const [params] = useSearchParams()
  const ticketId = params.get('ticket')
  const dispatch = useAppDispatch()
  const [paying, setPaying] = useState(false)
  const [paid, setPaid] = useState(false)

  const ticket = useAppSelector((state) =>
    state.tickets.ticketItems.find((item) => item.id === ticketId),
  )
  const product = useAppSelector((state) =>
    state.products.productItems.find((p) => p.id === ticket?.productId),
  )

  const handlePay = () => {
    if (!ticketId || paying) return

    // В OpenAPI tickets нет /pay — оплата через activate → checkout_url / avito orders.
    // Этот экран — демо-заглушка, если activate упал или url локальный.
    setPaying(true)
    dispatch(removeTicket(ticketId))
    setPaid(true)
    setPaying(false)
  }

  if (paid) {
    return (
      <section className="rounded-2xl bg-white p-8 text-center">
        <h1 className="m-0 text-2xl tracking-tight">Заказ оформлен</h1>
        <p className="mt-3 text-avito-muted">
          Демо-чекаут: тикет убран локально. В бою погашение приходит с бэка
          после оплаты по checkout_url.
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

  if (!ticketId || !ticket) {
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
          {product?.title ?? `Товар ${ticket.productId}`}
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
              Тикет: {ticket.id}
            </div>
          </div>
        </div>
        <p className="mt-4 text-sm leading-relaxed text-[#555]">
          Упрощённый чекаут для демо (нет endpoint оплаты в tickets API). Кнопка
          только гасит тикет в store.
        </p>
        <button
          type="button"
          onClick={handlePay}
          disabled={paying}
          className="mt-5 min-h-12 w-full cursor-pointer rounded-xl bg-avito-blue px-[18px] py-3 font-extrabold text-white disabled:cursor-default disabled:opacity-50"
        >
          {paying ? 'Оплата…' : 'Оплатить заказ'}
        </button>
      </div>
    </section>
  )
}
