import { Link } from 'react-router-dom'

export function QueueEmpty() {
  return (
    <div className="rounded-2xl bg-white px-5 py-10 text-center text-avito-muted sm:p-8">
      Вы ещё не вставали в очередь.{' '}
      <Link to="/catalog" className="font-extrabold text-avito-blue">
        Перейти в каталог
      </Link>
    </div>
  )
}
