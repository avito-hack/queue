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

export function formatCountdown(expiresAt?: string): string | null {
  if (!expiresAt) return null

  const diffMs = new Date(expiresAt).getTime() - Date.now()
  if (Number.isNaN(diffMs) || diffMs <= 0) return '00:00'

  const totalSec = Math.floor(diffMs / 1000)
  const minutes = Math.floor(totalSec / 60)
  const seconds = totalSec % 60

  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}
