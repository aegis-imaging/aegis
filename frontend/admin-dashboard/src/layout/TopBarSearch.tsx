import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiSearch, type SearchResponse } from '../api/subjects'

// TopBarSearch is the cross-entity search input that lives in the dashboard
// chrome. Modeled on XNAT's "Browse" dropdown (filterable, multi-entity)
// rather than its old quick-search server round-trip. Implementation is
// original to AEGIS.
export function TopBarSearch() {
  const [q, setQ] = useState('')
  const [results, setResults] = useState<SearchResponse | null>(null)
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const containerRef = useRef<HTMLDivElement | null>(null)
  const navigate = useNavigate()

  // Debounce search to keep typing responsive. 200ms hits the sweet spot
  // between feeling instant and not flooding the backend.
  useEffect(() => {
    const trimmed = q.trim()
    if (trimmed.length < 2) {
      setResults(null)
      return
    }
    const handle = setTimeout(async () => {
      setLoading(true)
      try {
        const r = await apiSearch(trimmed)
        setResults(r)
      } catch {
        setResults(null)
      } finally {
        setLoading(false)
      }
    }, 200)
    return () => clearTimeout(handle)
  }, [q])

  // Close dropdown when clicking outside.
  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', onClick)
    return () => document.removeEventListener('mousedown', onClick)
  }, [])

  const totalHits =
    (results?.projects.length ?? 0) +
    (results?.subjects.length ?? 0) +
    (results?.studies.length ?? 0)

  function goTo(path: string) {
    setOpen(false)
    setQ('')
    navigate(path)
  }

  return (
    <div className="xn-search" ref={containerRef}>
      <input
        type="search"
        className="xn-search-input"
        placeholder="Search projects, subjects, studies…"
        value={q}
        onChange={(e) => {
          setQ(e.target.value)
          setOpen(true)
        }}
        onFocus={() => q.trim().length >= 2 && setOpen(true)}
        aria-label="Search projects, subjects, and studies"
      />
      {open && q.trim().length >= 2 && (
        <div className="xn-search-dropdown" role="listbox">
          {loading && <div className="xn-search-status">Searching…</div>}
          {!loading && totalHits === 0 && (
            <div className="xn-search-status">No matches</div>
          )}
          {results && results.projects.length > 0 && (
            <Section title="Projects">
              {results.projects.map((p) => (
                <button
                  key={`p-${p.project_id}`}
                  className="xn-search-hit"
                  onClick={() => goTo(`/projects/${p.project_id}`)}
                >
                  <div className="xn-search-label">{p.label}</div>
                  {p.secondary && <div className="xn-search-secondary">{p.secondary}</div>}
                </button>
              ))}
            </Section>
          )}
          {results && results.subjects.length > 0 && (
            <Section title="Subjects">
              {results.subjects.map((s) => (
                <button
                  key={`s-${s.project_id}-${s.subject_id}`}
                  className="xn-search-hit"
                  onClick={() => goTo(`/projects/${s.project_id}/subjects/${s.subject_id}`)}
                >
                  <div className="xn-search-label">{s.label}</div>
                  {s.secondary && <div className="xn-search-secondary">{s.secondary}</div>}
                </button>
              ))}
            </Section>
          )}
          {results && results.studies.length > 0 && (
            <Section title="Studies">
              {results.studies.map((st) => (
                <button
                  key={`st-${st.study_id}`}
                  className="xn-search-hit"
                  onClick={() => {
                    const path = st.subject_id
                      ? `/projects/${st.project_id}/subjects/${st.subject_id}/studies/${st.study_id}`
                      : `/admin/studies` // fall back to admin list when no subject
                    goTo(path)
                  }}
                >
                  <div className="xn-search-label">{st.label}</div>
                  {st.secondary && <div className="xn-search-secondary">{st.secondary}</div>}
                </button>
              ))}
            </Section>
          )}
        </div>
      )}
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="xn-search-section">
      <div className="xn-search-section-title">{title}</div>
      <div className="xn-search-section-body">{children}</div>
    </div>
  )
}
