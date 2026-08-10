import { DEFAULT_DEMO_USER_ID } from './demoAuth'

export type EnqueueUser = {
  id: string
  first_name: string
  last_name: string
  has_ticket: boolean
}

export function currentUserId(): string {
  return localStorage.getItem('userId') ?? DEFAULT_DEMO_USER_ID
}

export function currentAuthToken(): string | null {
  return localStorage.getItem('authToken')
}

export function currentEnqueueUser(): EnqueueUser {
  return {
    id: currentUserId(),
    first_name: localStorage.getItem('userFirstName') ?? 'Demo',
    last_name: localStorage.getItem('userLastName') ?? 'User',
    has_ticket: false,
  }
}
