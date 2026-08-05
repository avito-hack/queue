import { api } from '../../shared/api/client'
const payOrder = async (ticketId: string) => {
  const response = await api.post(`/ticket/${ticketId}/pay`) 
  return response.data
}
export const ticketApi = { payOrder }