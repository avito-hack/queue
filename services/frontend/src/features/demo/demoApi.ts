import { api } from '../../shared/api/client'
import { authApi } from '../auth/api'
import {
  mapListingToProduct,
  type ListingDto,
} from '../product/mapListing'
import type { Product } from '../product/types'
import type { UserPositionResponse } from '../queue/positionLib'
import type { TicketListItemDto, TicketListResponse } from '../ticket/api'
import {
  appendDemoUsers,
  MAX_DEMO_USERS,
  readDemoUsers,
  type DemoUser,
} from './demoUsers'
import {
  DEFAULT_DEMO_TOKEN,
  DEFAULT_DEMO_USER_ID,
  isUserAlreadyRegisteredError,
} from '../../shared/auth/demoAuth'

function asBearerConfig(userToken: string) {
  const token = userToken.trim()
  return {
    skipAuth: true as const,
    headers: {
      Authorization: `Bearer ${token}`,
    },
  }
}

const enqueueAsUser = async (
  listingId: string,
  userToken: string,
): Promise<{ position?: number }> => {
  await api.post(
    `/v1/queue/${listingId}/enqueue`,
    undefined,
    asBearerConfig(userToken),
  )

  try {
    const response = await api.get<UserPositionResponse>(
      `/v1/queue/${listingId}/position`,
      asBearerConfig(userToken),
    )
    if (typeof response.data.position === 'number') {
      return { position: response.data.position }
    }
  } catch {
    // позиция опциональна
  }

  return {}
}

const dequeueAsUser = async (listingId: string, userToken: string) => {
  await api.delete(`/v1/queue/${listingId}/dequeue`, asBearerConfig(userToken))
}

const listTicketsAsUser = async (
  userToken: string,
): Promise<TicketListResponse> => {
  const response = await api.get<TicketListResponse>(
    '/v1/ticket/list',
    asBearerConfig(userToken),
  )
  return response.data
}

const invalidateTicketAsUser = async (
  ticketId: string,
  userToken: string,
) => {
  await api.post(
    `/v1/ticket/${ticketId}/decline`,
    undefined,
    {
      ...asBearerConfig(userToken),
      headers: {
        ...asBearerConfig(userToken).headers,
        'Idempotency-Key': crypto.randomUUID(),
      },
    },
  )
}

async function resolveUserId(userToken: string): Promise<string> {
  const token = userToken.trim()
  if (token === DEFAULT_DEMO_TOKEN) return DEFAULT_DEMO_USER_ID
  const known = readDemoUsers().find((user) => user.token === token)
  if (known) return known.id
  const validated = await authApi.validateToken(token)
  if (!validated.user_id) {
    throw new Error('Не удалось определить user_id по токену')
  }
  return validated.user_id
}

const issueTicketForUser = async (
  listingId: string,
  userToken: string,
): Promise<TicketListItemDto> => {
  const userId = await resolveUserId(userToken)
  const response = await api.post<TicketListItemDto>(
    '/internal/v1/ticket/issue',
    {
      queue_entry_id: crypto.randomUUID(),
      user_id: userId,
      listing_id: listingId,
      sku_id: listingId,
    },
    {
      skipAuth: true,
      headers: {
        'Idempotency-Key': crypto.randomUUID(),
      },
    },
  )
  return response.data
}

const changeListingQuantity = async (
  listingId: string,
  quantity: number,
): Promise<Product> => {
  const response = await api.put<ListingDto>(
    `/v1/avito/listings/${listingId}/quantity`,
    { quantity },
    { skipAuth: true },
  )
  const product = mapListingToProduct(response.data)
  if (!product) {
    throw new Error('Invalid listing after quantity change')
  }
  return product
}

const registerDemoUsers = async (requestedCount: number): Promise<{
  created: DemoUser[]
  all: DemoUser[]
}> => {
  const existing = readDemoUsers()
  const slotsLeft = MAX_DEMO_USERS - existing.length
  if (slotsLeft <= 0) {
    return { created: [], all: existing }
  }

  const count = Math.min(
    slotsLeft,
    Math.max(1, Math.min(MAX_DEMO_USERS, Math.floor(requestedCount))),
  )
  const created: DemoUser[] = []

  for (let index = 0; index < count; index += 1) {
    let attempt = 0
    while (attempt < 3) {
      attempt += 1
      const token = crypto.randomUUID()
      const name = `Demo Buyer ${existing.length + created.length + 1}`
      try {
        const user = await authApi.createUser({ name, token })
        created.push({
          id: user.id,
          token,
          name: user.name || name,
        })
        break
      } catch (error) {
        if (isUserAlreadyRegisteredError(error) && attempt < 3) {
          continue
        }
        throw error
      }
    }
  }

  const all = appendDemoUsers(created)
  return { created, all }
}

export const demoApi = {
  enqueueAsUser,
  dequeueAsUser,
  listTicketsAsUser,
  invalidateTicketAsUser,
  issueTicketForUser,
  changeListingQuantity,
  registerDemoUsers,
}
