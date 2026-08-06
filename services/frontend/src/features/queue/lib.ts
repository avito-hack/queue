import type { TicketEntry } from '../ticket/types'
import type { QueueEntry } from './types'

export function getProductActionLabel(
  inStock: boolean,
  queueEntry?: QueueEntry | null,
  ticket?: TicketEntry | null,
) {
  if (ticket) return 'Перейти к покупке'
  if (queueEntry?.status === 'queued') return 'Перейти к моим очередям'
  return inStock ? 'Встать в очередь' : 'Уведомить о поступлении'
}

export function formatCountdown(expiresAt?: string): string | null {
  if (!expiresAt) return null

  const diffMs = new Date(expiresAt).getTime() - Date.now()
  if (Number.isNaN(diffMs) || diffMs <= 0) return '00:00'

  const totalSec = Math.floor(diffMs / 1000)
  const hours = Math.floor(totalSec / 3600)
  const minutes = Math.floor((totalSec % 3600) / 60)
  const seconds = totalSec % 60

  const mm = String(minutes).padStart(2, '0')
  const ss = String(seconds).padStart(2, '0')

  if (hours > 0) {
    return `${String(hours).padStart(2, '0')}:${mm}:${ss}`
  }

  return `${mm}:${ss}`
}
