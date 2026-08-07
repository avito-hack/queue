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

export function isTicketExpired(expiresAt?: string): boolean {
  if (!expiresAt) return false
  const endsAt = new Date(expiresAt).getTime()
  if (Number.isNaN(endsAt)) return false
  return endsAt <= Date.now()
}

export function formatCountdown(expiresAt?: string): string | null {
  if (!expiresAt) return null

  const diffMs = new Date(expiresAt).getTime() - Date.now()
  if (Number.isNaN(diffMs) || diffMs <= 0) return '00:00'

  const totalSec = Math.floor(diffMs / 1000)
  const days = Math.floor(totalSec / 86400)
  const hours = Math.floor((totalSec % 86400) / 3600)
  const minutes = Math.floor((totalSec % 3600) / 60)
  const seconds = totalSec % 60

  const hh = String(hours).padStart(2, '0')
  const mm = String(minutes).padStart(2, '0')
  const ss = String(seconds).padStart(2, '0')

  if (days > 0) {
    return `${days}д ${hh}:${mm}:${ss}`
  }
  if (hours > 0) {
    return `${hh}:${mm}:${ss}`
  }

  return `${mm}:${ss}`
}
