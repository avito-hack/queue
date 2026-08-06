import { useEffect, useRef } from 'react'
import { useAppDispatch, useAppSelector } from '../../app/hooks'
import { queueApi } from './api'
import { updateQueueItem } from './queueSlice'

const POLL_MS = 5000

function currentUserId() {
  return localStorage.getItem('authToken') ?? 'user-1'
}

export function useQueuePolling(enabled = true) {
  const dispatch = useAppDispatch()
  const queueItems = useAppSelector((state) => state.queue.queueItems)
  const hasQueued = queueItems.some((item) => item.status === 'queued')
  const queueItemsRef = useRef(queueItems)
  queueItemsRef.current = queueItems

  useEffect(() => {
    if (!enabled || !hasQueued) return

    let cancelled = false

    const sync = async () => {
      try {
        const data = await queueApi.getPosition(currentUserId())
        if (cancelled || typeof data?.position !== 'number') return

        for (const item of queueItemsRef.current) {
          if (item.status !== 'queued') continue
          dispatch(updateQueueItem({ ...item, position: data.position }))
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
