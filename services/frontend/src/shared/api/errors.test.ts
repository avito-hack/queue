import { describe, expect, it } from 'vitest'
import { AxiosError } from 'axios'
import { getApiErrorMessage } from './errors'

function axiosError(status: number, data?: unknown) {
  return new AxiosError(
    'fail',
    String(status),
    undefined,
    undefined,
    {
      status,
      data,
      statusText: '',
      headers: {},
      config: {} as never,
    },
  )
}

describe('getApiErrorMessage', () => {
  it('maps 409/410/422 with fallbacks', () => {
    expect(getApiErrorMessage(axiosError(409), 'x')).toMatch(/Конфликт/)
    expect(getApiErrorMessage(axiosError(410), 'x')).toMatch(/недоступно/)
    expect(getApiErrorMessage(axiosError(422), 'x')).toMatch(/недоступ/)
  })

  it('prefers server message', () => {
    expect(
      getApiErrorMessage(
        axiosError(409, { message: 'Уже в очереди' }),
        'x',
      ),
    ).toBe('Уже в очереди')
  })

  it('uses fallback for unknown errors', () => {
    expect(getApiErrorMessage(new Error('nope'), 'Не удалось')).toBe(
      'Не удалось',
    )
  })
})
