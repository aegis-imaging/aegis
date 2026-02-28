import { useScrollAnimation } from '../hooks/useScrollAnimation'

const TECH_TAGS = [
  'Go',
  'React 19',
  'TypeScript',
  'PostgreSQL 15',
  'Python FastAPI',
  'Terraform',
  'Docker',
  'Weasis DWV',
]

const CLOUDS = [
  { name: 'Google Cloud', status: 'supported' as const, url: 'https://api.aegisimaging.ai/healthz' },
  { name: 'AWS', status: 'supported' as const, url: 'https://aws.api.aegisimaging.ai/healthz' },
  { name: 'Azure', status: 'supported' as const, url: 'https://azure.api.aegisimaging.ai/healthz' },
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
          Microservices architecture, multi-cloud, built on established standards
        </p>

        <div className={`arch__diagram-wrapper animate animate--scale-in animate--delay-2 ${isVisible ? 'animate--visible' : ''}`}>
          <iframe
            src="/architecture.html"
            title="AEGIS system architecture diagram"
            className="arch__diagram-iframe"
            loading="lazy"
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
                c.url ? (
                  <a
                    key={c.name}
                    href={c.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className={`arch__cloud-badge arch__cloud-badge--${c.status}`}
                    title={`View live ${c.name} deployment`}
                  >
                    {c.name} ↗
                  </a>
                ) : (
                  <span
                    key={c.name}
                    className={`arch__cloud-badge arch__cloud-badge--${c.status}`}
                  >
                    {c.name}
                    <span className="arch__cloud-planned"> (planned)</span>
                  </span>
                )
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
