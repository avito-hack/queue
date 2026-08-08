import { currentUserId } from '../../shared/auth/currentUser'
import type {
  ItemQueueMemberStatus,
  ItemQueueState,
  QueueEntry,
  QueueStatus,
} from './types'

/** GET /v1/queue/{itemID}/position */
export type UserPositionResponse = {
  position: number
}

/** GET /v1/queue/{itemID}/state — в OpenAPI нет длины очереди, только state. */
export type ItemQueueStateResponse = {
  state: ItemQueueState
}

/** Элемент GET /v1/user/queues (schema ItemQueueInfo). */
export type ItemQueueInfo = {
  item_id: string
  position: number
  status: ItemQueueMemberStatus
}

/** @deprecated используй ItemQueueInfo */
export type UserQueueInfo = Partial<ItemQueueInfo> & {
  item_id?: string
  position?: number
  status?: string
}

export type QueueUserQueuesSync = {
  updates: QueueEntry[]
  removeProductIds: string[]
}

function queueStatusFromMemberStatus(
  status?: string,
): QueueStatus | 'remove' | null {
  if (status === 'waiting_in_line') return 'queued'
  if (status === 'item_out_of_stock') return 'soldout'
  if (
    status === 'acquired_purchase_rights' ||
    status === 'placed_an_order' ||
    status === 'purchased_an_item' ||
    status === 'voluntarily_left_the_line' ||
    status === 'given_up_purchase_rights' ||
    status === 'lost_purchase_rights'
  ) {
    return 'remove'
  }
  return null
}

/**
 * Обновить локальные queued/soldout плитки по массиву /v1/user/queues.
 * Терминальные статусы → removeProductIds (тикет/выход из линии).
 */
export function queueItemsFromUserQueues(
  queueItems: QueueEntry[],
  infos: UserQueueInfo[],
): QueueUserQueuesSync {
  const updates: QueueEntry[] = []
  const removeProductIds: string[] = []

  for (const item of queueItems) {
    if (item.status !== 'queued' && item.status !== 'soldout') continue

    const info = infos.find((row) => row.item_id === item.productId)
    if (!info) continue

    const nextStatus = queueStatusFromMemberStatus(info.status)
    if (!nextStatus) continue

    if (nextStatus === 'remove') {
      if (!removeProductIds.includes(item.productId)) {
        removeProductIds.push(item.productId)
      }
      continue
    }

    const next: QueueEntry = {
      ...item,
      status: nextStatus,
      memberStatus: info.status as ItemQueueMemberStatus | undefined,
      position:
        nextStatus === 'queued' && typeof info.position === 'number'
          ? info.position
          : undefined,
    }

    if (
      next.position !== item.position ||
      next.status !== item.status ||
      next.memberStatus !== item.memberStatus
    ) {
      updates.push(next)
    }
  }

  return { updates, removeProductIds }
}

/**
 * Построить плитки очереди с нуля из GET /v1/user/queues (hydrate).
 * Терминальные статусы пропускаем — их закрывают тикеты / отсутствие в линии.
 */
export function queueEntriesFromUserQueues(
  infos: UserQueueInfo[],
): QueueEntry[] {
  const userId = currentUserId()
  const entries: QueueEntry[] = []

  for (const info of infos) {
    if (!info.item_id) continue
    const status = queueStatusFromMemberStatus(info.status)
    if (status !== 'queued' && status !== 'soldout') continue

    entries.push({
      id: `${info.item_id}-${userId}`,
      productId: info.item_id,
      status,
      memberStatus: info.status as ItemQueueMemberStatus | undefined,
      position:
        status === 'queued' && typeof info.position === 'number'
          ? info.position
          : undefined,
    })
  }

  return entries
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
