import { useEffect, useState } from 'react'
import { Link, NavLink } from 'react-router-dom'
import { useAppSelector } from '../../app/hooks'
import {
  DEMO_USERS_EVENT,
  readDemoUsers,
  type DemoUser,
} from '../../features/demo/demoUsers'
import { switchJuryUser } from '../../features/demo/switchJuryUser'
import avitoLogo from '../../assets/avito-logo.svg'
import clockIcon from '../../assets/clock.svg'
import {
  DEFAULT_DEMO_TOKEN,
  DEFAULT_DEMO_USER_ID,
  readStoredAuthToken,
} from '../../shared/auth/demoAuth'
import { currentUserId } from '../../shared/auth/currentUser'
import { showToast } from '../../shared/toast'

const navClass = ({ isActive }: { isActive: boolean }) =>
  [
    'inline-flex min-h-10 items-center rounded-xl px-3 py-2 text-sm font-bold no-underline transition-colors',
    isActive
      ? 'bg-[#f2f1f0] text-avito-ink'
      : 'bg-transparent text-[#5a5a5a] hover:bg-[#f2f1f0] hover:text-avito-ink',
  ].join(' ')

function activeToken(): string {
  return readStoredAuthToken() ?? DEFAULT_DEMO_TOKEN
}

export function Header() {
  const myQueuesCount = useAppSelector(
    (state) =>
      state.queue.queueItems.filter((item) => item.status === 'queued').length,
  )
  const [demoUsers, setDemoUsers] = useState<DemoUser[]>(() => readDemoUsers())
  const [selectedToken, setSelectedToken] = useState(activeToken)

  useEffect(() => {
    const sync = () => {
      setDemoUsers(readDemoUsers())
      setSelectedToken(activeToken())
    }
    window.addEventListener(DEMO_USERS_EVENT, sync)
    window.addEventListener('storage', sync)
    return () => {
      window.removeEventListener(DEMO_USERS_EVENT, sync)
      window.removeEventListener('storage', sync)
    }
  }, [])

  const handleSwitch = (token: string) => {
    if (!token || token === activeToken()) return
    const user =
      token === DEFAULT_DEMO_TOKEN
        ? { id: DEFAULT_DEMO_USER_ID, token: DEFAULT_DEMO_TOKEN }
        : demoUsers.find((item) => item.token === token)
    if (!user) {
      setSelectedToken(activeToken())
      return
    }

    void (async () => {
      const result = await switchJuryUser(user)
      if (result === 'removed') {
        setDemoUsers(readDemoUsers())
        setSelectedToken(activeToken())
        showToast(
          'Этот демо-пользователь больше недоступен. Создайте заново.',
          'error',
        )
      }
    })()
  }

  return (
    <header className="sticky top-0 z-30 border-b border-black/6 bg-white/95 backdrop-blur-md">
      <div className="page-shell flex h-14 items-center gap-2 sm:h-[68px] sm:gap-4">
        <Link
          to="/catalog"
          className="inline-flex min-w-0 items-center gap-2 text-[20px] font-extrabold tracking-tight text-avito-ink no-underline sm:gap-2.5 sm:text-[22px]"
        >
          <img
            src={avitoLogo}
            alt=""
            aria-hidden="true"
            className="size-7 shrink-0 sm:size-8"
          />
          <span className="leading-none">avito</span>
          <span className="hidden truncate text-[13px] font-semibold tracking-normal text-avito-muted xl:inline">
            очередь
          </span>
        </Link>

        <nav
          className="ml-1 hidden items-center gap-1 md:flex"
          aria-label="Основная навигация"
        >
          <NavLink to="/catalog" className={navClass}>
            Каталог
          </NavLink>
        </nav>

        <div className="flex-1" />

        <label className="flex min-w-0 items-center gap-1.5">
          <span className="hidden shrink-0 text-[11px] font-bold text-avito-muted sm:inline">
            Пользователи
          </span>
          <select
            className="max-w-[140px] min-h-10 truncate rounded-xl border border-black/8 bg-[#f7f7f7] px-2 text-[12px] font-bold text-avito-ink outline-none focus:border-[#00aaff] sm:max-w-[180px] sm:px-2.5"
            value={selectedToken}
            onChange={(event) => handleSwitch(event.target.value)}
            aria-label="Переключить демо-пользователя"
          >
            <option value={DEFAULT_DEMO_TOKEN}>
              Default User · {DEFAULT_DEMO_USER_ID.slice(0, 8)}
            </option>
            {demoUsers.map((user, index) => (
              <option key={user.token} value={user.token}>
                {index + 1}. {user.name}
              </option>
            ))}
          </select>
        </label>

        <Link
          to="/queue"
          className="inline-flex min-h-10 shrink-0 items-center gap-2 rounded-2xl bg-avito-blue-soft px-3 py-2 text-sm font-bold text-[#0078bd] no-underline sm:px-3.5"
          aria-label={
            myQueuesCount > 0
              ? `Мои очереди, ${myQueuesCount}`
              : 'Мои очереди'
          }
          title={`user: ${currentUserId()}`}
        >
          <img
            src={clockIcon}
            alt=""
            aria-hidden="true"
            className="size-4 shrink-0"
          />
          <span className="sm:hidden">Очереди</span>
          <span className="hidden sm:inline">Мои очереди</span>
          {myQueuesCount > 0 && (
            <span className="inline-flex min-w-5 items-center justify-center rounded-full bg-[#0078bd] px-1.5 py-0.5 text-[11px] leading-none font-extrabold text-white">
              {myQueuesCount > 99 ? '99+' : myQueuesCount}
            </span>
          )}
        </Link>
      </div>
    </header>
  )
}
