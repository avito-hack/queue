import { api } from '../../shared/api/client'
import { currentEnqueueUser, currentUserId } from '../../shared/auth/currentUser'
import type { UserPositionResponse, UserQueueInfo } from './positionLib'
import type { QueueEntry } from './types'

type EnqueueResponse = {
  status?: string
}

const joinQueue = async (listingId: string): Promise<QueueEntry> => {
  await api.post<EnqueueResponse>('/v1/queue/enqueue', {
    id: listingId,
    user: currentEnqueueUser(),
  })

  let position: number | undefined
  try {
    // позиция в очереди на этот товар (itemID), не userID
    const data = await getPosition(listingId)
    if (typeof data.position === 'number') {
      position = data.position
    }
  } catch {
    // позиция подтянется polling'ом /v1/user/queues
  }

  return {
    id: `${listingId}-${currentUserId()}`,
    productId: listingId,
    status: 'queued',
    position,
  }
}

const leaveQueue = async (_listingId: string) => {
  const response = await api.delete('/v1/queue/dequeue')
  return response.data
}

/** GET /v1/queue/{itemID}/position — одна очередь (товар/SKU). */
const getPosition = async (itemId: string): Promise<UserPositionResponse> => {
  const response = await api.get<UserPositionResponse>(
    `/v1/queue/${itemId}/position`,
  )
  return response.data
}

/** GET /v1/user/queues — все очереди текущего пользователя (из Bearer). */
const listUserQueues = async (): Promise<UserQueueInfo[]> => {
  const response = await api.get<UserQueueInfo[]>('/v1/user/queues')
  return Array.isArray(response.data) ? response.data : []
}

export const queueApi = {
  joinQueue,
  leaveQueue,
  getPosition,
  listUserQueues,
}
