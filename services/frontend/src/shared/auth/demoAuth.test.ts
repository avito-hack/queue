import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  DEFAULT_DEMO_TOKEN,
  DEFAULT_DEMO_USER_ID,
  ensureDemoAuth,
  markAuthReady,
  resetAuthWaitState,
  waitForAuthToken,
} from './demoAuth'

describe('ensureDemoAuth', () => {
  beforeEach(() => {
    localStorage.clear()
    resetAuthWaitState()
  })

  it('uses seed opaque token and keeps seed user id', async () => {
    const createUser = vi.fn().mockResolvedValue({ id: 'random-avito-user' })

    const result = await ensureDemoAuth(createUser)

    expect(createUser).toHaveBeenCalledWith({
      name: 'Demo Buyer',
      token: DEFAULT_DEMO_TOKEN,
    })
    expect(result.token).toBe(DEFAULT_DEMO_TOKEN)
    expect(result.userId).toBe(DEFAULT_DEMO_USER_ID)
    expect(localStorage.getItem('authToken')).toBe(DEFAULT_DEMO_TOKEN)
    expect(localStorage.getItem('userId')).toBe(DEFAULT_DEMO_USER_ID)
  })

  it('replaces stored jwt with seed opaque token', async () => {
    localStorage.setItem('authToken', 'aaa.bbb.ccc')
    const createUser = vi.fn().mockResolvedValue({ id: 'x' })

    const result = await ensureDemoAuth(createUser)

    expect(result.token).toBe(DEFAULT_DEMO_TOKEN)
  })

  it('reuses stored opaque token', async () => {
    const opaque = '11111111-1111-4111-8111-111111111111'
    localStorage.setItem('authToken', opaque)
    localStorage.setItem('userId', 'custom-user')
    const createUser = vi.fn().mockResolvedValue({ id: 'created-user' })

    const result = await ensureDemoAuth(createUser)

    expect(result.token).toBe(opaque)
    expect(result.userId).toBe('created-user')
    expect(createUser).toHaveBeenCalledTimes(1)
  })

  it('stays ready with seed credentials when createUser fails', async () => {
    const createUser = vi.fn().mockRejectedValue(new Error('conflict'))

    const result = await ensureDemoAuth(createUser)

    expect(result.token).toBe(DEFAULT_DEMO_TOKEN)
    expect(result.userId).toBe(DEFAULT_DEMO_USER_ID)
  })

  it('unblocks waitForAuthToken after ensureDemoAuth', async () => {
    const pending = waitForAuthToken()
    const createUser = vi.fn().mockResolvedValue({ id: 'x' })

    await ensureDemoAuth(createUser)
    await expect(pending).resolves.toBe(DEFAULT_DEMO_TOKEN)
  })
})

describe('waitForAuthToken', () => {
  beforeEach(() => {
    localStorage.clear()
    resetAuthWaitState()
  })

  it('resolves immediately when markAuthReady was called', async () => {
    markAuthReady('opaque-token')
    await expect(waitForAuthToken()).resolves.toBe('opaque-token')
  })
})
