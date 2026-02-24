import { useScrollAnimation } from '../hooks/useScrollAnimation'

const STATS = [
  { value: '7', label: 'Automated Processing Services' },
  { value: '18', label: 'HIPAA Identifiers Addressed' },
  { value: '370+', label: 'Automated Tests' },
  { value: '3', label: 'Cloud Platforms Supported' },
]

export function Stats() {
  const { ref, isVisible } = useScrollAnimation({ threshold: 0.3 })

  return (
    <section className="stats" ref={ref}>
      <div className={`stats__inner animate animate--scale-in ${isVisible ? 'animate--visible' : ''}`}>
        {STATS.map((stat) => (
          <div key={stat.label} className="stats__item">
            <span className="stats__number">{stat.value}</span>
            <span className="stats__label">{stat.label}</span>
          </div>
        ))}
      </div>
    </section>
  )
}
