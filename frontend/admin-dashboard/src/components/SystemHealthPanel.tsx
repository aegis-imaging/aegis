import { useState, useEffect, useCallback } from 'react'

type ServiceEntry = { status: string; url?: string }

type HealthSummary = {
  generated_at: string
  api: { status: string; database: string; storage: string }
  services: Record<string, ServiceEntry>
  pipeline: {
    studies_received_24h: number
    studies_approved_24h: number
    studies_stuck: number
    pipeline_error_rate: number
  }
  dimse: { pending_retries: number; dead_letter: number }
}

const SERVICE_LABELS: Record<string, string> = {
  defacing:       'Defacing',
  phi_detection:  'PHI Detection',
  qc_service:     'QC Service',
  bids_service:   'BIDS Conversion',
  classification: 'Classification',
  protocol:       'Protocol Check',
  dimse_receiver: 'DIMSE Receiver',
}

function StatusDot({ status }: { status: string }) {
  const colour =
    status === 'healthy'  ? '#0d9488' :   // teal
    status === 'disabled' ? '#9ca3af' :   // gray
    '#ea580c'                             // orange (unhealthy / degraded)

  const label =
    status === 'healthy'  ? 'Healthy'  :
    status === 'disabled' ? 'Disabled' :
    status === 'ok'       ? 'OK'       :
    'Degraded'

  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
      <span style={{
        display: 'inline-block', width: 10, height: 10, borderRadius: '50%',
        background: colour, flexShrink: 0,
      }} />
      <span style={{ color: colour, fontWeight: 600, fontSize: 13 }}>{label}</span>
    </span>
  )
}

function SectionCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={{
      background: '#fff', border: '1px solid #e5e7eb', borderRadius: 8,
      padding: '16px 20px', marginBottom: 16,
    }}>
      <h3 style={{ margin: '0 0 12px', fontSize: 15, fontWeight: 700, color: '#111827' }}>{title}</h3>
      {children}
    </div>
  )
}

function MetricRow({ label, value, sub }: { label: string; value: string | number; sub?: string }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline', padding: '4px 0', borderBottom: '1px solid #f3f4f6' }}>
      <span style={{ fontSize: 13, color: '#374151' }}>{label}</span>
      <span style={{ fontWeight: 600, fontSize: 14, color: '#111827' }}>
        {value}{sub && <span style={{ fontWeight: 400, fontSize: 12, color: '#6b7280', marginLeft: 4 }}>{sub}</span>}
      </span>
    </div>
  )
}

export function SystemHealthPanel() {
  const [summary, setSummary] = useState<HealthSummary | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lastRefreshed, setLastRefreshed] = useState<Date | null>(null)

  const refresh = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch('/api/system/health-summary')
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      const data: HealthSummary = await res.json()
      setSummary(data)
      setLastRefreshed(new Date())
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load health summary')
    } finally {
      setLoading(false)
    }
  }, [])

  // Initial load + auto-refresh every 60 s.
  useEffect(() => {
    refresh()
    const id = setInterval(refresh, 60_000)
    return () => clearInterval(id)
  }, [refresh])

  const apiOverall = summary?.api.status === 'ok' ? 'ok' : 'degraded'

  return (
    <div style={{ maxWidth: 900, margin: '0 auto', padding: '16px 0' }}>

      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
        <h2 style={{ margin: 0, fontSize: 20, fontWeight: 700, color: '#111827' }}>System Health</h2>
        <button
          type="button"
          onClick={refresh}
          disabled={loading}
          style={{
            marginLeft: 'auto', padding: '6px 14px', fontSize: 13, fontWeight: 600,
            background: loading ? '#e5e7eb' : '#0d9488', color: loading ? '#6b7280' : '#fff',
            border: 'none', borderRadius: 6, cursor: loading ? 'not-allowed' : 'pointer',
          }}
        >
          {loading ? 'Refreshing…' : '↺ Refresh'}
        </button>
      </div>

      {lastRefreshed && (
        <p style={{ fontSize: 12, color: '#6b7280', marginBottom: 16, marginTop: -12 }}>
          Last refreshed: {lastRefreshed.toLocaleTimeString()} · Auto-refreshes every 60 s
        </p>
      )}

      {error && (
        <div style={{
          background: '#ffedd5', border: '1px solid #fed7aa', borderRadius: 6,
          padding: '10px 14px', color: '#9a3412', marginBottom: 16, fontSize: 13,
        }}>
          {error}
        </div>
      )}

      {summary && (<>

        {/* ── API Core ────────────────────────────────────────────────────── */}
        <SectionCard title="API Core">
          <div style={{ display: 'flex', gap: 32, flexWrap: 'wrap' }}>
            <div>
              <div style={{ fontSize: 12, color: '#6b7280', marginBottom: 4 }}>Overall</div>
              <StatusDot status={apiOverall} />
            </div>
            <div>
              <div style={{ fontSize: 12, color: '#6b7280', marginBottom: 4 }}>Database</div>
              <StatusDot status={summary.api.database} />
            </div>
            <div>
              <div style={{ fontSize: 12, color: '#6b7280', marginBottom: 4 }}>Storage</div>
              <StatusDot status={summary.api.storage} />
            </div>
          </div>
        </SectionCard>

        {/* ── Sidecar Services ─────────────────────────────────────────────── */}
        <SectionCard title="Processing Services">
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: 12 }}>
            {Object.entries(summary.services).map(([name, svc]) => (
              <div key={name} style={{
                background: '#f9fafb', border: '1px solid #e5e7eb',
                borderRadius: 6, padding: '10px 14px',
              }}>
                <div style={{ fontSize: 12, color: '#6b7280', marginBottom: 6, fontWeight: 500 }}>
                  {SERVICE_LABELS[name] || name}
                </div>
                <StatusDot status={svc.status} />
              </div>
            ))}
          </div>
        </SectionCard>

        {/* ── Pipeline Metrics ──────────────────────────────────────────────── */}
        <SectionCard title="Pipeline (last 24 h)">
          <MetricRow
            label="Studies received"
            value={summary.pipeline.studies_received_24h}
          />
          <MetricRow
            label="Studies approved"
            value={summary.pipeline.studies_approved_24h}
          />
          <MetricRow
            label="Stuck studies"
            value={summary.pipeline.studies_stuck}
            sub="(idle > 60 min)"
          />
          <MetricRow
            label="Rejection rate"
            value={`${(summary.pipeline.pipeline_error_rate * 100).toFixed(1)}%`}
          />
        </SectionCard>

        {/* ── DIMSE ────────────────────────────────────────────────────────── */}
        <SectionCard title="DIMSE Receiver">
          {summary.dimse.pending_retries === 0 && summary.dimse.dead_letter === 0 ? (
            <span style={{ fontSize: 13, color: '#0d9488', fontWeight: 600 }}>✓ No pending retries or dead-letter items</span>
          ) : (
            <>
              <MetricRow
                label="Pending retries"
                value={summary.dimse.pending_retries}
              />
              <MetricRow
                label="Dead-letter queue"
                value={summary.dimse.dead_letter}
              />
              {summary.dimse.dead_letter > 0 && (
                <p style={{ margin: '8px 0 0', fontSize: 12, color: '#9a3412' }}>
                  Dead-letter items need manual review in the DIMSE Operations tab.
                </p>
              )}
            </>
          )}
        </SectionCard>

        <p style={{ fontSize: 11, color: '#9ca3af', textAlign: 'right' }}>
          Generated at {summary.generated_at} · Cached 30 s server-side
        </p>
      </>)}
    </div>
  )
}
