import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import './Product.css'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import clockIcon from '../../assets/clock.svg'
import { JoinSuccessModal } from '../../components/modals/JoinSuccessModal'
import { SoldOutModal } from '../../components/modals/SoldOutModal'
import { productApi } from '../../features/product/api'
import {
  patchProductQueueCount,
  upsertProduct,
} from '../../features/product/productSlice'
import { queueApi } from '../../features/queue/api'
import { getProductActionLabel } from '../../features/queue/lib'
import {
  joinQueue as joinQueueAction,
  updateQueueItem,
} from '../../features/queue/queueSlice'
import type { ItemQueueState, QueueEntry } from '../../features/queue/types'
import { reportApiError } from '../../shared/api/errors'
import { showToast } from '../../shared/toast'

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
  const [waitingCount, setWaitingCount] = useState<number | null>(null)
  const [listingLoadError, setListingLoadError] = useState(false)
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
    setWaitingCount(null)
    setListingLoadError(false)
  }

  useEffect(() => {
    if (!productId || product) return
    let cancelled = false
    void (async () => {
      try {
        const listing = await productApi.getListing(productId)
        if (cancelled) return
        dispatch(upsertProduct(listing))
      } catch (error) {
        if (cancelled) return
        reportApiError(error, 'Не удалось загрузить товар')
        setListingLoadError(true)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [dispatch, product, productId])

  useEffect(() => {
    if (!productId) return
    let cancelled = false

    const syncQueueState = async () => {
      try {
        const data = await queueApi.getItemQueueState(productId)
        if (cancelled) return
        setQueueState(data.state)
        const count =
          typeof data.waiting_count === 'number' && data.waiting_count >= 0
            ? data.waiting_count
            : 0
        setWaitingCount(count)
        dispatch(patchProductQueueCount({ id: productId, queueCount: count }))
      } catch {
        if (cancelled) return
        setQueueState(null)
        setWaitingCount(0)
      }
    }

    void syncQueueState()
    const id = window.setInterval(() => {
      void syncQueueState()
    }, 5000)

    return () => {
      cancelled = true
      window.clearInterval(id)
    }
  }, [dispatch, productId])

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
      try {
        const data = await queueApi.getItemQueueState(targetProductId)
        const count =
          typeof data.waiting_count === 'number' && data.waiting_count >= 0
            ? data.waiting_count
            : (entry.position ?? waitingCount ?? 0)
        setQueueState(data.state)
        setWaitingCount(count)
        dispatch(
          patchProductQueueCount({ id: targetProductId, queueCount: count }),
        )
      } catch {
        const fallback = entry.position ?? (waitingCount ?? 0) + 1
        setWaitingCount(fallback)
        dispatch(
          patchProductQueueCount({
            id: targetProductId,
            queueCount: fallback,
          }),
        )
      }
    } catch (error) {
      reportApiError(
        error,
        'Не удалось встать в очередь. Попробуйте ещё раз',
      )
    }
  }

  const handleNotify = () => {
    localStorage.setItem(notifyStorageKey(productId), '1')
    setNotified(true)
    showToast('Подписка оформлена. Сообщим, когда товар появится', 'info')
  }

  if (!product) {
    if (!listingLoadError) {
      return (
        <section className="rounded-2xl bg-white p-8 text-center text-avito-muted">
          Загружаем товар…
        </section>
      )
    }
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
  const queueCount = waitingCount ?? product.queueCount ?? 0
  const stateHint = queueStateHint(queueState)
  const actionLabel = notified
    ? 'Подписка оформлена'
    : getProductActionLabel(inStock, myQueueEntry, myTicket)
  const priceLabel = `${product.price.toLocaleString('ru-RU')} ₽`

  const handlePrimaryAction = () => {
    if (myQueueEntry || myTicket) {
      navigate('/queue')
      return
    }
    if (!inStock) {
      handleNotify()
      return
    }
    void handleJoinQueue(product.id)
  }

  return (
    <section>
      <div className="mb-4 flex flex-wrap items-center gap-x-2 gap-y-1 text-[13px] text-avito-muted sm:mb-5 sm:text-sm">
        <button
          type="button"
          className="cursor-pointer bg-transparent text-avito-muted"
          onClick={() => navigate('/catalog')}
        >
          Главная
        </button>
        <span>›</span>
        <span className="line-clamp-1">{product.title}</span>
      </div>

      <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1.5fr)_minmax(300px,0.8fr)] lg:gap-7">
        <div className="product-gallery relative grid aspect-[4/3] max-h-[420px] place-items-center overflow-hidden rounded-2xl sm:aspect-auto sm:min-h-[480px] sm:max-h-none sm:rounded-3xl">
          <div className="absolute top-3 left-3 z-[2] rounded-lg bg-avito-ink px-2.5 py-1.5 text-[11px] font-extrabold tracking-wide text-white uppercase sm:top-5 sm:left-5 sm:rounded-xl sm:px-3 sm:py-2 sm:text-[12px]">
            Лимит
          </div>
          <div
            className="product-emoji z-[1] -rotate-6 select-none"
            aria-hidden="true"
          >
            {product.image ?? '🛒'}
          </div>
        </div>

        <aside className="rounded-2xl bg-avito-card p-4 shadow-card sm:p-6 lg:sticky lg:top-[84px]">
          <div className="mb-2 text-[13px] text-avito-muted">
            Лимитированный товар
          </div>
          <h1 className="m-0 text-[24px] leading-[1.15] font-extrabold tracking-tight sm:text-[28px]">
            {product.title}
          </h1>
          <div className="mt-3 text-[28px] font-extrabold tracking-tight sm:text-[32px]">
            {priceLabel}
          </div>

          <div className="my-5 grid gap-2 rounded-2xl bg-[#f5f5f5] p-3.5 sm:p-4">
            <div className="flex items-center justify-between gap-3 text-sm">
              <span className="text-avito-muted">В наличии</span>
              <strong
                className={inStock ? 'text-avito-green' : 'text-avito-red'}
              >
                {inStock ? `${product.availableQuantity} шт.` : 'нет в наличии'}
              </strong>
            </div>
            <div className="flex items-center justify-between gap-3 text-sm">
              <span className="text-avito-muted">Уже в очереди</span>
              <strong>
                {queueCount} {pluralPeople(queueCount)}
              </strong>
            </div>
            {stateHint && (
              <div className="flex items-center justify-between gap-3 text-sm">
                <span className="text-avito-muted">Очередь</span>
                <strong>{stateHint}</strong>
              </div>
            )}
          </div>

          {myQueueEntry && (
            <div className="mb-4 flex items-start gap-3 rounded-2xl bg-avito-blue-soft p-3.5 text-sm leading-snug text-[#006ca8]">
              <div
                className="grid size-8 shrink-0 place-items-center rounded-full bg-white/80"
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
            <div className="mb-4 flex items-start gap-3 rounded-2xl bg-[#eafaf1] p-3.5 text-sm leading-snug text-[#0a7a3e]">
              <div
                className="grid size-8 shrink-0 place-items-center rounded-full bg-white/80"
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

          <div className="fixed inset-x-0 bottom-0 z-40 border-t border-black/6 bg-white/95 px-4 pt-3 backdrop-blur-md safe-pb sm:static sm:z-auto sm:border-0 sm:bg-transparent sm:p-0 sm:backdrop-blur-none">
            <button
              type="button"
              disabled={!inStock && !myQueueEntry && !myTicket && notified}
              className="min-h-12 w-full cursor-pointer rounded-2xl bg-avito-blue px-4 py-3 font-extrabold text-white transition duration-150 hover:bg-avito-blue-hover disabled:cursor-default disabled:opacity-70"
              onClick={handlePrimaryAction}
            >
              {actionLabel}
            </button>
            <div className="mt-3 hidden text-center text-xs leading-snug text-avito-muted sm:block">
              Деньги не списываются. Место в очереди не гарантирует покупку.
              Право на покупку временное и действует только для вас.
            </div>
          </div>
        </aside>
      </div>

      <section className="mt-5 rounded-2xl bg-white p-4 sm:mt-7 sm:p-6">
        <h2 className="mb-3 text-xl font-extrabold tracking-tight sm:mb-4 sm:text-2xl">
          Описание
        </h2>
        <p className="m-0 text-[15px] leading-relaxed text-[#4d4d4d]">
          {product.description ?? 'Описание появится позже.'}
        </p>
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
        onCatalog={() => {
          setSoldOutDismissed(true)
          navigate('/catalog')
        }}
      />
    </section>
  )
}
