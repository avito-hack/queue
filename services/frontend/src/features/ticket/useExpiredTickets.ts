import { useEffect, useEffectEvent } from 'react'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { isTicketExpired } from '../queue/lib'
import { removeTicket } from './ticketSlice'

export function useExpiredTickets(enabled = true) {
  const dispatch = useAppDispatch()
  const ticketItems = useAppSelector((state) => state.tickets.ticketItems)
  const hasTickets = ticketItems.length > 0

  const sync = useEffectEvent(() => {
    for (const ticket of ticketItems) {
      if (isTicketExpired(ticket.expiresAt)) {
        dispatch(removeTicket(ticket.id))
      }
    }
  })

  useEffect(() => {
    if (!enabled || !hasTickets) return

    sync()
    const id = window.setInterval(sync, 1000)

    return () => {
      window.clearInterval(id)
    }
  }, [enabled, hasTickets])
}
