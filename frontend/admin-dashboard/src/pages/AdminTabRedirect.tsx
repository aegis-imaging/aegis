import { Navigate, useParams, useSearchParams } from 'react-router-dom'

// AdminTabRedirect handles back-compat for legacy `/admin/<tab>` URLs that
// audit records, external links, email links, and historical bookmarks still
// reference. The whole admin URL tree moved to root-level (/studies,
// /audit, /routing, ...) when the /admin/ prefix was dropped; this component
// just rewrites the URL while preserving any path-tail (e.g. an institution
// detail id) and the full query string (e.g. ?study_id=).
//
// Routed under three patterns in main.tsx:
//   /admin/:tab            -> /:tab
//   /admin/:tab/:detail    -> /:tab/:detail
//   /admin                 -> /studies (handled via Navigate directly)
export function AdminTabRedirect() {
  const { tab, detail } = useParams<{ tab?: string; detail?: string }>()
  const [sp] = useSearchParams()
  if (!tab) return <Navigate to="/studies" replace />
  const path = detail ? `/${tab}/${detail}` : `/${tab}`
  const qs = sp.toString()
  return <Navigate to={qs ? `${path}?${qs}` : path} replace />
}
