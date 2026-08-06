import { useEffect, useRef } from 'react'
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
  const queueItemsRef = useRef(queueItems)
  queueItemsRef.current = queueItems

  useEffect(() => {
    if (!enabled || !hasQueued) return

    let cancelled = false

    const sync = async () => {
      try {
        const infos = await queueApi.listUserQueues()
        if (cancelled) return

        const updates = queueItemsFromUserQueues(queueItemsRef.current, infos)
        for (const item of updates) {
          dispatch(updateQueueItem(item))
        }
      } catch {
        // бэк ещё не готов
      }
    }

    void sync()
    const id = window.setInterval(() => {
      void sync()
    }, POLL_MS)

    return () => {
      cancelled = true
      window.clearInterval(id)
    }
  }, [dispatch, enabled, hasQueued])
}
