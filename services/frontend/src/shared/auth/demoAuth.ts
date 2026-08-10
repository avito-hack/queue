import axios from 'axios'

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

export function switchActiveUser(user: { id: string; token: string }): void {
  localStorage.setItem(DEMO_USER_ID_KEY, user.id)
  markAuthReady(user.token)
  window.location.reload()
}

export function isUserAlreadyRegisteredError(error: unknown): boolean {
  if (!axios.isAxiosError(error)) return false
  const status = error.response?.status
  if (status !== 400 && status !== 409) return false
  const data = error.response?.data
  if (!data || typeof data !== 'object') return false
  const message = (data as { message?: unknown }).message
  return (
    typeof message === 'string' &&
    message.toLowerCase().includes('conflicts with current state')
  )
}

function finishAuth(token: string, userId: string): { userId: string; token: string } {
  localStorage.setItem(DEMO_USER_ID_KEY, userId)
  markAuthReady(token)
  return { userId, token }
}

export async function ensureDemoAuth(
  createUser: (input: {
    name: string
    token: string
  }) => Promise<{ id: string }>,
  validateToken?: (token: string) => Promise<{ user_id: string }>,
  onInvalidToken?: (token: string) => void,
  isKnownToken?: (token: string) => boolean,
): Promise<{ userId: string; token: string }> {
  let storedToken = readStoredAuthToken()

  if (
    storedToken &&
    storedToken !== DEFAULT_DEMO_TOKEN &&
    isKnownToken &&
    !isKnownToken(storedToken)
  ) {
    onInvalidToken?.(storedToken)
    localStorage.setItem(DEMO_USER_ID_KEY, DEFAULT_DEMO_USER_ID)
    localStorage.setItem(AUTH_TOKEN_KEY, DEFAULT_DEMO_TOKEN)
    storedToken = DEFAULT_DEMO_TOKEN
  }

  const candidates =
    storedToken && storedToken !== DEFAULT_DEMO_TOKEN
      ? [storedToken, DEFAULT_DEMO_TOKEN]
      : [storedToken ?? DEFAULT_DEMO_TOKEN]

  if (validateToken) {
    for (const token of candidates) {
      try {
        const validated = await validateToken(token)
        if (validated.user_id) {
          return finishAuth(token, validated.user_id)
        }
      } catch {
        if (token !== DEFAULT_DEMO_TOKEN) {
          onInvalidToken?.(token)
        }
      }
    }
  } else if (storedToken) {
    return finishAuth(storedToken, readStoredUserId())
  }

  const token = DEFAULT_DEMO_TOKEN

  try {
    const user = await createUser({ name: DEMO_USER_NAME, token })
    const userId = user.id || DEFAULT_DEMO_USER_ID
    return finishAuth(token, token === DEFAULT_DEMO_TOKEN ? DEFAULT_DEMO_USER_ID : userId)
  } catch (error) {
    if (!isUserAlreadyRegisteredError(error)) {
      const err = error instanceof Error ? error : new Error(String(error))
      markAuthFailed(err)
      throw err
    }

    if (validateToken) {
      try {
        const validated = await validateToken(token)
        if (validated.user_id) {
          return finishAuth(token, validated.user_id)
        }
      } catch {
        // fallback ниже
      }
    }

    return finishAuth(token, DEFAULT_DEMO_USER_ID)
  }
}
