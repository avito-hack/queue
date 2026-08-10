import { useEffect, useMemo, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { demoApi } from '../../features/demo/demoApi'
import {
  clearDemoUsers,
  DEMO_USERS_EVENT,
  MAX_DEMO_USERS,
  readDemoUsers,
  type DemoUser,
} from '../../features/demo/demoUsers'
import { productApi } from '../../features/product/api'
import {
  setProductItems,
  upsertProduct,
} from '../../features/product/productSlice'
import {
  joinQueue as joinQueueAction,
  removeQueuedByProductId,
} from '../../features/queue/queueSlice'
import type { TicketListItemDto } from '../../features/ticket/api'
import {
  removeTicket,
  upsertTicket,
} from '../../features/ticket/ticketSlice'
import { toTicketEntry } from '../../features/ticket/useTicketPolling'
import {
  DEFAULT_DEMO_TOKEN,
  DEFAULT_DEMO_USER_ID,
  readStoredAuthToken,
} from '../../shared/auth/demoAuth'
import { currentUserId } from '../../shared/auth/currentUser'
import { switchJuryUser } from '../../features/demo/switchJuryUser'
import { reportApiError } from '../../shared/api/errors'
import { showToast } from '../../shared/toast'

function isActiveSessionToken(token: string): boolean {
  return token.trim() === (readStoredAuthToken() ?? DEFAULT_DEMO_TOKEN)
}

function productIdFromPath(pathname: string): string | null {
  const match = pathname.match(/^\/product\/([^/]+)/)
  return match?.[1] ?? null
}

export function DemoPanel() {
  const dispatch = useAppDispatch()
  const location = useLocation()
  const products = useAppSelector((state) => state.products.productItems)

  const [open, setOpen] = useState(true)
  const [listingId, setListingId] = useState('')
  const [userToken, setUserToken] = useState(
    () => readStoredAuthToken() ?? DEFAULT_DEMO_TOKEN,
  )
  const [ticketId, setTicketId] = useState('')
  const [quantity, setQuantity] = useState('1')
  const [registerCount, setRegisterCount] = useState('10')
  const [loadingProducts, setLoadingProducts] = useState(false)
  const [busy, setBusy] = useState<string | null>(null)
  const [userTickets, setUserTickets] = useState<TicketListItemDto[]>([])
  const [demoUsers, setDemoUsers] = useState<DemoUser[]>(() => readDemoUsers())
  const [createdPopup, setCreatedPopup] = useState<DemoUser[] | null>(null)

  const pathProductId = productIdFromPath(location.pathname)
  const [prevPathProductId, setPrevPathProductId] = useState(pathProductId)
  if (pathProductId !== prevPathProductId) {
    setPrevPathProductId(pathProductId)
    if (pathProductId) setListingId(pathProductId)
  }

  useEffect(() => {
    const sync = () => setDemoUsers(readDemoUsers())
    window.addEventListener(DEMO_USERS_EVENT, sync)
    return () => window.removeEventListener(DEMO_USERS_EVENT, sync)
  }, [])

  const selected = useMemo(
    () => products.find((p) => p.id === listingId) ?? null,
    [listingId, products],
  )

  const quantitySourceKey = selected
    ? `${selected.id}:${selected.availableQuantity}`
    : ''
  const [quantitySyncedKey, setQuantitySyncedKey] = useState('')
  if (selected && quantitySourceKey !== quantitySyncedKey) {
    setQuantitySyncedKey(quantitySourceKey)
    setQuantity(String(selected.availableQuantity))
  }

  const refreshProducts = async () => {
    setLoadingProducts(true)
    try {
      const list = await productApi.getProducts()
      dispatch(setProductItems(list))
      setListingId((current) => current || list[0]?.id || '')
    } catch (error) {
      reportApiError(error, 'Не удалось загрузить товары для демо')
    } finally {
      setLoadingProducts(false)
    }
  }

  useEffect(() => {
    let cancelled = false
    void (async () => {
      setLoadingProducts(true)
      try {
        const list = await productApi.getProducts()
        if (cancelled) return
        dispatch(setProductItems(list))
        setListingId((current) => current || list[0]?.id || '')
      } catch (error) {
        if (!cancelled) {
          reportApiError(error, 'Не удалось загрузить товары для демо')
        }
      } finally {
        if (!cancelled) setLoadingProducts(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [dispatch])

  const requireListing = () => {
    if (!listingId) {
      showToast('Выберите товар / очередь', 'error')
      return false
    }
    return true
  }

  const requireUser = () => {
    if (!userToken.trim()) {
      showToast('Вставьте токен / id пользователя', 'error')
      return false
    }
    return true
  }

  const listingLabel = selected?.title ?? listingId

  const handleRegisterUsers = async () => {
    const requested = Number(registerCount)
    if (!Number.isInteger(requested) || requested < 1 || requested > MAX_DEMO_USERS) {
      showToast(`Укажите число от 1 до ${MAX_DEMO_USERS}`, 'error')
      return
    }
    setBusy('register')
    try {
      const { created, all } = await demoApi.registerDemoUsers(requested)
      setDemoUsers(all)
      if (created.length === 0) {
        showToast(`Уже есть ${all.length}/${MAX_DEMO_USERS} пользователей`, 'info')
        return
      }
      setCreatedPopup(created)
      setUserToken(created[0]?.token ?? userToken)
      showToast(`Зарегистрировано: ${created.length}. Всего: ${all.length}`, 'info')
    } catch (error) {
      reportApiError(error, 'Не удалось зарегистрировать пользователей')
    } finally {
      setBusy(null)
    }
  }

  const handleSwitchUser = (token: string) => {
    setUserToken(token)
    const user =
      token === DEFAULT_DEMO_TOKEN
        ? { id: DEFAULT_DEMO_USER_ID, token: DEFAULT_DEMO_TOKEN }
        : demoUsers.find((item) => item.token === token)
    if (!user) return

    void (async () => {
      const result = await switchJuryUser(user)
      if (result === 'removed') {
        setDemoUsers(readDemoUsers())
        setUserToken(readStoredAuthToken() ?? DEFAULT_DEMO_TOKEN)
        showToast(
          'Этот демо-пользователь больше недоступен. Создайте заново.',
          'error',
        )
      }
    })()
  }

  const handleEnqueue = async () => {
    if (!requireListing() || !requireUser()) return
    setBusy('enqueue')
    try {
      const result = await demoApi.enqueueAsUser(listingId, userToken)
      if (isActiveSessionToken(userToken)) {
        dispatch(
          joinQueueAction({
            id: `${listingId}-${currentUserId()}`,
            productId: listingId,
            status: 'queued',
            position: result.position,
            memberStatus: 'waiting_in_line',
          }),
        )
      }
      const place =
        typeof result.position === 'number' ? ` Место: ${result.position}.` : ''
      showToast(`В очереди на «${listingLabel}».${place}`, 'info')
    } catch (error) {
      reportApiError(error, 'Не удалось поставить в очередь')
    } finally {
      setBusy(null)
    }
  }

  const handleDequeue = async () => {
    if (!requireListing() || !requireUser()) return
    setBusy('dequeue')
    try {
      await demoApi.dequeueAsUser(listingId, userToken)
      if (isActiveSessionToken(userToken)) {
        dispatch(removeQueuedByProductId(listingId))
      }
      showToast(`Убран из очереди «${listingLabel}»`, 'info')
    } catch (error) {
      reportApiError(error, 'Не удалось убрать из очереди')
    } finally {
      setBusy(null)
    }
  }

  const handleLoadTickets = async () => {
    if (!requireUser()) return
    setBusy('tickets')
    try {
      const data = await demoApi.listTicketsAsUser(userToken)
      const list = Array.isArray(data.ticket) ? data.ticket : []
      setUserTickets(list)
      setTicketId((current) =>
        list.some((ticket) => ticket.id === current) ? current : '',
      )
      showToast(
        list.length > 0
          ? `Найдено тикетов: ${list.length}. Выберите в списке.`
          : 'У пользователя нет тикетов',
        'info',
      )
    } catch (error) {
      reportApiError(error, 'Не удалось загрузить тикеты пользователя')
    } finally {
      setBusy(null)
    }
  }

  const handleIssueTicket = async () => {
    if (!requireListing() || !requireUser()) return
    setBusy('issue')
    try {
      const dto = await demoApi.issueTicketForUser(listingId, userToken)
      setUserTickets((prev) => {
        if (!dto.id) return prev
        if (prev.some((item) => item.id === dto.id)) return prev
        return [dto, ...prev]
      })
      if (isActiveSessionToken(userToken)) {
        const entry = toTicketEntry(dto)
        if (entry) {
          dispatch(upsertTicket(entry))
          dispatch(removeQueuedByProductId(listingId))
        }
      }
      showToast(
        dto.id
          ? `Тикет выдан (${dto.id.slice(0, 8)}…). Выберите его в п.5 при необходимости.`
          : `Тикет выдан на «${listingLabel}»`,
        'info',
      )
    } catch (error) {
      reportApiError(error, 'Не удалось выдать тикет')
    } finally {
      setBusy(null)
    }
  }

  const handleInvalidate = async () => {
    const id = ticketId.trim()
    if (!requireUser()) return
    if (!id) {
      showToast('Выберите тикет из списка или вставьте id', 'error')
      return
    }
    setBusy('invalidate')
    try {
      await demoApi.invalidateTicketAsUser(id, userToken)
      dispatch(removeTicket(id))
      setUserTickets((prev) => prev.filter((t) => t.id !== id))
      setTicketId('')
      showToast('Тикет закрыт', 'info')
    } catch (error) {
      reportApiError(error, 'Не удалось инвалидировать тикет')
    } finally {
      setBusy(null)
    }
  }

  const handleQuantity = async () => {
    if (!requireListing()) return
    const next = Number(quantity)
    if (!Number.isInteger(next) || next < 0) {
      showToast('Количество должно быть целым ≥ 0', 'error')
      return
    }
    setBusy('quantity')
    try {
      const product = await demoApi.changeListingQuantity(listingId, next)
      dispatch(upsertProduct(product))
      showToast(
        `«${product.title}»: количество = ${product.quantity} (доступно ${product.availableQuantity})`,
        'info',
      )
    } catch (error) {
      reportApiError(error, 'Не удалось изменить количество')
    } finally {
      setBusy(null)
    }
  }

  return (
    <div className="demo-panel fixed right-3 bottom-[calc(1rem+env(safe-area-inset-bottom))] z-[90] w-[min(100%-1.5rem,360px)] sm:right-4">
      <div className="overflow-hidden rounded-2xl border border-black/8 bg-white shadow-[0_12px_40px_rgba(0,0,0,0.16)]">
        <button
          type="button"
          className="flex w-full cursor-pointer items-center justify-between gap-3 bg-[#1a1a1a] px-3.5 py-3 text-left text-sm font-extrabold text-white"
          onClick={() => setOpen((value) => !value)}
          aria-expanded={open}
        >
          <span>Демо-сценарии</span>
          <span
            aria-hidden="true"
            className={`demo-panel__chevron ${open ? 'is-open' : ''}`}
          >
            ⌃
          </span>
        </button>

        <div className={`demo-panel__body ${open ? 'is-open' : ''}`}>
          <div className="demo-panel__body-inner">
            <div className="grid max-h-[min(70dvh,640px)] gap-3 overflow-y-auto p-3.5">
            <section className="grid gap-2 rounded-xl bg-[#f7f7f7] p-3">
              <div className="text-[13px] font-extrabold">
                1. Зарегистрировать пользователей
              </div>
              <p className="m-0 text-[12px] leading-snug text-avito-muted">
                Создаёт до {MAX_DEMO_USERS} buyers через avito{' '}
                <code className="text-[11px]">POST /users</code>. Храним в
                localStorage; переключение — в шапке «Пользователи» или ниже.
              </p>
              <div className="grid grid-cols-[88px_1fr] gap-2">
                <input
                  type="number"
                  min={1}
                  max={MAX_DEMO_USERS}
                  value={registerCount}
                  onChange={(event) => setRegisterCount(event.target.value)}
                  className="min-h-10 rounded-xl border border-black/8 bg-white px-3 text-[13px] font-semibold outline-none focus:border-[#00aaff]"
                />
                <button
                  type="button"
                  className="min-h-10 cursor-pointer rounded-xl bg-avito-blue px-3 text-[13px] font-extrabold text-white disabled:opacity-60"
                  onClick={() => void handleRegisterUsers()}
                  disabled={busy !== null}
                >
                  {busy === 'register'
                    ? 'Регистрируем…'
                    : `Добавить (сейчас ${demoUsers.length}/${MAX_DEMO_USERS})`}
                </button>
              </div>
              {demoUsers.length > 0 && (
                <>
                  <label className="grid gap-1 text-[12px] font-bold text-[#3c3c3c]">
                    Переключить пользователя
                    <select
                      className="min-h-10 w-full rounded-xl border border-black/8 bg-white px-3 text-[13px] font-semibold outline-none"
                      value={userToken}
                      onChange={(event) => handleSwitchUser(event.target.value)}
                    >
                      <option value={DEFAULT_DEMO_TOKEN}>Default User</option>
                      {demoUsers.map((user, index) => (
                        <option key={user.token} value={user.token}>
                          {index + 1}. {user.name}
                        </option>
                      ))}
                    </select>
                  </label>
                  <button
                    type="button"
                    className="min-h-9 cursor-pointer rounded-xl bg-white px-3 text-[12px] font-bold text-[#b82334]"
                    onClick={() => {
                      clearDemoUsers()
                      setDemoUsers([])
                      showToast('Список демо-пользователей очищен', 'info')
                    }}
                  >
                    Очистить список
                  </button>
                </>
              )}
            </section>

            <section className="grid gap-2 rounded-xl bg-[#f7f7f7] p-3">
              <div className="flex items-center justify-between gap-2">
                <div className="text-[13px] font-extrabold">Товар / очередь</div>
                <button
                  type="button"
                  className="cursor-pointer rounded-lg bg-white px-2 py-1 text-[11px] font-bold text-[#0078bd]"
                  onClick={() => void refreshProducts()}
                  disabled={loadingProducts}
                >
                  {loadingProducts ? '…' : 'Обновить'}
                </button>
              </div>
              <select
                className="min-h-10 w-full rounded-xl border border-black/8 bg-white px-3 text-[13px] font-semibold outline-none focus:border-[#00aaff]"
                value={listingId}
                onChange={(event) => setListingId(event.target.value)}
              >
                <option value="" disabled>
                  Выберите товар
                </option>
                {products.map((product) => (
                  <option key={product.id} value={product.id}>
                    {product.title} · ост. {product.availableQuantity}
                  </option>
                ))}
              </select>
              {selected && (
                <div className="rounded-lg bg-white px-3 py-2 text-[12px] leading-snug text-[#3c3c3c]">
                  <div className="font-extrabold text-avito-ink">
                    {selected.title}
                  </div>
                  <div className="mt-1 break-all text-avito-muted">
                    {selected.id}
                  </div>
                  <div className="mt-0.5">
                    В наличии: {selected.availableQuantity} / {selected.quantity}{' '}
                    · очередь {selected.queueEnabled ? 'вкл.' : 'выкл.'}
                  </div>
                </div>
              )}
              <label className="grid gap-1 text-[12px] font-bold text-[#3c3c3c]">
                Токен / id пользователя
                <select
                  className="min-h-10 w-full rounded-xl border border-black/8 bg-white px-3 text-[13px] font-semibold outline-none"
                  value={
                    demoUsers.some((u) => u.token === userToken) ||
                    userToken === DEFAULT_DEMO_TOKEN
                      ? userToken
                      : ''
                  }
                  onChange={(event) => {
                    if (event.target.value) setUserToken(event.target.value)
                  }}
                >
                  <option value="">Ввести вручную ↓</option>
                  <option value={DEFAULT_DEMO_TOKEN}>Default User</option>
                  {demoUsers.map((user, index) => (
                    <option key={user.token} value={user.token}>
                      {index + 1}. {user.name}
                    </option>
                  ))}
                </select>
                <input
                  type="text"
                  value={userToken}
                  onChange={(event) => setUserToken(event.target.value)}
                  placeholder="uuid токена"
                  className="min-h-10 w-full rounded-xl border border-black/8 bg-white px-3 text-[13px] font-semibold outline-none focus:border-[#00aaff]"
                  autoComplete="off"
                  spellCheck={false}
                />
              </label>
            </section>

            <section className="grid gap-2 rounded-xl bg-[#f7f7f7] p-3">
              <div className="text-[13px] font-extrabold">
                2. В очередь по пользователю
              </div>
              <button
                type="button"
                className="min-h-10 cursor-pointer rounded-xl bg-avito-blue px-3 text-[13px] font-extrabold text-white disabled:opacity-60"
                onClick={() => void handleEnqueue()}
                disabled={busy !== null || !listingId}
              >
                {busy === 'enqueue' ? 'Ставим…' : 'В очередь'}
              </button>
            </section>

            <section className="grid gap-2 rounded-xl bg-[#f7f7f7] p-3">
              <div className="text-[13px] font-extrabold">
                3. Убрать из очереди по пользователю
              </div>
              <button
                type="button"
                className="min-h-10 cursor-pointer rounded-xl bg-[#ebebeb] px-3 text-[13px] font-extrabold text-avito-ink disabled:opacity-60"
                onClick={() => void handleDequeue()}
                disabled={busy !== null || !listingId}
              >
                {busy === 'dequeue' ? 'Убираем…' : 'Убрать из очереди'}
              </button>
            </section>

            <section className="grid gap-2 rounded-xl bg-[#f7f7f7] p-3">
              <div className="text-[13px] font-extrabold">
                4. Выдать тикет на товар
              </div>
              <button
                type="button"
                className="min-h-10 cursor-pointer rounded-xl bg-avito-blue px-3 text-[13px] font-extrabold text-white disabled:opacity-60"
                onClick={() => void handleIssueTicket()}
                disabled={busy !== null || !listingId}
              >
                {busy === 'issue' ? 'Выдаём…' : 'Выдать тикет'}
              </button>
            </section>

            <section className="grid gap-2 rounded-xl bg-[#f7f7f7] p-3">
              <div className="text-[13px] font-extrabold">
                5. Инвалидировать тикет
              </div>
              <button
                type="button"
                className="min-h-10 cursor-pointer rounded-xl bg-white px-3 text-[12px] font-extrabold text-[#0078bd] disabled:opacity-60"
                onClick={() => void handleLoadTickets()}
                disabled={busy !== null}
              >
                {busy === 'tickets'
                  ? '…'
                  : userTickets.length > 0
                    ? `Обновить список (${userTickets.length})`
                    : 'Загрузить тикеты юзера'}
              </button>
              <label className="grid gap-1 text-[12px] font-bold text-[#3c3c3c]">
                Выберите тикет
                <select
                  className="min-h-10 w-full rounded-xl border border-black/8 bg-white px-3 text-[13px] font-semibold outline-none focus:border-[#00aaff]"
                  value={
                    userTickets.some((ticket) => ticket.id === ticketId)
                      ? ticketId
                      : ''
                  }
                  onChange={(event) => setTicketId(event.target.value)}
                  disabled={userTickets.length === 0}
                >
                  <option value="">
                    {userTickets.length
                      ? `Выберите из ${userTickets.length}…`
                      : 'Сначала загрузите тикеты'}
                  </option>
                  {userTickets.map((ticket) => (
                    <option key={ticket.id} value={ticket.id ?? ''}>
                      {(ticket.status ?? '?') +
                        ' · ' +
                        (ticket.listing_id?.slice(0, 8) ?? 'listing') +
                        ' · ' +
                        (ticket.id?.slice(0, 8) ?? '')}
                    </option>
                  ))}
                </select>
              </label>
              <input
                type="text"
                value={ticketId}
                onChange={(event) => setTicketId(event.target.value)}
                placeholder="или вставьте ticket id вручную"
                className="min-h-10 w-full rounded-xl border border-black/8 bg-white px-3 text-[13px] font-semibold outline-none focus:border-[#00aaff]"
                autoComplete="off"
                spellCheck={false}
              />
              <button
                type="button"
                className="min-h-10 cursor-pointer rounded-xl bg-[#ff4053] px-3 text-[13px] font-extrabold text-white disabled:opacity-60"
                onClick={() => void handleInvalidate()}
                disabled={busy !== null || !ticketId.trim()}
              >
                {busy === 'invalidate' ? 'Закрываем…' : 'Инвалидировать'}
              </button>
            </section>

            <section className="grid gap-2 rounded-xl bg-[#f7f7f7] p-3">
              <div className="text-[13px] font-extrabold">
                6. Изменить количество товара
              </div>
              <p className="m-0 text-[12px] leading-snug text-avito-muted">
                Меняет quantity выбранного объявления выше.
              </p>
              <label className="grid gap-1 text-[12px] font-bold text-[#3c3c3c]">
                Новое количество
                <input
                  type="number"
                  min={0}
                  step={1}
                  value={quantity}
                  onChange={(event) => setQuantity(event.target.value)}
                  className="min-h-10 w-full rounded-xl border border-black/8 bg-white px-3 text-[13px] font-semibold outline-none focus:border-[#00aaff]"
                />
              </label>
              <button
                type="button"
                className="min-h-10 cursor-pointer rounded-xl bg-avito-blue px-3 text-[13px] font-extrabold text-white disabled:opacity-60"
                onClick={() => void handleQuantity()}
                disabled={busy !== null || !listingId}
              >
                {busy === 'quantity' ? 'Сохраняем…' : 'Изменить количество'}
              </button>
            </section>
            </div>
          </div>
        </div>
      </div>

      {createdPopup && (
        <div
          className="fixed inset-0 z-[110] grid place-items-center bg-black/50 p-4"
          onClick={() => setCreatedPopup(null)}
          role="presentation"
        >
          <div
            className="max-h-[80dvh] w-full max-w-md overflow-y-auto rounded-2xl bg-white p-5 shadow-card"
            onClick={(event) => event.stopPropagation()}
            role="dialog"
            aria-modal="true"
            aria-labelledby="demo-users-title"
          >
            <h2
              id="demo-users-title"
              className="m-0 text-lg font-extrabold tracking-tight"
            >
              Созданные пользователи
            </h2>
            <p className="mt-2 text-[13px] text-avito-muted">
              Сохранены на фронте. Для API нужен token (Bearer).
            </p>
            <ul className="mt-3 grid gap-2 p-0">
              {createdPopup.map((user, index) => (
                <li
                  key={user.token}
                  className="list-none rounded-xl bg-[#f7f7f7] px-3 py-2 text-[12px] leading-snug"
                >
                  <div className="font-extrabold">
                    {index + 1}. {user.name}
                  </div>
                  <div className="mt-1 break-all text-avito-muted">
                    id: {user.id}
                  </div>
                  <div className="break-all text-avito-muted">
                    token: {user.token}
                  </div>
                </li>
              ))}
            </ul>
            <div className="mt-4 grid grid-cols-2 gap-2">
              <button
                type="button"
                className="min-h-11 cursor-pointer rounded-xl bg-[#ebebeb] px-3 font-extrabold"
                onClick={() => {
                  const text = createdPopup
                    .map(
                      (user, index) =>
                        `${index + 1}. ${user.name}\nid: ${user.id}\ntoken: ${user.token}`,
                    )
                    .join('\n\n')
                  void navigator.clipboard.writeText(text)
                  showToast('Скопировано в буфер', 'info')
                }}
              >
                Копировать
              </button>
              <button
                type="button"
                className="min-h-11 cursor-pointer rounded-xl bg-avito-blue px-3 font-extrabold text-white"
                onClick={() => setCreatedPopup(null)}
              >
                Закрыть
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
