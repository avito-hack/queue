import { api } from "../../shared/api/client"

const joinQueue = async (productId: string) => {
    const response = await api.post(`/queue/join`, { productId })
    return response.data
}

const leaveQueue = async (productId: string) => {
    const response = await api.post(`/queue/leave`, { productId })
    return response.data
}

export const queueApi = {
    joinQueue,
    leaveQueue,
}