import {
  DEFAULT_DEMO_TOKEN,
  DEFAULT_DEMO_USER_ID,
  readStoredAuthToken,
} from '../../shared/auth/demoAuth'

export type DemoUser = {
  id: string
  token: string
  name: string
}

export const DEMO_USERS_KEY = 'demoUsers'
export const DEMO_USERS_EVENT = 'demo-users-changed'
export const MAX_DEMO_USERS = 10

const AUTH_TOKEN_KEY = 'authToken'
const DEMO_USER_ID_KEY = 'userId'

export function readDemoUsers(): DemoUser[] {
  try {
    const raw = localStorage.getItem(DEMO_USERS_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return []
    return parsed
      .filter(
        (item): item is DemoUser =>
          !!item &&
          typeof item === 'object' &&
          typeof (item as DemoUser).id === 'string' &&
          typeof (item as DemoUser).token === 'string' &&
          typeof (item as DemoUser).name === 'string',
      )
      .slice(0, MAX_DEMO_USERS)
  } catch {
    return []
  }
}

export function writeDemoUsers(users: DemoUser[]): void {
  localStorage.setItem(
    DEMO_USERS_KEY,
    JSON.stringify(users.slice(0, MAX_DEMO_USERS)),
  )
  window.dispatchEvent(new Event(DEMO_USERS_EVENT))
}

export function appendDemoUsers(users: DemoUser[]): DemoUser[] {
  const merged = [...readDemoUsers()]
  for (const user of users) {
    if (merged.some((item) => item.token === user.token || item.id === user.id)) {
      continue
    }
    if (merged.length >= MAX_DEMO_USERS) break
    merged.push(user)
  }
  writeDemoUsers(merged)
  return merged
}

export function removeDemoUserByToken(token: string): void {
  const current = readDemoUsers()
  const next = current.filter((user) => user.token !== token)
  if (next.length === current.length) return
  writeDemoUsers(next)
}

export function clearDemoUsers(): void {
  const active = readStoredAuthToken()
  localStorage.removeItem(DEMO_USERS_KEY)
  window.dispatchEvent(new Event(DEMO_USERS_EVENT))
  if (active && active !== DEFAULT_DEMO_TOKEN) {
    localStorage.setItem(DEMO_USER_ID_KEY, DEFAULT_DEMO_USER_ID)
    localStorage.setItem(AUTH_TOKEN_KEY, DEFAULT_DEMO_TOKEN)
    window.location.reload()
  }
}
