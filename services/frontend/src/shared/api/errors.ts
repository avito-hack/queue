import axios from 'axios'
import { showToast } from '../toast'

const MESSAGE_BY_CODE: Record<string, string> = {
  unauthorized: 'Нужна авторизация. Обновите страницу и попробуйте снова',
  not_found: 'Не найдено',
  conflict: 'Действие сейчас недоступно',
  ticket_not_activatable: 'Тикет уже активирован или недоступен',
  ticket_activation_expired: 'Время на активацию тикета истекло',
  idempotency_conflict: 'Повторный запрос конфликтует с предыдущим',
  activation_in_progress: 'Активация тикета уже выполняется',
  checkout_rejected: 'Не удалось создать оформление заказа',
  checkout_unavailable: 'Оформление временно недоступно',
  queue_unavailable: 'Очередь сейчас недоступна',
  bad_request: 'Некорректный запрос',
  internal_error: 'Внутренняя ошибка сервера. Попробуйте позже',
}

const MESSAGE_BY_SERVER_TEXT: Record<string, string> = {
  'invalid request': 'Некорректный запрос',
  'queue not found': 'Очередь для этого товара не найдена',
  'queue already exists': 'Очередь для этого товара уже существует',
  'queue unavailable': 'Очередь сейчас недоступна',
  'user already in queue': 'Вы уже стоите в очереди на этот товар',
  'user already has an active ticket':
    'У вас уже есть активное право на покупку',
  'user not found in queue': 'Вас нет в этой очереди',
  'user cannot leave queue': 'Сейчас нельзя выйти из очереди',
  'user position not found': 'Не удалось определить ваше место в очереди',
  'queue is empty': 'Очередь пуста',
  'invalid queue state': 'Очередь в неподходящем состоянии',
  'invalid member status': 'Некорректный статус в очереди',
  'internal error': 'Внутренняя ошибка сервера. Попробуйте позже',
  'resource not found': 'Ресурс не найден',
  'invalid bearer token': 'Сессия недействительна. Обновите страницу',
  'listing is unavailable': 'Товар недоступен',
  'insufficient available quantity': 'Недостаточно товара в наличии',
  'operation conflicts with current state':
    'Действие конфликтует с текущим состоянием',
  'bearer token is required': 'Нужна авторизация. Обновите страницу',
  'bearer token is invalid': 'Сессия недействительна. Обновите страницу',
  'missing authorization header': 'Нужна авторизация. Обновите страницу',
  'user identity service unavailable':
    'Сервис авторизации временно недоступен',
  'ticket not found': 'Тикет не найден',
  'invalid ticket issue': 'Не удалось выдать право на покупку',
  'ticket is not issuable': 'Сейчас нельзя выдать право на покупку',
  'invalid ticket decline': 'Не удалось отказаться от тикета',
  'ticket is not declinable': 'От этого тикета сейчас нельзя отказаться',
  'invalid ticket filter': 'Некорректный фильтр тикетов',
  'tickets unauthorized': 'Нет доступа к сервису тикетов',
}

const MESSAGE_BY_STATUS: Record<number, string> = {
  400: 'Некорректный запрос',
  401: 'Нужна авторизация. Обновите страницу и попробуйте снова',
  403: 'Недостаточно прав для этого действия',
  404: 'Не найдено',
  409: 'Действие сейчас недоступно',
  410: 'Больше недоступно: истекло или отозвано',
  422: 'Нельзя выполнить это действие сейчас',
  429: 'Слишком много запросов. Подождите немного',
  500: 'Ошибка сервера. Попробуйте позже',
  502: 'Сервис временно недоступен',
  503: 'Сервис временно недоступен',
  504: 'Сервис не ответил вовремя',
}

function asRecord(data: unknown): Record<string, unknown> | null {
  if (!data || typeof data !== 'object') return null
  return data as Record<string, unknown>
}

function readStringField(
  record: Record<string, unknown>,
  key: string,
): string | null {
  const value = record[key]
  if (typeof value !== 'string') return null
  const trimmed = value.trim()
  return trimmed || null
}

function looksTechnical(message: string): boolean {
  const lower = message.toLowerCase()
  return (
    lower.includes('openapi') ||
    lower.includes('validate token') ||
    lower.includes('token is malformed') ||
    lower.includes('invalid number of segments') ||
    lower.includes('jwt') ||
    lower.includes('request body has an error') ||
    lower.includes('failed to decode') ||
    lower.includes('sql:') ||
    lower.includes('pq:') ||
    lower.includes('dial tcp') ||
    lower.includes('connection refused') ||
    /^error in /i.test(message) ||
    /^[a-z0-9_./:-]+$/i.test(message)
  )
}

function looksHumanRussian(message: string): boolean {
  return /[а-яё]/i.test(message)
}

function translateServerMessage(message: string): string | null {
  const normalized = message.trim().toLowerCase()
  if (MESSAGE_BY_SERVER_TEXT[normalized]) {
    return MESSAGE_BY_SERVER_TEXT[normalized]
  }

  for (const [key, value] of Object.entries(MESSAGE_BY_SERVER_TEXT)) {
    if (normalized.includes(key)) return value
  }

  if (normalized.includes('malformed') || normalized.includes('segments')) {
    return 'Сессия недействительна. Обновите страницу'
  }
  if (normalized.includes('unauthorized')) {
    return MESSAGE_BY_STATUS[401] ?? null
  }

  return null
}

function messageFromResponse(data: unknown): string | null {
  const record = asRecord(data)
  if (!record) return null

  const code =
    readStringField(record, 'code') ?? readStringField(record, 'error')
  if (code && MESSAGE_BY_CODE[code]) {
    const raw = readStringField(record, 'message')
    if (raw) {
      const translated = translateServerMessage(raw)
      if (translated) return translated
      if (looksHumanRussian(raw) && !looksTechnical(raw)) return raw
    }
    return MESSAGE_BY_CODE[code]
  }

  const raw = readStringField(record, 'message')
  if (!raw) return null

  const translated = translateServerMessage(raw)
  if (translated) return translated
  if (looksHumanRussian(raw) && !looksTechnical(raw)) return raw
  return null
}

/** Человекочитаемый текст по статусу / телу ответа бэка. */
export function getApiErrorMessage(error: unknown, fallback: string): string {
  if (!axios.isAxiosError(error)) {
    if (error instanceof Error && looksHumanRussian(error.message)) {
      return error.message
    }
    return fallback
  }

  if (!error.response) {
    if (error.code === 'ECONNABORTED') {
      return 'Превышено время ожидания ответа сервера'
    }
    return 'Нет связи с сервером. Проверьте подключение'
  }

  const fromBody = messageFromResponse(error.response.data)
  if (fromBody) return fromBody

  const byStatus = MESSAGE_BY_STATUS[error.response.status]
  if (byStatus) return byStatus

  return fallback
}

/** Лог + toast в UI. */
export function reportApiError(error: unknown, fallback: string): void {
  console.error(error)
  showToast(getApiErrorMessage(error, fallback), 'error')
}
