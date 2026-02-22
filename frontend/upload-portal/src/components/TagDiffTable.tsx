import type { DicomTag, TagAction } from '../types'
import { ACTION_LABELS } from '../dicom/tags'

interface TagDiffTableProps {
  tags: DicomTag[]
  privateTagsRemoved: number
}

const ACTION_COLORS: Record<TagAction, string> = {
  X: '#ea580c', // orange — removed (colorblind-safe: not red)
  Z: '#d97706', // amber — zeroed
  D: '#d97706', // amber — dummy
  U: '#2563eb', // blue — UID replaced
  C: '#7c3aed', // purple — cleaned
  K: '#0d9488', // teal — kept (colorblind-safe: not green)
}

const ACTION_BG: Record<TagAction, string> = {
  X: '#fff7ed',
  Z: '#fffbeb',
  D: '#fffbeb',
  U: '#eff6ff',
  C: '#faf5ff',
  K: '#f0fdfa',
}

export function TagDiffTable({ tags, privateTagsRemoved }: TagDiffTableProps) {
  // Split into modified and kept for better readability
  const modified = tags.filter(t => t.action !== 'K')
  const kept = tags.filter(t => t.action === 'K' && t.originalValue !== undefined)

  return (
    <div style={{ fontSize: '13px' }}>
      <h3 style={{ margin: '0 0 12px', fontSize: '16px', fontWeight: 600 }}>
        Anonymization Preview
      </h3>

      <div style={{
        display: 'flex', gap: '12px', marginBottom: '16px', flexWrap: 'wrap',
      }}>
        <StatBadge label="Tags modified" count={modified.length} color="#d97706" />
        <StatBadge label="Tags removed" count={modified.filter(t => t.action === 'X').length} color="#dc2626" />
        <StatBadge label="UIDs replaced" count={modified.filter(t => t.action === 'U').length} color="#2563eb" />
        <StatBadge label="Private tags removed" count={privateTagsRemoved} color="#6b7280" />
        <StatBadge label="Tags kept" count={kept.length} color="#16a34a" />
      </div>

      {/* Modified tags */}
      <div style={{ marginBottom: '24px' }}>
        <h4 style={{ margin: '0 0 8px', fontSize: '14px', color: '#374151' }}>
          Tags to be modified or removed
        </h4>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr style={{ borderBottom: '2px solid #e5e7eb', textAlign: 'left' }}>
              <th style={thStyle}>Tag</th>
              <th style={thStyle}>Keyword</th>
              <th style={thStyle}>Action</th>
              <th style={thStyle}>Original</th>
              <th style={thStyle}>After</th>
            </tr>
          </thead>
          <tbody>
            {modified.map(tag => (
              <tr key={tag.tag} style={{ borderBottom: '1px solid #f3f4f6' }}>
                <td style={{ ...tdStyle, fontFamily: 'monospace', fontSize: '12px' }}>
                  ({tag.tag.substring(0, 4)},{tag.tag.substring(4)})
                </td>
                <td style={tdStyle}>{tag.keyword}</td>
                <td style={tdStyle}>
                  <span style={{
                    display: 'inline-block',
                    padding: '1px 8px',
                    borderRadius: '4px',
                    fontSize: '11px',
                    fontWeight: 600,
                    color: ACTION_COLORS[tag.action],
                    backgroundColor: ACTION_BG[tag.action],
                  }}>
                    {ACTION_LABELS[tag.action]}
                  </span>
                </td>
                <td style={{ ...tdStyle, fontFamily: 'monospace', maxWidth: '200px', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                  {tag.originalValue || '(empty)'}
                </td>
                <td style={{ ...tdStyle, fontFamily: 'monospace', maxWidth: '200px', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                  {tag.anonymizedValue !== undefined ? (tag.anonymizedValue || '""') : '\u2014'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Kept tags (collapsed by default) */}
      <details>
        <summary style={{ cursor: 'pointer', fontWeight: 600, fontSize: '14px', color: '#374151' }}>
          Tags kept for research ({kept.length})
        </summary>
        <table style={{ width: '100%', borderCollapse: 'collapse', marginTop: '8px' }}>
          <thead>
            <tr style={{ borderBottom: '2px solid #e5e7eb', textAlign: 'left' }}>
              <th style={thStyle}>Tag</th>
              <th style={thStyle}>Keyword</th>
              <th style={thStyle}>Value</th>
            </tr>
          </thead>
          <tbody>
            {kept.map(tag => (
              <tr key={tag.tag} style={{ borderBottom: '1px solid #f3f4f6' }}>
                <td style={{ ...tdStyle, fontFamily: 'monospace', fontSize: '12px' }}>
                  ({tag.tag.substring(0, 4)},{tag.tag.substring(4)})
                </td>
                <td style={tdStyle}>{tag.keyword}</td>
                <td style={{ ...tdStyle, fontFamily: 'monospace' }}>
                  {tag.originalValue || '(empty)'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </details>
    </div>
  )
}

function StatBadge({ label, count, color }: { label: string; count: number; color: string }) {
  return (
    <div style={{
      padding: '6px 12px',
      borderRadius: '6px',
      backgroundColor: '#f9fafb',
      border: '1px solid #e5e7eb',
    }}>
      <span style={{ fontWeight: 700, color, marginRight: '6px' }}>{count}</span>
      <span style={{ color: '#6b7280' }}>{label}</span>
    </div>
  )
}

const thStyle: React.CSSProperties = {
  padding: '8px 12px',
  fontSize: '12px',
  fontWeight: 600,
  color: '#6b7280',
  textTransform: 'uppercase',
  letterSpacing: '0.05em',
}

const tdStyle: React.CSSProperties = {
  padding: '6px 12px',
  whiteSpace: 'nowrap',
}
