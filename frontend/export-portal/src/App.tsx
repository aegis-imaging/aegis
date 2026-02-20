import { useState, useEffect } from 'react'
import './App.css'

type ExportFile = {
  key: string
  url: string
}

type ExportData = {
  share_id: string
  status?: 'active' | 'expired' | 'revoked'
  study_uid: string
  modality: string
  body_part: string
  study_description: string
  instance_count: number
  expires_at: string
  expires_in_seconds?: number
  note: string
  created_by: string
  download_url: string
  files: ExportFile[]
}

type ErrorState = {
  status: number
  message: string
}

const DATE_TIME_FORMAT = new Intl.DateTimeFormat(undefined, {
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
  timeZoneName: 'short',
})

function fmtDateTime(iso: string) {
  if (!iso) return ''
  const dt = new Date(iso)
  if (Number.isNaN(dt.getTime())) return iso
  return DATE_TIME_FORMAT.format(dt)
}

function fmtRemaining(seconds: number) {
  const total = Math.max(0, Math.floor(seconds))
  const days = Math.floor(total / 86400)
  const hours = Math.floor((total % 86400) / 3600)
  const mins = Math.floor((total % 3600) / 60)
  const secs = total % 60

  if (days > 0) return `${days}d ${hours}h remaining`
  if (hours > 0) return `${hours}h ${mins}m remaining`
  if (mins > 0) return `${mins}m ${secs}s remaining`
  return `${secs}s remaining`
}

export function App() {
  const [data, setData] = useState<ExportData | null>(null)
  const [expiryEpochMs, setExpiryEpochMs] = useState<number | null>(null)
  const [nowMs, setNowMs] = useState(() => Date.now())
  const [error, setError] = useState<ErrorState | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const token = params.get('token')
    if (!token) {
      setError({ status: 400, message: 'No share token provided. Check the link you received.' })
      setLoading(false)
      return
    }

    fetch(`/api/export/${token}`)
      .then(async (resp) => {
        if (!resp.ok) {
          const body = await resp.json().catch(() => ({ error: 'Unknown error' }))
          setError({ status: resp.status, message: body.error || `HTTP ${resp.status}` })
          return
        }
        const json: ExportData = await resp.json()
        // Build download URL relative to current origin
        json.download_url = `/api/export/${token}/download`
        let nextExpiryEpochMs: number | null = null
        const parsedExpiresAt = Date.parse(json.expires_at)
        if (Number.isFinite(parsedExpiresAt)) {
          nextExpiryEpochMs = parsedExpiresAt
        }
        if (typeof json.expires_in_seconds === 'number') {
          const relativeExpiryMs = Date.now() + (Math.max(0, Math.floor(json.expires_in_seconds)) * 1000)
          if (nextExpiryEpochMs === null) {
            nextExpiryEpochMs = relativeExpiryMs
          }
        }
        setExpiryEpochMs(nextExpiryEpochMs)
        setNowMs(Date.now())
        setData(json)
      })
      .catch(() => {
        setError({ status: 0, message: 'Failed to connect to the server. Please try again later.' })
      })
      .finally(() => setLoading(false))
  }, [])

  const hasLiveCountdown = expiryEpochMs !== null && nowMs < expiryEpochMs

  useEffect(() => {
    if (!data || !hasLiveCountdown) return
    const id = window.setInterval(() => {
      setNowMs(Date.now())
    }, 1000)
    return () => window.clearInterval(id)
  }, [data?.share_id, hasLiveCountdown])

  if (loading) {
    return (
      <div className="container">
        <div className="card">
          <h1>AEGIS</h1>
          <p className="loading">Loading share...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="container">
        <div className="card">
          <h1>AEGIS</h1>
          <div className="error-box">
            {error.status === 410 ? (
              <>
                <h2>Link Expired or Revoked</h2>
                <p>This share link is no longer valid. Please contact the person who shared it with you to request a new link.</p>
              </>
            ) : error.status === 404 ? (
              <>
                <h2>Share Not Found</h2>
                <p>This share link was not found. It may have been deleted or the URL may be incorrect.</p>
              </>
            ) : (
              <>
                <h2>Error</h2>
                <p>{error.message}</p>
              </>
            )}
          </div>
        </div>
      </div>
    )
  }

  if (!data) return null

  const expiresDate = new Date(data.expires_at)
  const parsedExpiresMs = Number.isFinite(expiresDate.getTime()) ? expiresDate.getTime() : null
  const effectiveExpiryMs = expiryEpochMs ?? parsedExpiresMs
  const secondsUntilExpiry =
    effectiveExpiryMs === null
      ? null
      : Math.max(0, Math.floor((effectiveExpiryMs - nowMs) / 1000))
  const isExpiredByStatus = data.status === 'expired' || data.status === 'revoked'
  const isExpiredByTime = secondsUntilExpiry !== null && secondsUntilExpiry <= 0
  const isExpired = isExpiredByStatus || isExpiredByTime
  const isExpiringSoon =
    !isExpired && secondsUntilExpiry !== null && secondsUntilExpiry < 24 * 60 * 60

  return (
    <div className="container">
      <div className="card">
        <h1>AEGIS</h1>
        <p className="subtitle">Secure Study Download</p>

        <div className="study-info">
          <h2>Study Details</h2>
          <table className="info-table">
            <tbody>
              {data.modality && (
                <tr><td className="label">Modality</td><td>{data.modality}</td></tr>
              )}
              {data.body_part && (
                <tr><td className="label">Body Part</td><td>{data.body_part}</td></tr>
              )}
              {data.study_description && (
                <tr><td className="label">Description</td><td>{data.study_description}</td></tr>
              )}
              <tr><td className="label">Files</td><td>{data.files.length} DICOM file{data.files.length !== 1 ? 's' : ''}</td></tr>
              <tr>
                <td className="label">Expires</td>
                <td className={isExpiringSoon ? 'expiring-soon' : ''}>
                  {fmtDateTime(data.expires_at)}
                  {isExpired && ' (expired — refresh link)'}
                  {!isExpired && secondsUntilExpiry !== null && ` (${fmtRemaining(secondsUntilExpiry)})`}
                  {isExpiringSoon && ' (expiring soon)'}
                </td>
              </tr>
              {data.created_by && (
                <tr><td className="label">Shared by</td><td>{data.created_by}</td></tr>
              )}
            </tbody>
          </table>
          {data.note && (
            <div className="note">
              <strong>Note:</strong> {data.note}
            </div>
          )}
        </div>

        <div className="download-section">
          {!isExpired ? (
            <a href={data.download_url} className="btn-download" download>
              Download All ({data.files.length} file{data.files.length !== 1 ? 's' : ''}) as ZIP
            </a>
          ) : (
            <p className="download-disabled">This share has expired. Please request a new link.</p>
          )}
        </div>

        <details className="file-list">
          <summary>Individual files ({data.files.length})</summary>
          <ul>
            {data.files.map((f, i) => {
              const parts = f.key.split('/')
              const filename = parts[parts.length - 1]
              return (
                <li key={i}>
                  {!isExpired ? (
                    <a href={f.url} download={filename}>{filename}</a>
                  ) : (
                    <span className="file-link-disabled">{filename}</span>
                  )}
                </li>
              )
            })}
          </ul>
        </details>

        <p className="footer-text">
          Files are de-identified and shared securely via AEGIS.
          Download links expire at the time shown above.
        </p>
      </div>
    </div>
  )
}
