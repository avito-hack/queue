import { useEffect, useState } from 'react'
import { formatCountdown } from './lib'

export function useCountdown(expiresAt?: string) {
  const [, setTick] = useState(0)

  useEffect(() => {
    if (!expiresAt) return

    const id = window.setInterval(() => {
      setTick((tick) => tick + 1)
    }, 1000)

    return () => window.clearInterval(id)
  }, [expiresAt])

  return formatCountdown(expiresAt)
}
