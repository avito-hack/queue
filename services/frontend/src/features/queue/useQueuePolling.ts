import { useEffect, useEffectEvent } from 'react'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { queueApi } from './api'
import { queueItemsFromUserQueues } from './positionLib'
import { removeQueuedByProductId, updateQueueItem } from './queueSlice'

const POLL_MS = 5000

/**
 * Синхронизация позиций/soldout по GET /v1/user/queues
 * (массив ItemQueueInfo: item_id + position + status).
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
      const { updates, removeProductIds } = queueItemsFromUserQueues(
        queueItems,
        infos,
      )
      for (const productId of removeProductIds) {
        dispatch(removeQueuedByProductId(productId))
      }
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
