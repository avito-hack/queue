import type { QueueEntry, QueueStatus } from './types'

export const ACTIVE_QUEUE_STATUSES: QueueStatus[] = ['queued', 'ticket']

const ACTION_LABEL_BY_STATUS: Partial<Record<QueueStatus, string>> = {
  queued: 'Перейти к моим очередям',
  ticket: 'Перейти к покупке',
}

export function getProductActionLabel(
  inStock: boolean,
  entry?: QueueEntry | null,
) {
  if (entry) {
    return ACTION_LABEL_BY_STATUS[entry.status] ?? 'Перейти к моим очередям'
  }
  return inStock ? 'Встать в очередь' : 'Уведомить о поступлении'
}
