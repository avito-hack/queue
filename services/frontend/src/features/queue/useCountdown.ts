import { useEffect, useState } from 'react'
import { formatCountdown } from './lib'

export function useCountdown(expiresAt?: string) {
  const [label, setLabel] = useState(() => formatCountdown(expiresAt))

  useEffect(() => {
    setLabel(formatCountdown(expiresAt))

    if (!expiresAt) return

    const id = window.setInterval(() => {
      setLabel(formatCountdown(expiresAt))
    }, 1000)

    return () => window.clearInterval(id)
  }, [expiresAt])

  return label
}
