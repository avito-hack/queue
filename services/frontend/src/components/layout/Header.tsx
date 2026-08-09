import { Link, NavLink } from 'react-router-dom'
import avitoLogo from '../../assets/avito-logo.svg'
import clockIcon from '../../assets/clock.svg'

const navClass = ({ isActive }: { isActive: boolean }) =>
  [
    'inline-flex min-h-10 items-center rounded-xl px-3 py-2 text-sm font-bold no-underline transition-colors',
    isActive
      ? 'bg-[#f2f1f0] text-avito-ink'
      : 'bg-transparent text-[#5a5a5a] hover:bg-[#f2f1f0] hover:text-avito-ink',
  ].join(' ')

export function Header() {
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

        <Link
          to="/queue"
          className="inline-flex min-h-10 shrink-0 items-center gap-2 rounded-2xl bg-avito-blue-soft px-3 py-2 text-sm font-bold text-[#0078bd] no-underline sm:px-3.5"
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
      </div>
    </header>
  )
}
