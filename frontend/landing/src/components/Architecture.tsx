import { useScrollAnimation } from '../hooks/useScrollAnimation'

const TECH_TAGS = [
  'Go',
  'React 19',
  'TypeScript',
  'PostgreSQL 15',
  'Python FastAPI',
  'Terraform',
  'Docker',
  'OHIF Viewer',
]

const CLOUDS = [
  { name: 'Google Cloud', status: 'supported' as const },
  { name: 'AWS', status: 'supported' as const },
  { name: 'Azure', status: 'planned' as const },
]

export function Architecture() {
  const { ref, isVisible } = useScrollAnimation()

  return (
    <section id="architecture" className="section">
      <div className="section__inner" ref={ref}>
        <h2 className={`section__title animate animate--fade-up ${isVisible ? 'animate--visible' : ''}`}>
          Architecture
        </h2>
        <p className={`section__subtitle animate animate--fade-up animate--delay-1 ${isVisible ? 'animate--visible' : ''}`}>
          Multi-cloud, built on established standards
        </p>

        <div className={`arch__diagram-wrapper animate animate--scale-in animate--delay-2 ${isVisible ? 'animate--visible' : ''}`}>
          <img
            src="/architecture.png"
            alt="AEGIS system architecture diagram"
            className="arch__diagram"
            loading="lazy"
            width="1200"
            height="800"
          />
        </div>

        <div className={`arch__details animate animate--fade-up animate--delay-3 ${isVisible ? 'animate--visible' : ''}`}>
          <div className="arch__tech">
            <h3 className="arch__heading">Tech Stack</h3>
            <div className="arch__tags">
              {TECH_TAGS.map((tag) => (
                <span key={tag} className="arch__tag">{tag}</span>
              ))}
            </div>
          </div>

          <div className="arch__clouds">
            <h3 className="arch__heading">Cloud Support</h3>
            <div className="arch__cloud-badges">
              {CLOUDS.map((c) => (
                <span
                  key={c.name}
                  className={`arch__cloud-badge arch__cloud-badge--${c.status}`}
                >
                  {c.name}
                  {c.status === 'planned' && <span className="arch__cloud-planned"> (planned)</span>}
                </span>
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
