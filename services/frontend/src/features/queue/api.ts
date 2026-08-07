import { api } from '../../shared/api/client'
import { currentUserId } from '../../shared/auth/currentUser'
import type {
  ItemQueueInfo,
  ItemQueueStateResponse,
  UserPositionResponse,
} from './positionLib'
import type { QueueEntry } from './types'

const joinQueue = async (listingId: string): Promise<QueueEntry> => {
  // OpenAPI: POST /v1/queue/{itemID}/enqueue — без body, user из Bearer.
  await api.post(`/v1/queue/${listingId}/enqueue`)

  let position: number | undefined
  try {
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
    memberStatus: 'waiting_in_line',
  }
}

const leaveQueue = async (listingId: string) => {
  const response = await api.delete(`/v1/queue/${listingId}/dequeue`)
  return response.data
}

/** GET /v1/queue/{itemID}/position */
const getPosition = async (itemId: string): Promise<UserPositionResponse> => {
  const response = await api.get<UserPositionResponse>(
    `/v1/queue/${itemId}/position`,
  )
  return response.data
}

/**
 * GET /v1/queue/{itemID}/state
 * В контракте только enum state (нет length) — для UI «тикеты доступны/закончились».
 */
const getItemQueueState = async (
  itemId: string,
): Promise<ItemQueueStateResponse> => {
  const response = await api.get<ItemQueueStateResponse>(
    `/v1/queue/${itemId}/state`,
  )
  return response.data
}

/** GET /v1/user/queues — ItemQueueInfo[] */
const listUserQueues = async (): Promise<ItemQueueInfo[]> => {
  const response = await api.get<ItemQueueInfo[]>('/v1/user/queues')
  return Array.isArray(response.data) ? response.data : []
}

export const queueApi = {
  joinQueue,
  leaveQueue,
  getPosition,
  getItemQueueState,
  listUserQueues,
}
