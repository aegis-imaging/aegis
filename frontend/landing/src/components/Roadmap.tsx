import { useScrollAnimation } from '../hooks/useScrollAnimation'

const MOMENTUM = [
  { value: '7',    label: 'days to production' },
  { value: '846+', label: 'commits in week one' },
  { value: '122',  label: 'API routes' },
  { value: '370+', label: 'automated tests' },
]

type MilestoneStatus = 'done' | 'active' | 'planned'

interface Milestone {
  status: MilestoneStatus
  date: string
  title: string
  items: string[]
  tags: { label: string; style: 'done' | 'active' | 'cloud' | 'neutral' }[]
}

const MILESTONES: Milestone[] = [
  {
    status: 'done',
    date: 'Feb 17–24, 2026',
    title: 'Foundation — GCP Production',
    items: [
      '10+ services deployed: Go API, 7 Python processing sidecars, DIMSE C-STORE SCP, MCP server',
      'Full HIPAA pipeline: tag de-identification, defacing, PHI scan, QC, BIDS, protocol compliance',
      'Terraform IaC + Cloud Build CI/CD — every merge to develop auto-deploys all services',
      'Admin dashboard with Weasis DWV viewer, RBAC, audit trail, routing rules engine',
    ],
    tags: [
      { label: '✓ Live', style: 'done' },
      { label: 'GCP', style: 'cloud' },
      { label: 'Open Beta', style: 'neutral' },
    ],
  },
  {
    status: 'active',
    date: 'Q1 2026',
    title: 'Private Beta',
    items: [
      'Invite-gated access for early adopters — research institutions and imaging centers welcome',
      'Gathering feedback on de-identification workflows, PACS integration, and protocol compliance',
      'DIMSE C-MOVE / C-FIND for active PACS pull integration',
      'Hardening SLA monitoring, retention policies, and routing rules for production workloads',
    ],
    tags: [
      { label: '● Active', style: 'active' },
      { label: 'GCP', style: 'cloud' },
      { label: 'Invite Only', style: 'neutral' },
    ],
  },
  {
    status: 'planned',
    date: 'Q1–Q2 2026',
    title: 'AWS Deployment',
    items: [
      'Full AWS deployment: ECS Fargate, RDS PostgreSQL, S3 storage, ALB + Cognito auth',
      'Multi-cloud data federation — studies routable between GCP and AWS tenants',
      'AWS Marketplace listing for enterprise procurement',
      'Performance SLA dashboards and enhanced monitoring across both clouds',
    ],
    tags: [
      { label: 'Planned', style: 'neutral' },
      { label: 'AWS', style: 'cloud' },
      { label: 'Multi-Cloud', style: 'neutral' },
    ],
  },
  {
    status: 'planned',
    date: 'Q3 2026',
    title: 'Azure + SOC 2 Type II',
    items: [
      'Azure Container Apps deployment — full three-cloud feature parity',
      'SOC 2 Type II certification achieved',
      'HL7 FHIR notifications — integrate with hospital EMR/RIS systems',
      'Cross-tenant federated sharing — peer AEGIS instances can exchange approved studies',
    ],
    tags: [
      { label: 'Planned', style: 'neutral' },
      { label: 'Azure', style: 'cloud' },
      { label: 'SOC 2 Type II', style: 'neutral' },
    ],
  },
  {
    status: 'planned',
    date: 'Q4 2026',
    title: 'Enterprise GA',
    items: [
      'On-premises deployment option for institutions with strict data residency requirements',
      'PACS/VNA native query-retrieve — pull studies on demand, no push required',
      'Multi-tenant SaaS with per-organization data isolation for radiology groups',
      'Imaging data consortium marketplace — connect research networks to data sources',
    ],
    tags: [
      { label: 'Future', style: 'neutral' },
      { label: 'On-Premises', style: 'cloud' },
      { label: 'SaaS', style: 'neutral' },
    ],
  },
]

export function Roadmap() {
  const { ref: headerRef, isVisible: headerVisible } = useScrollAnimation({ threshold: 0.2 })
  const { ref: momentumRef, isVisible: momentumVisible } = useScrollAnimation({ threshold: 0.2 })
  const { ref: timelineRef, isVisible: timelineVisible } = useScrollAnimation({ threshold: 0.1 })

  return (
    <section className="section section--alt roadmap" id="roadmap">
      <div className="section__inner">
        {/* Header */}
        <div
          ref={headerRef}
          className={`roadmap__header animate animate--fade-up ${headerVisible ? 'animate--visible' : ''}`}
        >
          <h2 className="roadmap__heading">Built to move fast</h2>
          <p className="roadmap__subheading">
            From first commit to GCP production in 7 days — and a clear path to enterprise.
          </p>
        </div>

        {/* Momentum strip */}
        <div
          ref={momentumRef}
          className={`roadmap__momentum animate animate--scale-in ${momentumVisible ? 'animate--visible' : ''}`}
        >
          {MOMENTUM.map((m) => (
            <div key={m.label} className="roadmap__momentum-stat">
              <span className="roadmap__momentum-value">{m.value}</span>
              <span className="roadmap__momentum-label">{m.label}</span>
            </div>
          ))}
        </div>

        {/* Timeline */}
        <div
          ref={timelineRef}
          className={`roadmap__timeline animate animate--fade-up ${timelineVisible ? 'animate--visible' : ''}`}
        >
          {MILESTONES.map((m, i) => (
            <div
              key={m.title}
              className={`roadmap__milestone roadmap__milestone--${m.status} animate animate--fade-up animate--delay-${i + 1 as 1|2|3|4|5}${timelineVisible ? ' animate--visible' : ''}`}
            >
              <div className="roadmap__milestone-node" aria-hidden="true">
                {m.status === 'done' ? '✓' : m.status === 'active' ? '●' : '○'}
              </div>
              <div className="roadmap__milestone-card">
                <div className="roadmap__milestone-date">{m.date}</div>
                <div className="roadmap__milestone-title">{m.title}</div>
                <ul className="roadmap__milestone-list">
                  {m.items.map((item) => (
                    <li key={item}>{item}</li>
                  ))}
                </ul>
                <div className="roadmap__milestone-tags">
                  {m.tags.map((tag) => (
                    <span key={tag.label} className={`roadmap__tag roadmap__tag--${tag.style}`}>
                      {tag.label}
                    </span>
                  ))}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
