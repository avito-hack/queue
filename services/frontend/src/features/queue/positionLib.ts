import type { QueueEntry, QueueStatus } from './types'

/** GET /v1/queue/{itemID}/position */
export type UserPositionResponse = {
  position: number
}

/** Элемент GET /v1/user/queues */
export type UserQueueInfo = {
  item_id?: string
  position?: number
  status?: string
  ticket_id?: string | null
  ticket_status?: string | null
  can_activate?: boolean
  can_exit?: boolean
  can_decline_ticket?: boolean
}

function queueStatusFromUserStatus(status?: string): QueueStatus | null {
  if (status === 'waiting_in_line') return 'queued'
  if (status === 'item_out_of_stock') return 'soldout'
  return null
}

/**
 * Обновить локальные queued/soldout плитки по массиву /v1/user/queues.
 * Позиция всегда привязана к item_id — не размазываем одно число на все.
 */
export function queueItemsFromUserQueues(
  queueItems: QueueEntry[],
  infos: UserQueueInfo[],
): QueueEntry[] {
  const updates: QueueEntry[] = []

  for (const item of queueItems) {
    if (item.status !== 'queued' && item.status !== 'soldout') continue

    const info = infos.find((row) => row.item_id === item.productId)
    if (!info) continue

    const nextStatus = queueStatusFromUserStatus(info.status)
    if (!nextStatus) continue

    const next: QueueEntry = {
      ...item,
      status: nextStatus,
      position:
        nextStatus === 'queued' && typeof info.position === 'number'
          ? info.position
          : undefined,
    }

    if (next.position !== item.position || next.status !== item.status) {
      updates.push(next)
    }
  }

  return updates
}

/** Применить { position } к конкретной очереди itemId. */
export function queueItemWithPosition(
  queueItems: QueueEntry[],
  itemId: string,
  position: number,
): QueueEntry | null {
  const item = queueItems.find(
    (q) => q.productId === itemId && q.status === 'queued',
  )
  if (!item) return null
  return { ...item, position }
}
