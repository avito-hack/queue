import type { QueueTileView, TileKind } from './types'

export const statusStyles: Record<TileKind, string> = {
  queued: 'bg-avito-blue-soft text-[#0075b8]',
  ticket: 'bg-[#f0f9e7] text-[#4a7f14]',
  soldout: 'bg-[#fff0f2] text-[#b82334]',
}

export function statusLabel(
  tile: QueueTileView,
  countdown: string | null,
): string {
  if (tile.kind === 'queued') {
    return `В очереди · место ${tile.position ?? '—'}`
  }
  if (tile.kind === 'ticket') {
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
