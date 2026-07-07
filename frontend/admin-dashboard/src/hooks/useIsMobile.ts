import { useEffect, useState } from 'react'

// Must match nav.css's mobile breakpoint (@media (max-width: 768px)) — the
// sidebar drawer, overlay, and hamburger toggle all switch at 768px, and
// ResearcherSidebar uses this hook to decide anchor-vs-NavLink rendering.
const MOBILE_QUERY = '(max-width: 768px)'

// useIsMobile reports whether the viewport is at or below the mobile
// breakpoint, updating live on resize/rotation via matchMedia.
export function useIsMobile(): boolean {
  const [isMobile, setIsMobile] = useState(
    () => typeof window !== 'undefined' && window.matchMedia(MOBILE_QUERY).matches,
  )

  useEffect(() => {
    const mql = window.matchMedia(MOBILE_QUERY)
    const onChange = (e: MediaQueryListEvent) => setIsMobile(e.matches)
    mql.addEventListener('change', onChange)
    return () => mql.removeEventListener('change', onChange)
  }, [])

  return isMobile
}
