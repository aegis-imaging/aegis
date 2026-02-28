import { useScrollAnimation } from '../hooks/useScrollAnimation'

const MOMENTUM = [
  { value: '7',     label: 'days to production' },
  { value: '1,200+', label: 'git commits' },
  { value: '303+',  label: 'API routes' },
  { value: '750+',  label: 'automated tests' },
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
      'Microservices architecture: Go API + 9 Python microservices, DIMSE C-STORE SCP, MCP server — 12+ services total',
      'Full HIPAA pipeline: tag de-identification, defacing, PHI scan, pixel redaction, QC, BIDS, protocol compliance, analytics',
      'Terraform IaC + Cloud Build CI/CD — every merge to develop auto-deploys all services across 3 clouds',
      'Admin dashboard with Weasis DWV viewer, RBAC, audit trail, routing rules engine, clinical trial access control',
      'AI-native operations: MCP server with 50+ read tools, 64+ write tools, agent orchestrator with DICOM diagnostics',
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
    status: 'done',
    date: 'Feb 26, 2026',
    title: 'Azure Deployment',
    items: [
      'Azure Container Apps deployment — full three-cloud feature parity live',
      'Terraform IaC + GitHub Actions CI/CD (OIDC federated auth) — same auto-deploy pattern as GCP and AWS',
      'Azure Database for PostgreSQL (Flexible Server), Azure Blob Storage, Azure Container Registry',
      'DIMSE receiver on Azure Linux VM — static IP, TCP port 11112',
    ],
    tags: [
      { label: '✓ Live', style: 'done' },
      { label: 'Azure', style: 'cloud' },
      { label: 'Multi-Cloud', style: 'neutral' },
    ],
    liveUrl: 'https://azure.api.aegisimaging.ai/healthz',
  },
  {
    status: 'done',
    date: 'Feb 27, 2026',
    title: 'Advanced Processing & Security',
    items: [
      'Neuroimaging analytics: FreeSurfer, FSL, ANTs, SPM pipelines on BIDS-converted data (Phase 3)',
      'Comprehensive DICOM format support: JPEG2000, JPEG-LS, Enhanced multi-frame, Siemens Mosaic',
      'Pixel redaction pipeline: automated detection and masking of burned-in PHI in pixel data',
      'Clinical trial access control: project membership roles, capability-based write guards',
      '8 phases of cross-cloud security hardening with CI enforcement',
    ],
    tags: [
      { label: '✓ Complete', style: 'done' },
      { label: 'Analytics', style: 'neutral' },
      { label: 'Security', style: 'neutral' },
    ],
  },
  {
    status: 'planned',
    date: 'Q2 2026',
    title: 'Beta Hardening + Compliance Foundations',
    items: [
      'Production email — SMTP for share notifications, pipeline alerts, and digest subscriptions',
      'Business Associate Agreement (BAA) template + countersigning workflow — required before real PHI',
      'SOC 2 Type I audit initiation — point-in-time controls assessment; engages auditor and starts 6-month Type II observation window',
      'DIMSE C-MOVE / C-FIND — AEGIS actively queries and retrieves studies from PACS rather than waiting for push',
      'AWS Marketplace listing submission — enterprise procurement channel',
    ],
    tags: [
      { label: 'Target', style: 'neutral' },
      { label: 'Compliance', style: 'neutral' },
      { label: 'PACS', style: 'neutral' },
    ],
  },
  {
    status: 'planned',
    date: 'Q3 2026',
    title: 'SOC 2 Type II + Enterprise Integrations',
    items: [
      'SOC 2 Type II observation period underway — 6-month audit window; final report expected Q4 2026',
      'HL7 FHIR notifications — fire DiagnosticReport / ImagingStudy resources to Epic, Cerner, and other EMR/RIS on study events',
      'DICOM conformance statement v2 — formal PS3.2 statement across all three clouds; required for PACS vendor certification',
      'Cross-tenant federated sharing — peer AEGIS instances exchange approved studies across institutions',
    ],
    tags: [
      { label: 'Vision', style: 'neutral' },
      { label: 'SOC 2 Type II', style: 'neutral' },
      { label: 'Enterprise', style: 'neutral' },
    ],
  },
  {
    status: 'planned',
    date: 'Q4 2026',
    title: 'Enterprise GA',
    items: [
      'SOC 2 Type II report issued — formal certification after 6-month observation period',
      'On-premises deployment — Helm chart / Docker Compose for air-gapped and data-residency-constrained institutions',
      'Multi-tenant SaaS mode — per-organization database isolation; tenant provisioning API for radiology groups',
      'PACS/VNA native integration — full C-MOVE / C-FIND at enterprise scale across modalities and VNA vendors',
    ],
    tags: [
      { label: 'Vision', style: 'neutral' },
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
            From first commit to three-cloud production in 9 days — and a clear path to enterprise with the right team.
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
              className={`roadmap__milestone roadmap__milestone--${m.status} animate animate--fade-up animate--delay-${i + 1 as 1|2|3|4|5|6|7}${timelineVisible ? ' animate--visible' : ''}`}
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
