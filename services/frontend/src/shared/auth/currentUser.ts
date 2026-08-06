const DEMO_USER_ID = '00000000-0000-4000-8000-000000000001'

export type EnqueueUser = {
  id: string
  first_name: string
  last_name: string
  has_ticket: boolean
}

export function currentUserId(): string {
  return localStorage.getItem('authToken') ?? DEMO_USER_ID
}

export function currentEnqueueUser(): EnqueueUser {
  return {
    id: currentUserId(),
    first_name: localStorage.getItem('userFirstName') ?? 'Demo',
    last_name: localStorage.getItem('userLastName') ?? 'User',
    has_ticket: false,
  }
}
