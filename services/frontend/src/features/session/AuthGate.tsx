import { useEffect, useState, type ReactNode } from 'react'
import { reportApiError } from '../../shared/api/errors'
import {
  DEFAULT_DEMO_TOKEN,
  ensureDemoAuth,
} from '../../shared/auth/demoAuth'
import { authApi } from '../auth/api'
import { readDemoUsers, removeDemoUserByToken } from '../demo/demoUsers'

type AuthStatus = 'loading' | 'ready' | 'error'

function isKnownJuryToken(token: string): boolean {
  if (token === DEFAULT_DEMO_TOKEN) return true
  return readDemoUsers().some((user) => user.token === token)
}

export function AuthGate({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>('loading')

  useEffect(() => {
    let cancelled = false

    void (async () => {
      try {
        await ensureDemoAuth(
          authApi.createUser,
          authApi.validateToken,
          removeDemoUserByToken,
          isKnownJuryToken,
        )
        if (!cancelled) setStatus('ready')
      } catch (error) {
        reportApiError(
          error,
          'Не удалось войти. Проверьте доступность сервисов',
        )
        if (!cancelled) setStatus('error')
      }
    })()

    return () => {
      cancelled = true
    }
  }, [])

  if (status === 'loading') {
    return (
      <div className="grid min-h-dvh place-items-center px-4 text-avito-muted">
        Авторизация…
      </div>
    )
  }

  if (status === 'error') {
    return (
      <div className="grid min-h-dvh place-items-center gap-3 p-6 text-center">
        <p className="max-w-sm text-avito-muted">
          Не удалось войти. Нажмите «Повторить» или обновите страницу.
        </p>
        <button
          type="button"
          className="min-h-11 cursor-pointer rounded-2xl bg-avito-blue px-5 py-2.5 font-extrabold text-white"
          onClick={() => window.location.reload()}
        >
          Повторить
        </button>
      </div>
    )
  }

  return children
}
