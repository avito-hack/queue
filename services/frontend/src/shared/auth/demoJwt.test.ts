import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  DEFAULT_DEMO_USER_ID,
  ensureDemoAuth,
  markAuthReady,
  resetAuthWaitState,
  signDemoJwt,
  waitForAuthToken,
} from './demoJwt'

describe('signDemoJwt', () => {
  it('returns a three-part HS256 token', async () => {
    const token = await signDemoJwt(DEFAULT_DEMO_USER_ID, 'test-secret')
    expect(token.split('.')).toHaveLength(3)
  })

  it('embeds user_id claim in payload', async () => {
    const token = await signDemoJwt(DEFAULT_DEMO_USER_ID, 'test-secret')
    const payloadPart = token.split('.')[1]
    expect(payloadPart).toBeTruthy()
    const json = atob(payloadPart!.replace(/-/g, '+').replace(/_/g, '/'))
    const payload = JSON.parse(json) as { user_id: string }
    expect(payload.user_id).toBe(DEFAULT_DEMO_USER_ID)
  })
})

describe('ensureDemoAuth', () => {
  beforeEach(() => {
    localStorage.clear()
    resetAuthWaitState()
    vi.stubEnv('VITE_JWT_SECRET', 'test-secret')
  })

  it('creates jwt, registers user, and stores token', async () => {
    const createUser = vi.fn().mockResolvedValue({ id: 'avito-user' })

    const result = await ensureDemoAuth(createUser)

    expect(createUser).toHaveBeenCalledWith({
      name: 'Demo User',
      token: result.token,
    })
    expect(result.userId).toBe(DEFAULT_DEMO_USER_ID)
    expect(localStorage.getItem('authToken')).toBe(result.token)
    expect(localStorage.getItem('userId')).toBe(DEFAULT_DEMO_USER_ID)
  })

  it('reuses existing jwt and still registers in avito', async () => {
    const token = await signDemoJwt(DEFAULT_DEMO_USER_ID, 'test-secret')
    localStorage.setItem('authToken', token)
    localStorage.setItem('userId', DEFAULT_DEMO_USER_ID)
    const createUser = vi.fn().mockResolvedValue({ id: 'avito-user' })

    const result = await ensureDemoAuth(createUser)

    expect(result.token).toBe(token)
    expect(createUser).toHaveBeenCalledTimes(1)
  })

  it('unblocks waitForAuthToken after ensureDemoAuth', async () => {
    const pending = waitForAuthToken()
    const createUser = vi.fn().mockResolvedValue({ id: 'avito-user' })

    await ensureDemoAuth(createUser)
    await expect(pending).resolves.toEqual(expect.any(String))
  })
})

describe('waitForAuthToken', () => {
  beforeEach(() => {
    localStorage.clear()
    resetAuthWaitState()
  })

  it('resolves immediately when markAuthReady was called', async () => {
    markAuthReady('aaa.bbb.ccc')
    await expect(waitForAuthToken()).resolves.toBe('aaa.bbb.ccc')
  })
})
