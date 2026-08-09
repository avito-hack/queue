import { currentUserId } from '../../shared/auth/currentUser'

const STORAGE_KEY = 'paidTickets'

type PaidByUser = Record<string, string[]>

function readAll(): PaidByUser {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as PaidByUser
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function writeAll(value: PaidByUser) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(value))
}

export function isTicketPaid(
  ticketId: string,
  userId: string = currentUserId(),
): boolean {
  const id = ticketId.trim()
  if (!id) return false
  return (readAll()[userId] ?? []).includes(id)
}

export function markTicketPaid(
  ticketId: string,
  userId: string = currentUserId(),
): void {
  const id = ticketId.trim()
  if (!id) return
  const all = readAll()
  const current = all[userId] ?? []
  if (current.includes(id)) return
  all[userId] = [...current, id]
  writeAll(all)
}
