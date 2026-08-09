const DEMO_USER_ID_KEY = 'userId'
const AUTH_TOKEN_KEY = 'authToken'
const DEMO_USER_NAME = 'Demo Buyer2'

export const DEFAULT_DEMO_USER_ID = '00000000-0000-4000-8000-000000000001'
export const DEFAULT_DEMO_TOKEN = DEFAULT_DEMO_USER_ID

type AuthWaiter = {
  resolve: (token: string) => void
  reject: (error: Error) => void
}

let authReadyToken: string | null = null
let authError: Error | null = null
let authWaiters: AuthWaiter[] = []

function looksLikeJwt(token: string): boolean {
  return token.split('.').length === 3
}

function readStoredUserId(): string {
  return localStorage.getItem(DEMO_USER_ID_KEY) ?? DEFAULT_DEMO_USER_ID
}

export function readStoredAuthToken(): string | null {
  const token = localStorage.getItem(AUTH_TOKEN_KEY)
  if (!token || looksLikeJwt(token)) return null
  return token
}

export function waitForAuthToken(): Promise<string> {
  const stored = readStoredAuthToken()
  if (stored) return Promise.resolve(stored)
  if (authReadyToken) return Promise.resolve(authReadyToken)
  if (authError) return Promise.reject(authError)

  return new Promise((resolve, reject) => {
    authWaiters.push({ resolve, reject })
  })
}

export function markAuthReady(token: string): void {
  authReadyToken = token
  authError = null
  localStorage.setItem(AUTH_TOKEN_KEY, token)
  const waiters = authWaiters
  authWaiters = []
  for (const waiter of waiters) waiter.resolve(token)
}

export function markAuthFailed(error: Error): void {
  authError = error
  authReadyToken = null
  const waiters = authWaiters
  authWaiters = []
  for (const waiter of waiters) waiter.reject(error)
}

export function resetAuthWaitState(): void {
  authReadyToken = null
  authError = null
  authWaiters = []
}

export async function ensureDemoAuth(
  createUser: (input: {
    name: string
    token: string
  }) => Promise<{ id: string }>,
): Promise<{ userId: string; token: string }> {
  try {
    const token = readStoredAuthToken() ?? DEFAULT_DEMO_TOKEN
    let userId = readStoredUserId()

    const user = await createUser({ name: DEMO_USER_NAME, token })
    if (token !== DEFAULT_DEMO_TOKEN && user.id) {
      userId = user.id
    } else {
      userId = DEFAULT_DEMO_USER_ID
    }

    localStorage.setItem(DEMO_USER_ID_KEY, userId)
    markAuthReady(token)
    return { userId, token }
  } catch (error) {
    const err = error instanceof Error ? error : new Error(String(error))
    markAuthFailed(err)
    throw err
  }
}
