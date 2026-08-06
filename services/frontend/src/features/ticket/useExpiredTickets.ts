import { useEffect, useRef } from 'react'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { isTicketExpired } from '../queue/lib'
import { removeTicket } from './ticketSlice'

export function useExpiredTickets(enabled = true) {
  const dispatch = useAppDispatch()
  const ticketItems = useAppSelector((state) => state.tickets.ticketItems)
  const hasTickets = ticketItems.length > 0
  const ticketsRef = useRef(ticketItems)
  ticketsRef.current = ticketItems

  useEffect(() => {
    if (!enabled || !hasTickets) return

    const sync = () => {
      for (const ticket of ticketsRef.current) {
        if (isTicketExpired(ticket.expiresAt)) {
          dispatch(removeTicket(ticket.id))
        }
      }
    }

    sync()
    const id = window.setInterval(sync, 1000)

    return () => {
      window.clearInterval(id)
    }
  }, [dispatch, enabled, hasTickets])
}
