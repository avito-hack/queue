import { authApi } from '../auth/api'
import {
  DEFAULT_DEMO_TOKEN,
  DEFAULT_DEMO_USER_ID,
  switchActiveUser,
} from '../../shared/auth/demoAuth'
import { removeDemoUserByToken } from './demoUsers'

export async function switchJuryUser(user: {
  id: string
  token: string
}): Promise<'ok' | 'removed'> {
  if (user.token === DEFAULT_DEMO_TOKEN) {
    switchActiveUser({ id: DEFAULT_DEMO_USER_ID, token: DEFAULT_DEMO_TOKEN })
    return 'ok'
  }

  try {
    const validated = await authApi.validateToken(user.token)
    switchActiveUser({
      id: validated.user_id || user.id,
      token: user.token,
    })
    return 'ok'
  } catch {
    removeDemoUserByToken(user.token)
    return 'removed'
  }
}
