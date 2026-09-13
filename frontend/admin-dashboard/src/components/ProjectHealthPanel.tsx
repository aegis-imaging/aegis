import { useState } from 'react'

type FunnelStage = {
  stage: string
  count: number
  pct_of_total: number
  pct_of_prev: number
}

type ProjectHealthData = {
  project_id: string
  period_days: number
  generated_at: string
  studies: {
    received: number
    approved: number
    rejected: number
    defacing: number
    clean: number
    defaced: number
  }
  storage: {
    raw_file_count: number
    clean_file_count: number
    total_file_count: number
    total_studies: number
  } | null
  stuck_count: number
  routing: {
    attempts: number
    successful: number
    failed: number
    success_rate: number
  }
  funnel: FunnelStage[]
}

const STAGE_LABELS: Record<string, string> = {
  received: 'Received',
  classified: 'Classified',
  phi_scanned: 'PHI Scanned',
  defaced: 'Defaced',
  qc_passed: 'QC Passed',
  bids_converted: 'BIDS',
  approved: 'Approved',
  exported: 'Exported',
}

function stageColor(pct: number): string {
  if (pct >= 80) return '#0d9488'
  if (pct >= 50) return '#b45309'
  return '#ea580c'
}

function pctStr(n: number, d: number): string {
  if (d === 0) return '—'
  return `${((n / d) * 100).toFixed(1)}%`
}

function MetricCard({ label, value, sub, warn }: { label: string; value: string | number; sub?: string; warn?: boolean }) {
  return (
    <div style={{
      background: '#fff', border: `1px solid ${warn ? '#fed7aa' : '#e5e7eb'}`,
      borderRadius: 8, padding: '12px 16px', flex: '1 1 140px', minWidth: 120,
    }}>
      <div style={{ fontSize: 11, fontWeight: 600, color: '#6b7280', textTransform: 'uppercase', letterSpacing: '0.04em', marginBottom: 4 }}>
        {label}
      </div>
      <div style={{ fontSize: 22, fontWeight: 700, color: warn ? '#9a3412' : '#111827' }}>{value}</div>
      {sub && <div style={{ fontSize: 11, color: '#6b7280', marginTop: 2 }}>{sub}</div>}
    </div>
  )
}

type Props = { projectId: string; projectName: string; onClose: () => void }

export function ProjectHealthPanel({ projectId, projectName, onClose }: Props) {
  const [days, setDays] = useState(30)
  const [data, setData] = useState<ProjectHealthData | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function load(d: number) {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`/api/stats/project-health?project_id=${encodeURIComponent(projectId)}&days=${d}`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      setData(await res.json())
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load health data')
    } finally {
      setLoading(false)
    }
  }

  const received = data?.funnel.find(f => f.stage === 'received')?.count ?? 0

  return (
    <div style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)',
      display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000,
    }} onClick={onClose}>
      <div style={{
        background: '#f9fafb', borderRadius: 10, padding: '24px 28px',
        width: '92%', maxWidth: 680, maxHeight: '92vh', overflowY: 'auto',
        boxShadow: '0 20px 60px rgba(0,0,0,0.2)',
      }} onClick={e => e.stopPropagation()}>

        {/* Header */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 18 }}>
          <div>
            <h3 style={{ margin: 0, fontSize: 17, fontWeight: 700, color: '#111827' }}>Project Health</h3>
            <div style={{ fontSize: 13, color: '#6b7280', marginTop: 2 }}>{projectName}</div>
          </div>
          <button type="button" onClick={onClose} aria-label="Close panel"
            style={{ marginLeft: 'auto', background: 'none', border: 'none', fontSize: 22, cursor: 'pointer', color: '#6b7280', lineHeight: 1 }}>×</button>
        </div>

        {/* Period selector + load */}
        <div style={{ display: 'flex', gap: 8, marginBottom: 18, alignItems: 'center' }}>
          <span style={{ fontSize: 13, color: '#374151' }}>Period:</span>
          {[7, 30, 90, 365].map(d => (
            <button key={d} type="button"
              onClick={() => { setDays(d); load(d) }}
              style={{
                padding: '4px 12px', fontSize: 12, fontWeight: 600,
                background: days === d && data ? '#0d9488' : '#f3f4f6',
                color: days === d && data ? '#fff' : '#374151',
                border: 'none', borderRadius: 4, cursor: 'pointer',
              }}>{d}d</button>
          ))}
          {!data && !loading && (
            <button type="button" onClick={() => load(days)}
              style={{
                marginLeft: 'auto', padding: '7px 18px', fontSize: 13, fontWeight: 600,
                background: '#0d9488', color: '#fff', border: 'none', borderRadius: 6, cursor: 'pointer',
              }}>Load</button>
          )}
          {data && (
            <span style={{ marginLeft: 'auto', fontSize: 11, color: '#9ca3af' }}>
              {new Date(data.generated_at).toLocaleString()}
            </span>
          )}
        </div>

        {loading && (
          <div style={{ textAlign: 'center', color: '#6b7280', fontSize: 13, padding: '32px 0' }}>Loading health data…</div>
        )}

        {error && (
          <div style={{ background: '#ffedd5', border: '1px solid #fed7aa', borderRadius: 6, padding: '10px 14px', color: '#9a3412', fontSize: 13 }}>
            {error}
          </div>
        )}

        {data && (<>
          {/* Key metrics */}
          <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', marginBottom: 20 }}>
            <MetricCard label="Received" value={received} sub={`last ${data.period_days} days`} />
            <MetricCard
              label="Approved"
              value={pctStr(data.funnel.find(f => f.stage === 'approved')?.count ?? 0, received)}
              sub={`${data.funnel.find(f => f.stage === 'approved')?.count ?? 0} studies`}
            />
            <MetricCard
              label="Stuck"
              value={data.stuck_count}
              sub="currently idle"
              warn={data.stuck_count > 0}
            />
            <MetricCard
              label="Routing"
              value={data.routing.attempts > 0 ? `${(data.routing.success_rate * 100).toFixed(0)}%` : '—'}
              sub={`${data.routing.attempts} attempts`}
              warn={data.routing.attempts > 0 && data.routing.success_rate < 0.9}
            />
            {data.storage && (
              <MetricCard
                label="Files"
                value={data.storage.total_file_count.toLocaleString()}
                sub={`${data.storage.raw_file_count} raw · ${data.storage.clean_file_count} clean`}
              />
            )}
          </div>

          {/* Pipeline funnel */}
          {data.funnel.length > 0 && (
            <div>
              <h4 style={{ margin: '0 0 10px', fontSize: 13, fontWeight: 700, color: '#6b7280', textTransform: 'uppercase', letterSpacing: '0.04em' }}>
                Pipeline Funnel — Last {data.period_days} days
              </h4>
              <div style={{ background: '#fff', border: '1px solid #e5e7eb', borderRadius: 8, overflow: 'hidden' }}>
                {data.funnel.map((stage, i) => {
                  const label = STAGE_LABELS[stage.stage] ?? stage.stage
                  const barPct = received > 0 ? (stage.count / received) * 100 : 0
                  const color = stageColor(stage.pct_of_total)
                  return (
                    <div key={stage.stage} style={{
                      display: 'flex', alignItems: 'center', gap: 10,
                      padding: '9px 14px',
                      borderBottom: i < data.funnel.length - 1 ? '1px solid #f3f4f6' : 'none',
                    }}>
                      {/* Stage name */}
                      <div style={{ width: 100, fontSize: 13, color: '#374151', fontWeight: 500, flexShrink: 0 }}>{label}</div>
                      {/* Bar */}
                      <div style={{ flex: 1, height: 8, background: '#f3f4f6', borderRadius: 4, overflow: 'hidden', minWidth: 60 }}>
                        <div style={{ width: `${Math.max(barPct, barPct > 0 ? 1 : 0)}%`, height: '100%', background: color, borderRadius: 4, transition: 'width 0.4s ease' }} />
                      </div>
                      {/* Count */}
                      <div style={{ width: 40, fontSize: 13, fontWeight: 700, color: '#111827', textAlign: 'right', flexShrink: 0 }}>
                        {stage.count.toLocaleString()}
                      </div>
                      {/* % of received */}
                      <div style={{ width: 52, fontSize: 12, color, textAlign: 'right', flexShrink: 0, fontWeight: 600 }}>
                        {received > 0 ? `${stage.pct_of_total.toFixed(1)}%` : '—'}
                      </div>
                      {/* conversion from prev */}
                      {i > 0 && (
                        <div style={{ width: 52, fontSize: 11, color: '#9ca3af', textAlign: 'right', flexShrink: 0 }}>
                          {data.funnel[i - 1].count > 0 ? `↓${stage.pct_of_prev.toFixed(0)}%` : ''}
                        </div>
                      )}
                      {i === 0 && <div style={{ width: 52 }} />}
                    </div>
                  )
                })}
              </div>
              <div style={{ fontSize: 11, color: '#9ca3af', marginTop: 6, textAlign: 'right' }}>
                % of received · conversion rate from prior stage
              </div>
            </div>
          )}
        </>)}
      </div>
    </div>
  )
}
