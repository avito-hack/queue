import { Link, NavLink } from 'react-router-dom'
import avitoLogo from '../../assets/avito-logo.svg'
import clockIcon from '../../assets/clock.svg'

const navClass = ({ isActive }: { isActive: boolean }) =>
  [
    'rounded-[10px] px-3.5 py-2.5 text-[13px] font-bold no-underline transition-colors sm:text-sm',
    isActive
      ? 'bg-[#f1f1f1] text-avito-ink'
      : 'bg-transparent text-[#4d4d4d] hover:bg-[#f1f1f1] hover:text-avito-ink',
  ].join(' ')

export function Header() {
  return (
    <header className="sticky top-0 z-30 border-b border-black/5 bg-white/95 backdrop-blur-md">
      <div className="mx-auto flex h-[68px] w-[min(1180px,calc(100%-22px))] items-center gap-2.5 sm:w-[min(1180px,calc(100%-32px))] sm:gap-6">
        <Link
          to="/catalog"
          className="inline-flex items-center gap-2.5 whitespace-nowrap text-[21px] font-extrabold tracking-tight text-avito-ink no-underline sm:text-2xl"
        >
          <img
            src={avitoLogo}
            alt=""
            aria-hidden="true"
            className="size-7 shrink-0 sm:size-8"
          />
          <span>avito</span>
          <span className="hidden text-sm font-bold tracking-normal text-[#777] lg:inline">
            лимитированные товары
          </span>
        </Link>

        <nav
          className="order-3 flex gap-1.5 sm:order-none"
          aria-label="Основная навигация"
        >
          <NavLink to="/catalog" className={navClass}>
            Каталог
          </NavLink>
          <NavLink to="/queue" className={navClass}>
            Мои очереди
          </NavLink>
        </nav>

        <div className="flex-1" />

        <Link
          to="/queue"
          className="ml-auto inline-flex items-center gap-2 rounded-full bg-avito-blue-soft px-3 py-2 text-sm font-bold text-[#0078bd] no-underline sm:ml-0"
        >
          <img
            src={clockIcon}
            alt=""
            aria-hidden="true"
            className="size-4 shrink-0"
          />
          <span className="sm:hidden">Очереди</span>
          <span className="hidden sm:inline">Мои очереди</span>
        </Link>

        <div
          className="hidden size-9 place-items-center rounded-full bg-avito-purple text-sm font-extrabold text-white sm:grid"
          aria-hidden="true"
        >
          Г
        </div>
      </div>
    </header>
  )
}
