import { api } from '../../shared/api/client'

const payOrder = async (ticketId: string) => {
  const response = await api.post(`/ticket/${ticketId}/pay`)
  return response.data
}

const declineTicket = async (ticketId: string) => {
  const response = await api.post(
    `/ticket/${ticketId}/decline`,
    undefined,
    {
      headers: {
        'Idempotency-Key': crypto.randomUUID(),
      },
    },
  )
  return response.data
}

export const ticketApi = { payOrder, declineTicket }
