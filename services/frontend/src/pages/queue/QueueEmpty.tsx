import { Link } from 'react-router-dom'

export function QueueEmpty() {
  return (
    <div className="rounded-2xl bg-white p-8 text-center text-avito-muted">
      Вы ещё не вставали в очередь.{' '}
      <Link to="/catalog" className="font-extrabold text-[#008ed8]">
        Перейти в каталог
      </Link>
    </div>
  )
}
