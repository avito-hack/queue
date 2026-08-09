import { api } from '../../shared/api/client'

export type CreateUserResponse = {
  id: string
  name: string
  createdAt: string
}

export type ValidateUserResponse = {
  user_id: string
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

const validateToken = async (token: string): Promise<ValidateUserResponse> => {
  const response = await api.post<ValidateUserResponse>(
    '/v1/avito/users/validate',
    { token },
    { skipAuth: true },
  )
  return response.data
}

export const authApi = {
  createUser,
  validateToken,
}
