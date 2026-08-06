import { api } from '../../shared/api/client'

const joinQueue = async (productId: string) => {
  const response = await api.post(`/queue/join`, { productId })
  return response.data
}

const leaveQueue = async (productId: string) => {
  const response = await api.post(`/queue/leave`, { productId })
  return response.data
}

const getPosition = async (userId: string) => {
  const response = await api.get(`/v1/queue/${userId}/position`)
  return response.data
}

export const queueApi = {
  joinQueue,
  leaveQueue,
  getPosition,
}
