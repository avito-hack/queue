import { api } from '../../shared/api/client'

export type ActivateTicketResponse = {
  ticket_id?: string
  status?: string
  order_id?: string
  checkout_url?: string
}

/** Элемент списка — поля, которые маппит toTicketEntry. */
export type TicketListItemDto = {
  id?: string
  listing_id?: string
  status?: string
  activation_deadline?: string
  available_actions?: string[]
  checkout_url?: string | null
  finished_at?: string | null
}

export type TicketListResponse = {
  ticket?: TicketListItemDto[]
}

const listTickets = async (): Promise<TicketListResponse> => {
  const response = await api.get<TicketListResponse>('/v1/ticket/list')
  return response.data
}

const getTicket = async (ticketId: string) => {
  const response = await api.get(`/v1/ticket/${ticketId}`)
  return response.data
}

const activateTicket = async (
  ticketId: string,
): Promise<ActivateTicketResponse> => {
  const response = await api.post<ActivateTicketResponse>(
    `/v1/ticket/${ticketId}/activate`,
    undefined,
    {
      headers: {
        'Idempotency-Key': crypto.randomUUID(),
      },
    },
  )
  return response.data
}

const declineTicket = async (ticketId: string) => {
  const response = await api.post(
    `/v1/ticket/${ticketId}/decline`,
    undefined,
    {
      headers: {
        'Idempotency-Key': crypto.randomUUID(),
      },
    },
  )
  return response.data
}

export const ticketApi = {
  listTickets,
  getTicket,
  activateTicket,
  declineTicket,
}
