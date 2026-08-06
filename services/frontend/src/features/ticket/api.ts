import { api } from '../../shared/api/client'

const listTickets = async () => {
  const response = await api.get('/v1/ticket/list')
  return response.data
}

const getTicket = async (ticketId: string) => {
  const response = await api.get(`/v1/ticket/${ticketId}`)
  return response.data
}

const activateTicket = async (ticketId: string) => {
  const response = await api.post(
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

const payOrder = async (ticketId: string) => {
  const response = await api.post(`/ticket/${ticketId}/pay`)
  return response.data
}

const declineTicket = async (ticketId: string) => {
  const response = await api.post(`/ticket/${ticketId}/decline`, undefined, {
    headers: {
      'Idempotency-Key': crypto.randomUUID(),
    },
  })
  return response.data
}

export const ticketApi = {
  listTickets,
  getTicket,
  activateTicket,
  payOrder,
  declineTicket,
}
