import { api } from '../../shared/api/client'

export type CreateUserResponse = {
  id: string
  name: string
  createdAt: string
}

const createUser = async (input: {
  name: string
  token: string
}): Promise<CreateUserResponse> => {
  const response = await api.post<CreateUserResponse>('/v1/avito/users', input, {
    skipAuth: true,
  })
  return response.data
}

export const authApi = {
  createUser,
}
