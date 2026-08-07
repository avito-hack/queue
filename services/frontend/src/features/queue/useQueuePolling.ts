import { useEffect, useEffectEvent } from 'react'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { queueApi } from './api'
import { queueItemsFromUserQueues } from './positionLib'
import { updateQueueItem } from './queueSlice'

const POLL_MS = 5000

/**
 * Синхронизация позиций/soldout по GET /v1/user/queues
 * (массив очередей пользователя с item_id + position).
 */
export function useQueuePolling(enabled = true) {
  const dispatch = useAppDispatch()
  const queueItems = useAppSelector((state) => state.queue.queueItems)
  const hasQueued = queueItems.some(
    (item) => item.status === 'queued' || item.status === 'soldout',
  )

  const sync = useEffectEvent(async () => {
    try {
      const infos = await queueApi.listUserQueues()
      const updates = queueItemsFromUserQueues(queueItems, infos)
      for (const item of updates) {
        dispatch(updateQueueItem(item))
      }
    } catch {
      // бэк ещё не готов
    }
  })

  useEffect(() => {
    if (!enabled || !hasQueued) return

    void sync()
    const id = window.setInterval(() => {
      void sync()
    }, POLL_MS)

    return () => {
      window.clearInterval(id)
    }
  }, [enabled, hasQueued])
}
