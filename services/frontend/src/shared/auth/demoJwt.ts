const DEMO_USER_ID_KEY = 'userId'
const AUTH_TOKEN_KEY = 'authToken'
const DEMO_USER_NAME = 'Demo User1'

export const DEFAULT_DEMO_USER_ID = '00000000-0000-4000-8000-000000000001'

type AuthWaiter = {
  resolve: (token: string) => void
  reject: (error: Error) => void
}

let authReadyToken: string | null = null
let authError: Error | null = null
let authWaiters: AuthWaiter[] = []

export function getDemoJwtSecret(): string {
  const secret = import.meta.env.VITE_JWT_SECRET
  if (typeof secret !== 'string' || !secret.trim()) {
    throw new Error('VITE_JWT_SECRET is required')
  }
  return secret
}

function bytesToBase64Url(bytes: Uint8Array): string {
  let binary = ''
  for (let i = 0; i < bytes.length; i += 1) {
    binary += String.fromCharCode(bytes[i] ?? 0)
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

function textToBase64Url(text: string): string {
  return bytesToBase64Url(new TextEncoder().encode(text))
}

export function looksLikeJwt(token: string): boolean {
  return token.split('.').length === 3
}

export async function signDemoJwt(
  userId: string,
  secret: string,
  expiresInSec = 60 * 60 * 24 * 30,
): Promise<string> {
  const header = { alg: 'HS256', typ: 'JWT' }
  const now = Math.floor(Date.now() / 1000)
  const payload = {
    user_id: userId,
    exp: now + expiresInSec,
    iat: now,
  }

  const body = `${textToBase64Url(JSON.stringify(header))}.${textToBase64Url(JSON.stringify(payload))}`
  const key = await crypto.subtle.importKey(
    'raw',
    new TextEncoder().encode(secret),
    { name: 'HMAC', hash: 'SHA-256' },
    false,
    ['sign'],
  )
  const signature = await crypto.subtle.sign(
    'HMAC',
    key,
    new TextEncoder().encode(body),
  )
  return `${body}.${bytesToBase64Url(new Uint8Array(signature))}`
}

function readStoredUserId(): string {
  return localStorage.getItem(DEMO_USER_ID_KEY) ?? DEFAULT_DEMO_USER_ID
}

export function readStoredAuthToken(): string | null {
  const token = localStorage.getItem(AUTH_TOKEN_KEY)
  return token && looksLikeJwt(token) ? token : null
}

/** Ждёт, пока ensureDemoAuth положит JWT — для axios interceptor. */
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

/** Сброс для тестов. */
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
    const userId = readStoredUserId()
    const stored = readStoredAuthToken()
    const token = stored ?? (await signDemoJwt(userId, getDemoJwtSecret()))

    await createUser({ name: DEMO_USER_NAME, token })
    localStorage.setItem(DEMO_USER_ID_KEY, userId)
    markAuthReady(token)
    return { userId, token }
  } catch (error) {
    const err = error instanceof Error ? error : new Error(String(error))
    markAuthFailed(err)
    throw err
  }
}
