import type { QueueTileView, TileKind } from './types'

export const statusStyles: Record<TileKind, string> = {
  queued: 'bg-avito-blue-soft text-[#0075b8]',
  ticket: 'bg-[#f0f9e7] text-[#4a7f14]',
  soldout: 'bg-[#fff0f2] text-[#b82334]',
}

const REDEEMED_BADGE =
  'bg-[#fff4e5] text-[#b25c00] ring-1 ring-[#ffb347]/55'

export function isRedeemedTile(tile: QueueTileView): boolean {
  return (
    tile.kind === 'ticket' &&
    (tile.ticketStatus === 'redeemed' || Boolean(tile.checkoutUrl))
  )
}

export function tileStatusClass(tile: QueueTileView): string {
  if (isRedeemedTile(tile)) return REDEEMED_BADGE
  return statusStyles[tile.kind]
}

export function statusLabel(
  tile: QueueTileView,
  countdown: string | null,
): string {
  if (tile.kind === 'queued') {
    return `В очереди · место ${tile.position ?? '—'}`
  }
  if (tile.kind === 'ticket') {
    if (isRedeemedTile(tile)) {
      return 'Тикет активирован · к оплате'
    }
    return countdown
      ? `Право на покупку · ${countdown}`
      : 'Право на покупку'
  }
  return 'Товар закончился'
}

export function metaRows(
  tile: QueueTileView,
  countdown: string | null,
): { label: string; value: string }[] {
  if (tile.kind === 'queued') {
    return [
      {
        label: 'Перед вами',
        value: String(Math.max(0, (tile.position ?? 1) - 1)),
      },
      { label: 'Тикет', value: 'ожидается' },
    ]
  }
  if (tile.kind === 'ticket') {
    if (isRedeemedTile(tile)) {
      return [
        { label: 'Этап', value: 'оформление заказа' },
        { label: 'Оплата', value: 'ожидается' },
      ]
    }
    return [
      { label: 'Право на покупку', value: '1 шт.' },
      { label: 'Осталось', value: countdown ?? '—' },
    ]
  }
  return [
    { label: 'Поступление', value: 'можно подписаться' },
    { label: 'Тикет', value: 'нет' },
  ]
}
