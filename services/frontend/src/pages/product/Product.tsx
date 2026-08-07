import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import './Product.css'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import clockIcon from '../../assets/clock.svg'
import { JoinSuccessModal } from '../../components/modals/JoinSuccessModal'
import { SoldOutModal } from '../../components/modals/SoldOutModal'
import { queueApi } from '../../features/queue/api'
import { getProductActionLabel } from '../../features/queue/lib'
import {
  joinQueue as joinQueueAction,
  updateQueueItem,
} from '../../features/queue/queueSlice'
import type { ItemQueueState, QueueEntry } from '../../features/queue/types'

const similarProducts = [
  { emoji: '👟', price: '14 500 ₽', title: 'Кроссовки Northline Base' },
  { emoji: '🧢', price: '3 900 ₽', title: 'Кепка из коллекции Drop 01' },
  { emoji: '🧥', price: '12 800 ₽', title: 'Куртка Northline Shell' },
  { emoji: '🎒', price: '6 700 ₽', title: 'Рюкзак Northline City' },
]

function pluralPeople(count: number): string {
  const mod10 = count % 10
  const mod100 = count % 100
  if (mod10 === 1 && mod100 !== 11) return 'человек'
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) {
    return 'человека'
  }
  return 'человек'
}

function notifyStorageKey(productId: string) {
  return `notify:${productId}`
}

function queueStateHint(state: ItemQueueState | null): string | null {
  if (state === 'tickets_available') return 'тикеты доступны'
  if (state === 'tickets_partially_issued') return 'тикеты частично выданы'
  if (state === 'tickets_exhausted') return 'тикеты закончились'
  return null
}

export function Product() {
  const { id } = useParams<{ id: string }>()
  const productId = id ?? ''
  const dispatch = useAppDispatch()
  const navigate = useNavigate()
  const [joinedEntry, setJoinedEntry] = useState<QueueEntry | null>(null)
  const [soldOutDismissed, setSoldOutDismissed] = useState(false)
  const [prevProductId, setPrevProductId] = useState(productId)
  const [queueState, setQueueState] = useState<ItemQueueState | null>(null)
  const [notified, setNotified] = useState(
    () => localStorage.getItem(notifyStorageKey(productId)) === '1',
  )

  const product = useAppSelector((state) =>
    state.products.productItems.find((p) => p.id === productId),
  )

  const myQueueEntry = useAppSelector((state) =>
    state.queue.queueItems.find(
      (item) => item.productId === productId && item.status === 'queued',
    ),
  )
  const mySoldoutEntry = useAppSelector((state) =>
    state.queue.queueItems.find(
      (item) => item.productId === productId && item.status === 'soldout',
    ),
  )
  const myTicket = useAppSelector((state) =>
    state.tickets.ticketItems.find((item) => item.productId === productId),
  )
  const alreadyInQueue = Boolean(myQueueEntry || myTicket)

  // Сброс UI-состояния при смене товара (без setState в effect).
  if (productId !== prevProductId) {
    setPrevProductId(productId)
    setNotified(localStorage.getItem(notifyStorageKey(productId)) === '1')
    setSoldOutDismissed(false)
    setJoinedEntry(null)
    setQueueState(null)
  }

  // /state в OpenAPI без length — тянем enum; число «уже в очереди» пока из listing/mock.
  useEffect(() => {
    if (!productId) return
    let cancelled = false
    void (async () => {
      try {
        const data = await queueApi.getItemQueueState(productId)
        if (!cancelled) setQueueState(data.state)
      } catch {
        if (!cancelled) setQueueState(null)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [productId])

  useEffect(() => {
    if (!product || product.availableQuantity > 0 || !myQueueEntry) return
    dispatch(
      updateQueueItem({
        ...myQueueEntry,
        status: 'soldout',
        position: undefined,
      }),
    )
  }, [dispatch, myQueueEntry, product])

  const soldOutOpen =
    !soldOutDismissed &&
    Boolean(
      mySoldoutEntry ||
        (product && product.availableQuantity === 0 && myQueueEntry),
    )
  const completeJoin = (entry: QueueEntry) => {
    dispatch(joinQueueAction(entry))
    setJoinedEntry(entry)
  }

  const handleJoinQueue = async (targetProductId: string) => {
    if (alreadyInQueue) return

    try {
      const entry = await queueApi.joinQueue(targetProductId)
      completeJoin(entry)
    } catch (error) {
      console.error(error)
      completeJoin({
        id: `${targetProductId}-entry`,
        productId: targetProductId,
        status: 'queued',
        position: 8,
      })
    }
  }

  const handleNotify = () => {
    localStorage.setItem(notifyStorageKey(productId), '1')
    setNotified(true)
  }

  if (!product) {
    return (
      <section className="rounded-2xl bg-white p-8 text-center text-avito-muted">
        Товар не найден. Откройте его из{' '}
        <button
          type="button"
          className="cursor-pointer bg-transparent font-extrabold text-[#008ed8]"
          onClick={() => navigate('/catalog')}
        >
          каталога
        </button>
        .
      </section>
    )
  }

  const inStock = product.availableQuantity > 0
  const queueCount = product.queueCount ?? 0
  const stateHint = queueStateHint(queueState)
  const actionLabel = notified
    ? 'Подписка оформлена'
    : getProductActionLabel(inStock, myQueueEntry, myTicket)
  const priceLabel = `${product.price.toLocaleString('ru-RU')} ₽`

  return (
    <section>
      <div className="mb-[22px] flex flex-wrap items-center gap-2 text-sm text-avito-muted">
        <a href="#" className="text-avito-muted no-underline">
          Главная
        </a>
        <span>›</span>
        <a href="#" className="text-avito-muted no-underline">
          Одежда и обувь
        </a>
        <span>›</span>
        <span>{product.title}</span>
      </div>

      <div className="grid items-start gap-7 lg:grid-cols-[minmax(0,1.55fr)_minmax(330px,0.75fr)]">
        <div className="product-gallery relative grid min-h-[360px] place-items-center overflow-hidden rounded-3xl sm:min-h-[520px]">
          <div className="absolute top-5 left-5 z-[2] rounded-full bg-avito-ink px-3 py-2 text-[13px] font-extrabold tracking-wide text-white">
            Лимитированный выпуск
          </div>
          <div
            className="product-emoji z-[1] -rotate-8 select-none"
            aria-hidden="true"
          >
            {product.image ?? '🛒'}
          </div>
          <div
            className="absolute bottom-[18px] left-1/2 z-[2] flex -translate-x-1/2 gap-1.5"
            aria-hidden="true"
          >
            <span className="h-2 w-[22px] rounded-full bg-avito-ink" />
            <span className="size-2 rounded-full bg-avito-ink/25" />
            <span className="size-2 rounded-full bg-avito-ink/25" />
            <span className="size-2 rounded-full bg-avito-ink/25" />
          </div>
        </div>

        <aside className="rounded-[22px] bg-avito-card p-6 shadow-card lg:sticky lg:top-[92px]">
          <div className="mb-2.5 text-sm text-avito-muted">
            Лимитированный товар
          </div>
          <h1 className="m-0 text-[26px] leading-[1.14] tracking-tight sm:text-[30px]">
            {product.title}
          </h1>
          <div className="mt-3.5 text-[27px] font-extrabold tracking-tight sm:text-[30px]">
            {priceLabel}
          </div>

          <div className="my-[22px] grid gap-2.5 rounded-[14px] bg-[#f7f7f7] p-4">
            <div className="flex items-center justify-between gap-3.5 text-sm">
              <span className="text-avito-muted">В наличии</span>
              <strong
                className={inStock ? 'text-avito-green' : 'text-avito-red'}
              >
                {inStock ? `${product.availableQuantity} шт.` : 'нет в наличии'}
              </strong>
            </div>
            <div className="flex items-center justify-between gap-3.5 text-sm">
              <span className="text-avito-muted">Уже в очереди</span>
              <strong>
                {queueCount} {pluralPeople(queueCount)}
              </strong>
            </div>
            {stateHint && (
              <div className="flex items-center justify-between gap-3.5 text-sm">
                <span className="text-avito-muted">Очередь</span>
                <strong>{stateHint}</strong>
              </div>
            )}
          </div>

          {myQueueEntry && (
            <div className="mb-[18px] flex items-start gap-3 rounded-[14px] bg-avito-blue-soft p-3.5 text-sm leading-snug text-[#006ca8]">
              <div
                className="grid size-[34px] shrink-0 place-items-center rounded-full bg-white/70"
                aria-hidden="true"
              >
                <img src={clockIcon} alt="" className="size-4" />
              </div>
              <div>
                <strong>Вы в очереди</strong>
                <span className="mt-1 block">
                  Ваше место: {myQueueEntry.position ?? '—'}. Мы сообщим, когда
                  появится право на покупку.
                </span>
              </div>
            </div>
          )}

          {myTicket && (
            <div className="mb-[18px] flex items-start gap-3 rounded-[14px] bg-[#f0f9e7] p-3.5 text-sm leading-snug text-[#477b12]">
              <div
                className="grid size-[34px] shrink-0 place-items-center rounded-full bg-white/70"
                aria-hidden="true"
              >
                ✓
              </div>
              <div>
                <strong>Есть право на покупку</strong>
                <span className="mt-1 block">
                  Товар закреплён за вами на ограниченное время. Перейдите к
                  оформлению, пока право не истекло.
                </span>
              </div>
            </div>
          )}

          <div className="grid gap-2.5">
            <button
              type="button"
              disabled={!inStock && !myQueueEntry && !myTicket && notified}
              className="min-h-12 w-full cursor-pointer rounded-xl bg-avito-blue px-[18px] py-3 font-extrabold text-white transition duration-150 hover:-translate-y-px hover:bg-avito-blue-hover disabled:cursor-default disabled:opacity-70 disabled:hover:translate-y-0"
              onClick={() => {
                if (myQueueEntry || myTicket) {
                  navigate('/queue')
                  return
                }
                if (!inStock) {
                  handleNotify()
                  return
                }
                void handleJoinQueue(product.id)
              }}
            >
              {actionLabel}
            </button>
          </div>

          <div className="mt-3 text-center text-xs leading-snug text-avito-muted">
            Деньги не списываются. Место в очереди не гарантирует покупку.
            Право на покупку временное и действует только для вас.
          </div>
        </aside>
      </div>

      <section className="mt-[30px] rounded-[18px] bg-white p-6">
        <h2 className="mb-[18px] text-2xl tracking-tight">Описание</h2>
        <p className="m-0 leading-relaxed text-[#4d4d4d]">
          {product.description ?? 'Описание появится позже.'}
        </p>
      </section>

      <section id="similar-products" className="mt-[34px]">
        <h2 className="mb-[18px] text-2xl tracking-tight">Похожие товары</h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {similarProducts.map((item) => (
            <article
              className="overflow-hidden rounded-2xl bg-white"
              key={item.title}
            >
              <div
                className="grid h-[150px] place-items-center bg-linear-to-br from-[#e8f7ff] to-[#f3e8ff] text-7xl"
                aria-hidden="true"
              >
                {item.emoji}
              </div>
              <div className="p-3.5">
                <div className="font-extrabold">{item.price}</div>
                <div className="mt-1 text-sm leading-snug text-[#3c3c3c]">
                  {item.title}
                </div>
              </div>
            </article>
          ))}
        </div>
      </section>

      <JoinSuccessModal
        open={joinedEntry !== null}
        position={joinedEntry?.position}
        productTitle={product.title}
        onClose={() => setJoinedEntry(null)}
        onGoToQueues={() => {
          setJoinedEntry(null)
          navigate('/queue')
        }}
      />

      <SoldOutModal
        open={soldOutOpen}
        productTitle={product.title}
        notified={notified}
        onClose={() => setSoldOutDismissed(true)}
        onNotify={handleNotify}
        onSimilar={() => {
          setSoldOutDismissed(true)
          document
            .getElementById('similar-products')
            ?.scrollIntoView({ behavior: 'smooth' })
        }}
      />    </section>
  )
}
