import { Link } from 'react-router-dom'

type QueueStatus = 'queued' | 'ticket' | 'soldout'

type QueueTile = {
  productId: string
  title: string
  emoji: string
  status: QueueStatus
  position?: number
  expiresIn?: string
  notified?: boolean
}

const mockTiles: QueueTile[] = [
  {
    productId: '1',
    title: 'Кроссовки Northline Drop 01',
    emoji: '👟',
    status: 'queued',
    position: 8,
  },
  {
    productId: '2',
    title: 'Куртка Northline Shell',
    emoji: '🧥',
    status: 'ticket',
    expiresIn: '28:42',
  },
  {
    productId: '3',
    title: 'Кепка Drop 01',
    emoji: '🧢',
    status: 'soldout',
    notified: false,
  },
]

const statusStyles: Record<QueueStatus, string> = {
  queued: 'bg-avito-blue-soft text-[#0075b8]',
  ticket: 'bg-[#f0f9e7] text-[#4a7f14]',
  soldout: 'bg-[#fff0f2] text-[#b82334]',
}

export function Queue() {
  const activeCount = mockTiles.filter((t) => t.status === 'queued').length
  const ticketCount = mockTiles.filter((t) => t.status === 'ticket').length
  const doneCount = mockTiles.filter((t) => t.status === 'soldout').length

  return (
    <section>
      <div className="mb-[22px] flex flex-col items-start justify-between gap-5 sm:flex-row sm:items-end">
        <div>
          <h1 className="m-0 text-2xl tracking-tight sm:text-[30px]">
            Мои очереди
          </h1>
          <p className="mt-2 max-w-[620px] leading-normal text-avito-muted">
            Каждая плитка — отдельная очередь. Откройте товар, чтобы посмотреть
            позицию, право на покупку или продолжить оформление.
          </p>
        </div>
        <Link
          to="/product/1"
          className="inline-flex min-h-10 items-center rounded-xl bg-[#f1f1f1] px-4 py-2 font-extrabold text-avito-ink no-underline"
        >
          ← Вернуться к товару
        </Link>
      </div>

      <div className="mb-[22px] grid grid-cols-1 gap-3.5 sm:grid-cols-3">
        <SummaryTile label="Активных очередей" value={activeCount} />
        <SummaryTile label="Доступно к покупке" value={ticketCount} />
        <SummaryTile label="Завершено" value={doneCount} />
      </div>

      <div className="grid grid-cols-1 gap-[18px] sm:grid-cols-2 lg:grid-cols-3">
        {mockTiles.map((tile) => (
          <QueueCard key={tile.productId} tile={tile} />
        ))}
      </div>
    </section>
  )
}

function SummaryTile({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-2xl bg-white p-[18px]">
      <div className="text-sm text-avito-muted">{label}</div>
      <div className="mt-2 text-[28px] font-extrabold">{value}</div>
    </div>
  )
}

function QueueCard({ tile }: { tile: QueueTile }) {
  return (
    <Link
      to={`/product/${tile.productId}`}
      className="flex min-h-[365px] flex-col overflow-hidden rounded-[20px] border border-transparent bg-white no-underline transition duration-150 hover:-translate-y-0.5 hover:border-[#c9c9c9] hover:shadow-card"
    >
      <div className="grid h-[170px] place-items-center bg-linear-to-br from-[#dff5ff] to-[#e7dcff] text-[92px]">
        {tile.emoji}
      </div>

      <div className="flex flex-1 flex-col p-[18px]">
        <div
          className={`mb-2.5 self-start rounded-full px-2.5 py-1.5 text-xs font-extrabold ${statusStyles[tile.status]}`}
        >
          {statusLabel(tile)}
        </div>

        <div className="text-lg font-extrabold leading-snug text-avito-ink">
          {tile.title}
        </div>

        <div className="mt-4 grid gap-2 text-sm text-[#555]">
          {metaRows(tile).map((row) => (
            <div key={row.label} className="flex justify-between gap-3">
              <span>{row.label}</span>
              <strong className="text-avito-ink">{row.value}</strong>
            </div>
          ))}
        </div>

        <div className="mt-auto pt-[18px] text-sm font-extrabold text-[#008ed8]">
          {tile.status === 'ticket'
            ? 'Перейти к покупке →'
            : 'Перейти на страницу товара →'}
        </div>
      </div>
    </Link>
  )
}

function statusLabel(tile: QueueTile): string {
  if (tile.status === 'queued') {
    return `В очереди · место ${tile.position}`
  }
  if (tile.status === 'ticket') {
    return `Право на покупку · ${tile.expiresIn}`
  }
  return 'Товар закончился'
}

function metaRows(tile: QueueTile): { label: string; value: string }[] {
  if (tile.status === 'queued') {
    return [
      { label: 'Перед вами', value: String(Math.max(0, (tile.position ?? 1) - 1)) },
      { label: 'Тикет', value: 'ожидается' },
    ]
  }
  if (tile.status === 'ticket') {
    return [
      { label: 'Право на покупку', value: '1 шт.' },
      { label: 'Осталось', value: tile.expiresIn ?? '—' },
    ]
  }
  return [
    {
      label: 'Поступление',
      value: tile.notified ? 'уведомление включено' : 'можно подписаться',
    },
    { label: 'Тикет', value: 'нет' },
  ]
}
