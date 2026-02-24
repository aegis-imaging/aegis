import { ReactNode } from 'react'

interface CollapsibleSectionProps {
  id: string
  label: string
  collapsed: boolean
  onToggle: () => void
  children: ReactNode
}

export function CollapsibleSection({ id, label, collapsed, onToggle, children }: CollapsibleSectionProps) {
  return (
    <div className={`cs-wrap${collapsed ? ' cs-wrap--collapsed' : ''}`} id={`cs-${id}`}>
      <button
        className="cs-toggle"
        onClick={onToggle}
        aria-expanded={!collapsed}
        aria-controls={`cs-body-${id}`}
      >
        <span className="cs-toggle__label">{label}</span>
        <span className="cs-toggle__icon" aria-hidden="true">‹</span>
      </button>
      <div className="cs-body" id={`cs-body-${id}`} role="region">
        <div className="cs-body__inner">
          {children}
        </div>
      </div>
    </div>
  )
}
