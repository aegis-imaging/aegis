import { useEffect, useRef, useState } from 'react'

export type StudyEvent = {
  type: 'study.status_changed' | 'study.created' | 'study.deleted'
  study_id: string
  project_id: string
  status?: string
  updated_at?: string
}

type Options = {
  projectId?: string
  onEvent: (ev: StudyEvent) => void
}

/**
 * useStudyEvents subscribes to the SSE stream at GET /api/studies/events.
 * Calls onEvent whenever the server pushes a study change.
 * Reconnects automatically with exponential back-off on drop (1 s initial,
 * doubling per consecutive failure, capped at 30 s; reset on open).
 *
 * Returns { connected } — starts optimistically true so callers never show
 * a "disconnected" state during initial page load; flips false only after
 * an actual connection error, and back to true when a socket opens.
 */
export function useStudyEvents({ projectId, onEvent }: Options): { connected: boolean } {
  const onEventRef = useRef(onEvent)
  onEventRef.current = onEvent

  const [connected, setConnected] = useState(true)

  useEffect(() => {
    let disposed = false
    let es: EventSource | null = null
    let timeoutId: ReturnType<typeof setTimeout> | undefined
    let reconnectDelay = 1000

    function connect() {
      if (disposed) return
      const url = projectId
        ? `/api/studies/events?project_id=${encodeURIComponent(projectId)}`
        : '/api/studies/events'

      const source = new EventSource(url)
      es = source

      source.onmessage = (e) => {
        try {
          const ev: StudyEvent = JSON.parse(e.data)
          onEventRef.current(ev)
        } catch {
          // ignore malformed events
        }
      }

      source.onopen = () => {
        reconnectDelay = 1000 // reset back-off on successful connect
        setConnected(true)
      }

      // Attached inside connect() so EVERY socket — initial AND each
      // reconnect — reschedules on error. (Previously only the first
      // socket had the rescheduling handler; reconnected sockets got a
      // close-only handler, so the stream died after one retry.)
      source.onerror = () => {
        source.close()
        setConnected(false)
        if (disposed) return
        clearTimeout(timeoutId)
        timeoutId = setTimeout(() => {
          // Exponential back-off (capped at 30 s)
          reconnectDelay = Math.min(reconnectDelay * 2, 30_000)
          connect()
        }, reconnectDelay)
      }
    }

    connect()

    return () => {
      disposed = true
      clearTimeout(timeoutId)
      es?.close()
    }
  }, [projectId])

  return { connected }
}
