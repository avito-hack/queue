import { AxiosError } from 'axios'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  DEFAULT_DEMO_TOKEN,
  DEFAULT_DEMO_USER_ID,
  ensureDemoAuth,
  isUserAlreadyRegisteredError,
  markAuthReady,
  resetAuthWaitState,
  waitForAuthToken,
} from './demoAuth'

function conflictError() {
  return new AxiosError(
    'Request failed with status code 400',
    'ERR_BAD_REQUEST',
    undefined,
    undefined,
    {
      status: 400,
      statusText: 'Bad Request',
      headers: {},
      config: { headers: {} } as never,
      data: { message: 'operation conflicts with current state' },
    },
  )
}

describe('ensureDemoAuth', () => {
  beforeEach(() => {
    localStorage.clear()
    resetAuthWaitState()
  })

  it('validates existing token without create', async () => {
    const createUser = vi.fn()
    const validateToken = vi.fn().mockResolvedValue({
      user_id: DEFAULT_DEMO_USER_ID,
    })

    const result = await ensureDemoAuth(createUser, validateToken)

    expect(validateToken).toHaveBeenCalledWith(DEFAULT_DEMO_TOKEN)
    expect(createUser).not.toHaveBeenCalled()
    expect(result.token).toBe(DEFAULT_DEMO_TOKEN)
    expect(result.userId).toBe(DEFAULT_DEMO_USER_ID)
    expect(localStorage.getItem('authToken')).toBe(DEFAULT_DEMO_TOKEN)
    expect(localStorage.getItem('userId')).toBe(DEFAULT_DEMO_USER_ID)
  })

  it('creates default user when validate fails and no stored token', async () => {
    const createUser = vi.fn().mockResolvedValue({ id: 'random-avito-user' })
    const validateToken = vi.fn().mockRejectedValue(new Error('not found'))

    const result = await ensureDemoAuth(createUser, validateToken)

    expect(createUser).toHaveBeenCalledWith({
      name: 'Demo Buyer2',
      token: DEFAULT_DEMO_TOKEN,
    })
    expect(result.token).toBe(DEFAULT_DEMO_TOKEN)
    expect(result.userId).toBe(DEFAULT_DEMO_USER_ID)
  })

  it('replaces stored jwt with seed opaque token', async () => {
    localStorage.setItem('authToken', 'aaa.bbb.ccc')
    const createUser = vi.fn().mockResolvedValue({ id: 'x' })
    const validateToken = vi.fn().mockRejectedValue(new Error('not found'))

    const result = await ensureDemoAuth(createUser, validateToken)

    expect(result.token).toBe(DEFAULT_DEMO_TOKEN)
  })

  it('reuses stored opaque token via validate', async () => {
    const opaque = '11111111-1111-4111-8111-111111111111'
    localStorage.setItem('authToken', opaque)
    localStorage.setItem('userId', 'custom-user')
    const createUser = vi.fn()
    const validateToken = vi.fn().mockResolvedValue({ user_id: 'custom-user' })

    const result = await ensureDemoAuth(createUser, validateToken)

    expect(result.token).toBe(opaque)
    expect(result.userId).toBe('custom-user')
    expect(createUser).not.toHaveBeenCalled()
  })

  it('falls back to default when stored jury token is invalid', async () => {
    const opaque = '11111111-1111-4111-8111-111111111111'
    localStorage.setItem('authToken', opaque)
    const createUser = vi.fn()
    const onInvalidToken = vi.fn()
    const validateToken = vi
      .fn()
      .mockRejectedValueOnce(new Error('invalid bearer token'))
      .mockResolvedValueOnce({ user_id: DEFAULT_DEMO_USER_ID })

    const result = await ensureDemoAuth(
      createUser,
      validateToken,
      onInvalidToken,
      () => true,
    )

    expect(createUser).not.toHaveBeenCalled()
    expect(onInvalidToken).toHaveBeenCalledWith(opaque)
    expect(validateToken).toHaveBeenNthCalledWith(1, opaque)
    expect(validateToken).toHaveBeenNthCalledWith(2, DEFAULT_DEMO_TOKEN)
    expect(result.token).toBe(DEFAULT_DEMO_TOKEN)
    expect(result.userId).toBe(DEFAULT_DEMO_USER_ID)
  })

  it('skips validate for unknown jury token not in local list', async () => {
    const opaque = '11111111-1111-4111-8111-111111111111'
    localStorage.setItem('authToken', opaque)
    const createUser = vi.fn()
    const onInvalidToken = vi.fn()
    const validateToken = vi.fn().mockResolvedValue({
      user_id: DEFAULT_DEMO_USER_ID,
    })

    const result = await ensureDemoAuth(
      createUser,
      validateToken,
      onInvalidToken,
      () => false,
    )

    expect(onInvalidToken).toHaveBeenCalledWith(opaque)
    expect(validateToken).toHaveBeenCalledTimes(1)
    expect(validateToken).toHaveBeenCalledWith(DEFAULT_DEMO_TOKEN)
    expect(result.token).toBe(DEFAULT_DEMO_TOKEN)
  })

  it('treats create conflict as already registered', async () => {
    const createUser = vi.fn().mockRejectedValue(conflictError())
    const validateToken = vi.fn().mockRejectedValue(new Error('not found'))

    const result = await ensureDemoAuth(createUser, validateToken)

    expect(result.token).toBe(DEFAULT_DEMO_TOKEN)
    expect(result.userId).toBe(DEFAULT_DEMO_USER_ID)
    expect(localStorage.getItem('authToken')).toBe(DEFAULT_DEMO_TOKEN)
  })

  it('resolves user id via second validate on create conflict', async () => {
    const createUser = vi.fn().mockRejectedValue(conflictError())
    const validateToken = vi
      .fn()
      .mockRejectedValueOnce(new Error('not found'))
      .mockResolvedValueOnce({
        user_id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
      })

    const result = await ensureDemoAuth(createUser, validateToken)

    expect(createUser).toHaveBeenCalledTimes(1)
    expect(validateToken).toHaveBeenCalledTimes(2)
    expect(result.userId).toBe('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa')
  })

  it('fails when createUser rejects with non-conflict error', async () => {
    const createUser = vi.fn().mockRejectedValue(new Error('offline'))
    const validateToken = vi.fn().mockRejectedValue(new Error('not found'))

    await expect(ensureDemoAuth(createUser, validateToken)).rejects.toThrow(
      'offline',
    )
  })

  it('unblocks waitForAuthToken after ensureDemoAuth', async () => {
    const pending = waitForAuthToken()
    const createUser = vi.fn()
    const validateToken = vi.fn().mockResolvedValue({
      user_id: DEFAULT_DEMO_USER_ID,
    })

    await ensureDemoAuth(createUser, validateToken)
    await expect(pending).resolves.toBe(DEFAULT_DEMO_TOKEN)
  })
})

describe('isUserAlreadyRegisteredError', () => {
  it('detects avito conflict payload', () => {
    expect(isUserAlreadyRegisteredError(conflictError())).toBe(true)
    expect(isUserAlreadyRegisteredError(new Error('conflict'))).toBe(false)
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
