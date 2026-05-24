import { useState, useEffect } from 'react'

// ── Types ─────────────────────────────────────────────────────────────────────

type TCIASeriesItem = {
  series_uid: string
  modality: string
  body_part: string
  description: string
  slice_count: number
  collection: string
}

type ImportProgress = {
  series_uid: string
  description: string
  status: 'pending' | 'importing' | 'done' | 'error'
  error?: string
  studies_created?: number
  study_ids?: string[]
}

type Project = {
  id: string
  name: string
  slug: string
  archived?: boolean
}

// ── Constants ─────────────────────────────────────────────────────────────────

const COLLECTIONS = [
  { value: 'UPENN-GBM', label: 'UPENN-GBM — University of Pennsylvania Glioblastoma Brain MRI' },
  { value: 'GBM-DSC-MRI-DRO', label: 'GBM-DSC-MRI-DRO — GBM DSC-MRI Digital Reference Objects' },
]

// ── Component ─────────────────────────────────────────────────────────────────

export function TCIAPanel({ isAdmin }: { isAdmin: boolean }) {
  const [projects, setProjects] = useState<Project[]>([])
  const [collection, setCollection] = useState(COLLECTIONS[0].value)
  const [minSlices, setMinSlices] = useState(20)
  const [projectSlug, setProjectSlug] = useState('')

  const [series, setSeries] = useState<TCIASeriesItem[]>([])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [searching, setSearching] = useState(false)
  const [searchError, setSearchError] = useState<string | null>(null)

  const [progress, setProgress] = useState<ImportProgress[]>([])
  const [importing, setImporting] = useState(false)

  // Load project list once.
  useEffect(() => {
    fetch('/api/projects')
      .then(r => r.json())
      .then((data: Project[]) => {
        const active = (data ?? []).filter(p => !p.archived)
        setProjects(active)
        if (active.length > 0) setProjectSlug(active[0].slug)
      })
      .catch(() => { /* non-fatal */ })
  }, [])

  // ── Search ─────────────────────────────────────────────────────────────────

  async function searchSeries() {
    setSearching(true)
    setSearchError(null)
    setSeries([])
    setSelected(new Set())
    setProgress([])
    try {
      const url = `/api/tcia/series?collection=${encodeURIComponent(collection)}&min_slices=${minSlices}`
      const res = await fetch(url)
      if (!res.ok) {
        const err = await res.json()
        throw new Error(err.error ?? `HTTP ${res.status}`)
      }
      const data = await res.json()
      setSeries(data.series ?? [])
    } catch (err) {
      setSearchError(err instanceof Error ? err.message : 'Search failed')
    } finally {
      setSearching(false)
    }
  }

  // ── Selection ──────────────────────────────────────────────────────────────

  function toggleAll(checked: boolean) {
    setSelected(checked ? new Set(series.map(s => s.series_uid)) : new Set())
  }

  function toggleOne(uid: string) {
    setSelected(prev => {
      const next = new Set(prev)
      next.has(uid) ? next.delete(uid) : next.add(uid)
      return next
    })
  }

  // ── Import ─────────────────────────────────────────────────────────────────

  async function importSelected() {
    const toImport = series.filter(s => selected.has(s.series_uid))
    if (toImport.length === 0) return

    setImporting(true)
    setProgress(toImport.map(s => ({
      series_uid: s.series_uid,
      description: s.description || s.modality || s.series_uid.slice(-12),
      status: 'pending',
    })))

    for (const item of toImport) {
      setProgress(prev =>
        prev.map(p => p.series_uid === item.series_uid ? { ...p, status: 'importing' } : p)
      )
      try {
        const res = await fetch('/api/tcia/import', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            series_uid: item.series_uid,
            collection: item.collection,
            project_slug: projectSlug,
          }),
        })
        if (!res.ok) {
          const err = await res.json()
          throw new Error(err.error ?? `HTTP ${res.status}`)
        }
        const result = await res.json()
        setProgress(prev =>
          prev.map(p =>
            p.series_uid === item.series_uid
              ? { ...p, status: 'done', studies_created: result.studies_created, study_ids: result.study_ids }
              : p
          )
        )
      } catch (err) {
        setProgress(prev =>
          prev.map(p =>
            p.series_uid === item.series_uid
              ? { ...p, status: 'error', error: err instanceof Error ? err.message : 'Import failed' }
              : p
          )
        )
      }
    }

    setImporting(false)
  }

  // ── Render ─────────────────────────────────────────────────────────────────

  const allChecked = series.length > 0 && series.every(s => selected.has(s.series_uid))
  const someChecked = selected.size > 0

  return (
    <div className="tcia-panel">

      {/* Header */}
      <div className="tcia-header">
        <h2>TCIA Import</h2>
        <p className="tcia-subtitle">
          Download real brain MRI studies from the{' '}
          <a
            href="https://www.cancerimagingarchive.net"
            target="_blank"
            rel="noopener noreferrer"
          >
            Cancer Imaging Archive
          </a>
          {' '}directly into AEGIS. All listed collections are publicly available under Creative Commons licensing.
          Data is already de-identified at the DICOM tag level by TCIA.
        </p>
      </div>

      {/* Search bar */}
      <div className="tcia-search-bar">
        <div className="tcia-field">
          <label className="tcia-label">Collection</label>
          <select
            className=""
            value={collection}
            onChange={e => setCollection(e.target.value)}
            disabled={searching || importing}
          >
            {COLLECTIONS.map(c => (
              <option key={c.value} value={c.value}>{c.label}</option>
            ))}
          </select>
        </div>

        <div className="tcia-field tcia-field--narrow">
          <label className="tcia-label">Min slices</label>
          <input
            type="number"
            className="aegis-filter"
            value={minSlices}
            min={1}
            max={1000}
            onChange={e => setMinSlices(Math.max(1, Number(e.target.value)))}
            disabled={searching || importing}
          />
        </div>

        {projects.length > 0 && (
          <div className="tcia-field">
            <label className="tcia-label">Project</label>
            <select
              className=""
              value={projectSlug}
              onChange={e => setProjectSlug(e.target.value)}
              disabled={searching || importing}
            >
              {projects.map(p => (
                <option key={p.id} value={p.slug}>{p.name}</option>
              ))}
            </select>
          </div>
        )}

        <div className="tcia-field tcia-field--action">
          <button
            type="button"
            className="aegis-btn-primary"
            onClick={searchSeries}
            disabled={searching || importing}
          >
            {searching ? 'Searching…' : 'Search'}
          </button>
        </div>
      </div>

      {searchError && (
        <div className="form-error tcia-search-error">{searchError}</div>
      )}

      {/* Series results table */}
      {series.length > 0 && (
        <div className="tcia-results">
          <div className="tcia-results-header">
            <span className="tcia-count">
              {series.length} series found in {collection}
            </span>
            {isAdmin && (
              <button
                type="button"
                className="aegis-btn-primary"
                onClick={importSelected}
                disabled={!someChecked || importing}
              >
                {importing
                  ? 'Importing…'
                  : `Import Selected (${selected.size})`}
              </button>
            )}
          </div>

          <div className="tcia-table-wrap">
            <table className="table tcia-table">
              <thead>
                <tr>
                  {isAdmin && (
                    <th>
                      <input
                        type="checkbox"
                        checked={allChecked}
                        onChange={e => toggleAll(e.target.checked)}
                        disabled={importing}
                        title="Select all"
                      />
                    </th>
                  )}
                  <th>Modality</th>
                  <th>Body Part</th>
                  <th>Description</th>
                  <th>Slices</th>
                  <th>Series UID</th>
                </tr>
              </thead>
              <tbody>
                {series.map(s => (
                  <tr
                    key={s.series_uid}
                    className={selected.has(s.series_uid) ? 'tcia-row--selected' : ''}
                  >
                    {isAdmin && (
                      <td>
                        <input
                          type="checkbox"
                          checked={selected.has(s.series_uid)}
                          onChange={() => toggleOne(s.series_uid)}
                          disabled={importing}
                        />
                      </td>
                    )}
                    <td>
                      <span className="badge">{s.modality || '—'}</span>
                    </td>
                    <td>{s.body_part || '—'}</td>
                    <td className="tcia-desc">{s.description || '—'}</td>
                    <td>{s.slice_count}</td>
                    <td className="tcia-uid" title={s.series_uid}>
                      …{s.series_uid.slice(-20)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Import progress */}
      {progress.length > 0 && (
        <div className="tcia-progress">
          <h3>Import progress</h3>
          <table className="table tcia-table">
            <thead>
              <tr>
                <th>Description</th>
                <th>Status</th>
                <th>Result</th>
              </tr>
            </thead>
            <tbody>
              {progress.map(p => (
                <tr key={p.series_uid}>
                  <td className="tcia-desc">{p.description}</td>
                  <td>
                    {p.status === 'pending' && (
                      <span className="badge badge--pending">Pending</span>
                    )}
                    {p.status === 'importing' && (
                      <span className="badge badge--scanning">Importing…</span>
                    )}
                    {p.status === 'done' && (
                      <span className="badge badge--approved">Done</span>
                    )}
                    {p.status === 'error' && (
                      <span className="badge badge--rejected">Error</span>
                    )}
                  </td>
                  <td>
                    {p.status === 'done' && (
                      <span>
                        {p.studies_created ?? 0} {(p.studies_created ?? 0) === 1 ? 'study' : 'studies'} imported
                        {(p.studies_created ?? 0) === 0 && ' (duplicate — already in AEGIS)'}
                      </span>
                    )}
                    {p.status === 'error' && (
                      <span className="tcia-error-text">{p.error}</span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Attribution */}
      <p className="tcia-attribution">
        Data provided by the{' '}
        <a href="https://www.cancerimagingarchive.net" target="_blank" rel="noopener noreferrer">
          National Cancer Institute Cancer Imaging Archive (TCIA)
        </a>
        . Please cite the original collection when publishing results.
      </p>
    </div>
  )
}
