import { useScrollAnimation } from '../hooks/useScrollAnimation'

const MOMENTUM = [
  { value: '7',    label: 'days to production' },
  { value: '846+', label: 'commits in week one' },
  { value: '126+', label: 'API routes' },
  { value: '400+', label: 'automated tests' },
]

type MilestoneStatus = 'done' | 'active' | 'planned'

interface Milestone {
  status: MilestoneStatus
  date: string
  title: string
  items: string[]
  tags: { label: string; style: 'done' | 'active' | 'cloud' | 'neutral' }[]
  liveUrl?: string
}

const MILESTONES: Milestone[] = [
  {
    status: 'done',
    date: 'Feb 17–24, 2026',
    title: 'Foundation — GCP Production',
    items: [
      'Microservices architecture: Go API + 7 Python microservices, DIMSE C-STORE SCP, MCP server — 10+ services total',
      'Full HIPAA pipeline: tag de-identification, defacing, PHI scan, QC, BIDS, protocol compliance',
      'Terraform IaC + Cloud Build CI/CD — every merge to develop auto-deploys all services',
      'Admin dashboard with Weasis DWV viewer, RBAC, audit trail, routing rules engine',
      'AI-native operations: MCP server with 52+ read tools, 29+ write tools, agent orchestrator with DICOM diagnostics',
    ],
    tags: [
      { label: '✓ Live', style: 'done' },
      { label: 'GCP', style: 'cloud' },
      { label: 'Open Beta', style: 'neutral' },
    ],
    liveUrl: 'https://api.aegisimaging.ai/healthz',
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
    status: 'done',
    date: 'Q1 2026',
    title: 'AWS Deployment',
    items: [
      'ECS Fargate (10 microservices), RDS PostgreSQL, S3, ALB + Cognito — fully live',
      'GitHub Actions CI/CD mirrors GCP Cloud Build — auto-deploy on every push to develop',
      'DIMSE receiver on EC2 with Elastic IP — same static-IP PACS pattern as GCP',
      'Cross-cloud DICOM routing verified live — GCP→AWS STOW-RS tested, bidirectional API key auth',
    ],
    tags: [
      { label: '✓ Live', style: 'done' },
      { label: 'AWS', style: 'cloud' },
      { label: 'Multi-Cloud', style: 'neutral' },
    ],
    liveUrl: 'https://aws.api.aegisimaging.ai/healthz',
  },
  {
    status: 'active',
    date: 'Feb 26, 2026 (Day 9)',
    title: 'Azure Deployment',
    items: [
      'Azure Container Apps deployment — full three-cloud feature parity deploying now',
      'Terraform IaC + GitHub Actions CI/CD (OIDC federated auth) — same auto-deploy pattern as GCP and AWS',
      'Azure Database for PostgreSQL (Flexible Server), Azure Blob Storage, Azure Container Registry',
      'DIMSE receiver on Azure Linux VM — static Elastic IP, TCP port 11112',
    ],
    tags: [
      { label: '● Deploying', style: 'active' },
      { label: 'Azure', style: 'cloud' },
      { label: 'Day 9', style: 'neutral' },
    ],
  },
  {
    status: 'planned',
    date: 'Q3 2026',
    title: 'SOC 2 Type II + Enterprise Integrations',
    items: [
      'SOC 2 Type II certification — formal audit after 6-month observation period',
      'HL7 FHIR notifications — integrate with hospital EMR/RIS systems',
      'Cross-tenant federated sharing — peer AEGIS instances can exchange approved studies',
      'AWS Marketplace listing for enterprise procurement',
    ],
    tags: [
      { label: 'Planned', style: 'neutral' },
      { label: 'SOC 2 Type II', style: 'neutral' },
      { label: 'Enterprise', style: 'neutral' },
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
              className={`roadmap__milestone roadmap__milestone--${m.status} animate animate--fade-up animate--delay-${i + 1 as 1|2|3|4|5|6}${timelineVisible ? ' animate--visible' : ''}`}
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
                  {m.liveUrl && (
                    <a
                      href={m.liveUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="roadmap__tag roadmap__tag--done"
                    >
                      View live ↗
                    </a>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
