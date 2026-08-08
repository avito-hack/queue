import axios from 'axios'
import { showToast } from '../toast'

function serverMessage(data: unknown): string | null {
  if (!data || typeof data !== 'object') return null
  const record = data as Record<string, unknown>
  if (typeof record.message === 'string' && record.message.trim()) {
    return record.message
  }
  if (typeof record.error === 'string' && record.error.trim()) {
    return record.error
  }
  return null
}

/** Человекочитаемый текст по статусу / телу ответа бэка. */
export function getApiErrorMessage(error: unknown, fallback: string): string {
  if (!axios.isAxiosError(error)) {
    return fallback
  }

  const status = error.response?.status
  const fromServer = serverMessage(error.response?.data)

  switch (status) {
    case 401:
      return fromServer ?? 'Нужна авторизация'
    case 404:
      return fromServer ?? 'Не найдено'
    case 409:
      return fromServer ?? 'Конфликт: действие сейчас недоступно'
    case 410:
      return fromServer ?? 'Больше недоступно (истекло или отозвано)'
    case 422:
      return fromServer ?? 'Очередь или действие недоступны'
    case 502:
    case 503:
      return fromServer ?? 'Сервис временно недоступен'
    default:
      return fromServer ?? fallback
  }
}

/** Лог + toast в UI. */
export function reportApiError(error: unknown, fallback: string): void {
  console.error(error)
  showToast(getApiErrorMessage(error, fallback), 'error')
}
