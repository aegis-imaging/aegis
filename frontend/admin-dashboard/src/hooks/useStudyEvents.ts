import { useEffect, useRef, useCallback } from 'react'

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
 * Reconnects automatically with exponential back-off on drop.
 */
export function useStudyEvents({ projectId, onEvent }: Options) {
  const onEventRef = useRef(onEvent)
  onEventRef.current = onEvent

  const reconnectDelay = useRef(1000)

  const connect = useCallback(() => {
    const url = projectId
      ? `/api/studies/events?project_id=${encodeURIComponent(projectId)}`
      : '/api/studies/events'

    const es = new EventSource(url)

    es.onmessage = (e) => {
      try {
        const ev: StudyEvent = JSON.parse(e.data)
        onEventRef.current(ev)
      } catch {
        // ignore malformed events
      }
    }

    es.onopen = () => {
      reconnectDelay.current = 1000 // reset on successful connect
    }

    // EventSource auto-reconnects on network errors; this handles other cases.
    es.onerror = () => {
      es.close()
    }

    return es
  }, [projectId])

  useEffect(() => {
    let es = connect()
    let timeoutId: ReturnType<typeof setTimeout>

    function reconnect() {
      es.close()
      timeoutId = setTimeout(() => {
        es = connect()
        // Exponential back-off (capped at 30 s)
        reconnectDelay.current = Math.min(reconnectDelay.current * 2, 30_000)
      }, reconnectDelay.current)
    }

    // Re-use the same onerror to schedule reconnect.
    es.onerror = () => {
      es.close()
      reconnect()
    }

    return () => {
      clearTimeout(timeoutId)
      es.close()
    }
  }, [connect])
}
