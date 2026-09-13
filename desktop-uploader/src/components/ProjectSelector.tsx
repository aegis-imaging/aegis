import type { Project } from '../lib/aegis-api'

interface Props {
  projects: Project[]
  selected: Project | null
  onSelect: (project: Project) => void
}

export function ProjectSelector({ projects, selected, onSelect }: Props) {
  return (
    <div>
      <label style={{ display: 'block', fontSize: 12, color: '#374151', marginBottom: 4 }}>
        Project
      </label>
      <select
        value={selected?.id ?? ''}
        onChange={e => {
          const next = projects.find(p => p.id === e.target.value)
          if (next) onSelect(next)
        }}
        style={{ minWidth: 320, padding: 6 }}
      >
        <option value="" disabled>
          {projects.length === 0 ? 'Loading projects…' : 'Select a project…'}
        </option>
        {projects.map(p => (
          <option key={p.id} value={p.id}>
            {p.name} ({p.slug})
          </option>
        ))}
      </select>
    </div>
  )
}
