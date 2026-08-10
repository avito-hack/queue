import { describe, expect, it } from 'vitest'
import { AxiosError } from 'axios'
import { getApiErrorMessage } from './errors'

function axiosError(status: number, data?: unknown, code?: string) {
  const error = new AxiosError(
    'fail',
    code,
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
  return error
}

describe('getApiErrorMessage', () => {
  it('maps known queue conflict messages to russian', () => {
    expect(
      getApiErrorMessage(
        axiosError(409, {
          code: 'conflict',
          message: 'user already in queue',
        }),
        'x',
      ),
    ).toBe('Вы уже стоите в очереди на этот товар')
  })

  it('maps auth jwt gibberish to clear russian', () => {
    expect(
      getApiErrorMessage(
        axiosError(401, {
          code: 'unauthorized',
          message:
            'validate token: token is malformed: token contains an invalid number of segments',
        }),
        'Не удалось встать в очередь',
      ),
    ).toBe('Сессия недействительна. Обновите страницу')
  })

  it('maps status fallbacks when body is empty', () => {
    expect(getApiErrorMessage(axiosError(409), 'x')).toBe(
      'Действие сейчас недоступно',
    )
    expect(getApiErrorMessage(axiosError(410), 'x')).toMatch(/недоступно/)
    expect(getApiErrorMessage(axiosError(422), 'x')).toMatch(/Нельзя/)
  })

  it('keeps human russian server message', () => {
    expect(
      getApiErrorMessage(
        axiosError(409, { message: 'Уже в очереди' }),
        'x',
      ),
    ).toBe('Уже в очереди')
  })

  it('uses network message without response', () => {
    const error = new AxiosError('Network Error', 'ERR_NETWORK')
    expect(getApiErrorMessage(error, 'Не удалось')).toMatch(/связи/)
  })

  it('uses fallback for unknown non-axios errors', () => {
    expect(getApiErrorMessage(new Error('nope'), 'Не удалось')).toBe(
      'Не удалось',
    )
  })

  it('maps avito conflict english text', () => {
    expect(
      getApiErrorMessage(
        axiosError(400, {
          message: 'operation conflicts with current state',
        }),
        'Не удалось авторизоваться',
      ),
    ).toBe('Действие конфликтует с текущим состоянием')
  })
})
