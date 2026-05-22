import type { StudySummary as StudySummaryType } from '@aegis/client'

export type DisplayTimezoneMode = 'utc' | 'local' | 'custom'

export interface StudySummaryProps {
  summary: StudySummaryType
  displayTimezoneMode: DisplayTimezoneMode
  displayTimezoneCustom: string
}

export function StudySummary({ summary, displayTimezoneMode, displayTimezoneCustom }: StudySummaryProps) {
  const studyDateTime = formatStudyDateTime(summary, displayTimezoneMode, displayTimezoneCustom)
  const dicomOffset = normalizeOffset(summary.timezoneOffsetFromUtc)

  return (
    <div style={{
      border: '1px solid #e5e7eb',
      borderRadius: '8px',
      padding: '20px',
      backgroundColor: '#fff',
    }}>
      <h3 style={{ margin: '0 0 16px', fontSize: '16px', fontWeight: 600 }}>
        Study Summary
      </h3>
      <div style={{
        display: 'grid',
        gridTemplateColumns: '1fr 1fr',
        gap: '12px',
        fontSize: '14px',
      }}>
        <Field label="Patient Name" value={summary.patientName} sensitive />
        <Field label="Patient ID" value={summary.patientId} sensitive />
        <Field label="Study Date/Time" value={studyDateTime} />
        <Field label="DICOM TZ Offset" value={dicomOffset || '(not provided)'} />
        <Field label="Modality" value={summary.modality} />
        <Field label="Body Part" value={summary.bodyPart} />
        <Field label="Description" value={summary.studyDescription} />
        <Field label="Series" value={String(summary.seriesCount)} />
        <Field label="Images" value={String(summary.imageCount)} />
      </div>
    </div>
  )
}

function formatStudyDateTime(
  summary: StudySummaryType,
  mode: DisplayTimezoneMode,
  customTimeZone: string,
): string {
  if (summary.studyDateTimeIso) {
    const dt = new Date(summary.studyDateTimeIso)
    if (!Number.isNaN(dt.getTime())) {
      if (mode === 'local') return fmtDateWithIntl(dt)
      if (mode === 'custom') {
        const tz = normalizeIanaTimeZone(customTimeZone)
        return tz ? fmtDateWithIntl(dt, tz) : fmtDateUtc(dt)
      }
      return fmtDateUtc(dt)
    }
  }

  const rawDate = formatDicomDate(summary.studyDate)
  const rawTime = formatDicomTime(summary.studyTime)
  const rawOffset = normalizeOffset(summary.timezoneOffsetFromUtc)
  const raw = [rawDate, rawTime].filter(Boolean).join(' ').trim()
  if (!raw) return '(empty)'
  if (rawOffset) return `${raw} (offset ${rawOffset})`
  return `${raw} (floating DICOM time; no timezone offset)`
}

function formatDicomDate(value: string): string {
  if (!/^\d{8}$/.test(value)) return value || ''
  return `${value.slice(0, 4)}-${value.slice(4, 6)}-${value.slice(6, 8)}`
}

function formatDicomTime(value: string): string {
  if (!value) return ''
  const main = value.split('.')[0]
  if (!/^\d{2}(\d{2})?(\d{2})?$/.test(main)) return value
  const hh = main.slice(0, 2)
  const mm = main.length >= 4 ? main.slice(2, 4) : '00'
  const ss = main.length >= 6 ? main.slice(4, 6) : '00'
  return `${hh}:${mm}:${ss}`
}

function normalizeOffset(value: string): string {
  if (!/^[+-]\d{4}$/.test(value || '')) return ''
  return `${value.slice(0, 3)}:${value.slice(3)}`
}

function normalizeIanaTimeZone(value: string): string | null {
  const candidate = value.trim()
  if (!candidate) return null
  try {
    return new Intl.DateTimeFormat('en-US', { timeZone: candidate }).resolvedOptions().timeZone
  } catch {
    return null
  }
}

function fmtDateWithIntl(dt: Date, timeZone?: string) {
  const formatter = new Intl.DateTimeFormat('en-CA', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
    hourCycle: 'h23',
    timeZoneName: 'short',
    ...(timeZone ? { timeZone } : {}),
  })
  const parts = formatter.formatToParts(dt)
  const get = (type: string) => parts.find((p) => p.type === type)?.value ?? ''
  const y = get('year')
  const m = get('month')
  const d = get('day')
  const hh = get('hour')
  const mm = get('minute')
  const ss = get('second')
  const tz = get('timeZoneName') || (timeZone ?? 'Local')
  return `${y}-${m}-${d} ${hh}:${mm}:${ss} ${tz}`
}

function fmtDateUtc(dt: Date) {
  const y = dt.getUTCFullYear()
  const m = String(dt.getUTCMonth() + 1).padStart(2, '0')
  const d = String(dt.getUTCDate()).padStart(2, '0')
  const hh = String(dt.getUTCHours()).padStart(2, '0')
  const mm = String(dt.getUTCMinutes()).padStart(2, '0')
  const ss = String(dt.getUTCSeconds()).padStart(2, '0')
  return `${y}-${m}-${d} ${hh}:${mm}:${ss} UTC`
}

function Field({ label, value, sensitive }: { label: string; value: string; sensitive?: boolean }) {
  return (
    <div>
      <div style={{ fontSize: '12px', color: '#6b7280', marginBottom: '2px' }}>
        {label}
      </div>
      <div style={{
        fontFamily: 'monospace',
        color: sensitive && value ? '#ea580c' : '#111827',
        fontWeight: sensitive && value ? 600 : 400,
      }}>
        {value || '(empty)'}
        {sensitive && value && (
          <span style={{
            marginLeft: '8px',
            fontSize: '11px',
            color: '#ea580c',
            backgroundColor: '#fff7ed',
            padding: '1px 6px',
            borderRadius: '4px',
          }}>
            PHI — will be removed
          </span>
        )}
      </div>
    </div>
  )
}
