const TECH_STACK = [
  { label: 'Backend', value: 'Go on Cloud Run / ECS Fargate' },
  { label: 'Frontend', value: 'React + TypeScript' },
  { label: 'Database', value: 'PostgreSQL 15' },
  { label: 'Viewer', value: 'OHIF Viewer v3' },
  { label: 'Infrastructure', value: 'Terraform (GCP + AWS)' },
  { label: 'Auth', value: 'IAP, Cognito, Azure AD' },
]

const CLOUDS = [
  { name: 'Google Cloud', status: 'supported' as const },
  { name: 'AWS', status: 'supported' as const },
  { name: 'Azure', status: 'planned' as const },
]

export function Architecture() {
  return (
    <section id="architecture" className="section">
      <div className="section__inner">
        <h2 className="section__title">Architecture</h2>
        <p className="section__subtitle">
          Multi-cloud, open-source, built on established standards
        </p>

        <div className="arch__diagram-wrapper">
          <img
            src="/architecture.png"
            alt="AEGIS system architecture diagram"
            className="arch__diagram"
            loading="lazy"
          />
        </div>

        <div className="arch__details">
          <div className="arch__tech">
            <h3 className="arch__heading">Tech Stack</h3>
            <dl className="arch__dl">
              {TECH_STACK.map((t) => (
                <div key={t.label} className="arch__dl-row">
                  <dt className="arch__dt">{t.label}</dt>
                  <dd className="arch__dd">{t.value}</dd>
                </div>
              ))}
            </dl>
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
