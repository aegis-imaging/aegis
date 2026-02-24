import { useState } from 'react'

type ComplianceReport = {
  project_id: string
  generated_at: string
  period_days: number
  studies: { total: number; approved: number; rejected: number; pending: number }
  phi_detection: { scanned: number; flagged: number; flag_rate_pct: number }
  defacing: { required: number; completed: number; failed: number; avg_qa_score: number }
  protocol_compliance: { checked: number; compliant: number; minor_deviations: number; non_compliant: number }
  exports: { shares_created: number; shares_downloaded: number; total_downloads: number }
}

function pct(n: number, d: number): string {
  if (d === 0) return '—'
  return `${((n / d) * 100).toFixed(1)}%`
}

function ReportRow({ label, value, sub }: { label: string; value: string | number; sub?: string }) {
  return (
    <tr>
      <td style={{ padding: '6px 10px', fontSize: 13, color: '#374151', borderBottom: '1px solid #f3f4f6' }}>{label}</td>
      <td style={{ padding: '6px 10px', fontSize: 13, fontWeight: 600, color: '#111827', textAlign: 'right', borderBottom: '1px solid #f3f4f6' }}>
        {value}{sub && <span style={{ fontWeight: 400, fontSize: 12, color: '#6b7280', marginLeft: 4 }}>{sub}</span>}
      </td>
    </tr>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div style={{ marginBottom: 20 }}>
      <h4 style={{ margin: '0 0 8px', fontSize: 13, fontWeight: 700, color: '#6b7280', textTransform: 'uppercase', letterSpacing: '0.04em' }}>{title}</h4>
      <table style={{ width: '100%', borderCollapse: 'collapse', background: '#fff', border: '1px solid #e5e7eb', borderRadius: 6, overflow: 'hidden' }}>
        <tbody>{children}</tbody>
      </table>
    </div>
  )
}

type Props = { projectId: string; onClose: () => void }

export function ComplianceReportPanel({ projectId, onClose }: Props) {
  const [days, setDays] = useState(30)
  const [report, setReport] = useState<ComplianceReport | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function load(d: number) {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`/api/projects/${projectId}/compliance-report?days=${d}`)
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      setReport(await res.json())
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load report')
    } finally {
      setLoading(false)
    }
  }

  function handleDownload() {
    if (!report) return
    const blob = new Blob([JSON.stringify(report, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `compliance-report-${projectId}-${days}d.json`
    a.click()
    URL.revokeObjectURL(url)
  }

  const r = report

  return (
    <div style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.4)',
      display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000,
    }} onClick={onClose}>
      <div style={{
        background: '#fff', borderRadius: 10, padding: '24px 28px',
        width: '90%', maxWidth: 560, maxHeight: '90vh', overflowY: 'auto',
        boxShadow: '0 20px 60px rgba(0,0,0,0.2)',
      }} onClick={e => e.stopPropagation()}>

        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
          <h3 style={{ margin: 0, fontSize: 17, fontWeight: 700, color: '#111827' }}>Compliance Report</h3>
          <button type="button" onClick={onClose} style={{ marginLeft: 'auto', background: 'none', border: 'none', fontSize: 20, cursor: 'pointer', color: '#6b7280' }}>×</button>
        </div>

        {/* Period selector */}
        <div style={{ display: 'flex', gap: 8, marginBottom: 16, alignItems: 'center' }}>
          <span style={{ fontSize: 13, color: '#374151' }}>Period:</span>
          {[7, 30, 90, 365].map(d => (
            <button
              key={d}
              type="button"
              onClick={() => { setDays(d); load(d) }}
              style={{
                padding: '4px 12px', fontSize: 12, fontWeight: 600,
                background: days === d ? '#0d9488' : '#f3f4f6',
                color: days === d ? '#fff' : '#374151',
                border: 'none', borderRadius: 4, cursor: 'pointer',
              }}
            >{d}d</button>
          ))}
          {!r && !loading && (
            <button
              type="button"
              onClick={() => load(days)}
              style={{
                marginLeft: 'auto', padding: '6px 16px', fontSize: 13, fontWeight: 600,
                background: '#0d9488', color: '#fff', border: 'none', borderRadius: 6, cursor: 'pointer',
              }}
            >Generate</button>
          )}
          {r && (
            <button
              type="button"
              onClick={handleDownload}
              style={{
                marginLeft: 'auto', padding: '6px 14px', fontSize: 12, fontWeight: 600,
                background: '#f3f4f6', color: '#374151', border: '1px solid #e5e7eb', borderRadius: 6, cursor: 'pointer',
              }}
            >↓ JSON</button>
          )}
        </div>

        {loading && <p style={{ color: '#6b7280', fontSize: 13, textAlign: 'center' }}>Loading…</p>}

        {error && (
          <div style={{ background: '#ffedd5', border: '1px solid #fed7aa', borderRadius: 6, padding: '10px 14px', color: '#9a3412', fontSize: 13 }}>
            {error}
          </div>
        )}

        {r && (<>
          <p style={{ fontSize: 12, color: '#6b7280', marginTop: 0, marginBottom: 16 }}>
            Generated {r.generated_at} · Last {r.period_days} days
          </p>

          <Section title="Studies">
            <ReportRow label="Total received" value={r.studies.total} />
            <ReportRow label="Approved" value={r.studies.approved} sub={pct(r.studies.approved, r.studies.total)} />
            <ReportRow label="Rejected" value={r.studies.rejected} sub={pct(r.studies.rejected, r.studies.total)} />
            <ReportRow label="Pending review" value={r.studies.pending} />
          </Section>

          <Section title="PHI Detection">
            <ReportRow label="Scanned" value={r.phi_detection.scanned} />
            <ReportRow label="Flagged" value={r.phi_detection.flagged} sub={`${r.phi_detection.flag_rate_pct.toFixed(2)}% flag rate`} />
          </Section>

          <Section title="Defacing">
            <ReportRow label="Required" value={r.defacing.required} />
            <ReportRow label="Completed" value={r.defacing.completed} sub={pct(r.defacing.completed, r.defacing.required)} />
            <ReportRow label="Failed / pending" value={r.defacing.failed} />
            <ReportRow label="Avg QA score" value={r.defacing.avg_qa_score > 0 ? r.defacing.avg_qa_score.toFixed(3) : '—'} />
          </Section>

          <Section title="Protocol Compliance">
            <ReportRow label="Checked" value={r.protocol_compliance.checked} />
            <ReportRow label="Compliant" value={r.protocol_compliance.compliant} sub={pct(r.protocol_compliance.compliant, r.protocol_compliance.checked)} />
            <ReportRow label="Minor deviations" value={r.protocol_compliance.minor_deviations} />
            <ReportRow label="Non-compliant" value={r.protocol_compliance.non_compliant} />
          </Section>

          <Section title="Exports">
            <ReportRow label="Shares created" value={r.exports.shares_created} />
            <ReportRow label="Shares downloaded" value={r.exports.shares_downloaded} sub={pct(r.exports.shares_downloaded, r.exports.shares_created)} />
            <ReportRow label="Total downloads" value={r.exports.total_downloads} />
          </Section>
        </>)}
      </div>
    </div>
  )
}
