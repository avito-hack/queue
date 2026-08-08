import { useEffect, useState, type ReactNode } from 'react'
import { reportApiError } from '../../shared/api/errors'
import { ensureDemoAuth } from '../../shared/auth/demoAuth'
import { authApi } from '../auth/api'

type AuthStatus = 'loading' | 'ready' | 'error'

export function AuthGate({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus>('loading')

  useEffect(() => {
    let cancelled = false

    void (async () => {
      try {
        await ensureDemoAuth(authApi.createUser)
        if (!cancelled) setStatus('ready')
      } catch (error) {
        reportApiError(error, 'Не удалось авторизоваться')
        if (!cancelled) setStatus('error')
      }
    })()

    return () => {
      cancelled = true
    }
  }, [])

  if (status === 'loading') {
    return (
      <div className="grid min-h-screen place-items-center text-avito-muted">
        Авторизация…
      </div>
    )
  }

  if (status === 'error') {
    return (
      <div className="grid min-h-screen place-items-center gap-3 p-6 text-center">
        <p className="text-avito-muted">
          Не удалось получить demo-токен / зарегистрировать пользователя.
        </p>
        <button
          type="button"
          className="cursor-pointer rounded-xl bg-avito-blue px-4 py-2 font-extrabold text-white"
          onClick={() => window.location.reload()}
        >
          Повторить
        </button>
      </div>
    )
  }

  return children
}
