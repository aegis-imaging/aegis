import { useState } from 'react'
import { useScrollAnimation } from '../hooks/useScrollAnimation'

/* ── Mobile detection (initial state only — no re-render on resize) ───── */
const isMobile = typeof window !== 'undefined' && window.innerWidth < 640

/* ── Tiny reusable sub-components ───────────────────────────────────────── */

function Zone({ className, title, sub, children, defaultOpen }: {
  className: string; title: string; sub?: string; children: React.ReactNode; defaultOpen?: boolean
}) {
  const [open, setOpen] = useState(defaultOpen ?? !isMobile)
  return (
    <div className={`az-zone ${className}${open ? '' : ' az-zone--collapsed'}`}>
      <button className="az-zone-hdr" onClick={() => setOpen(!open)}>
        <span className="az-zone-title">{title}</span>
        {sub && <span className="az-zone-sub">{sub}</span>}
        <span className="az-zone-chevron">{open ? '−' : '+'}</span>
      </button>
      {open && <div className="az-zone-body">{children}</div>}
    </div>
  )
}

function TabZone({ className, title, sub, tabs, defaultOpen }: {
  className: string; title: string; sub?: string
  tabs: { label: string; content: React.ReactNode }[]
  defaultOpen?: boolean
}) {
  const [open, setOpen] = useState(defaultOpen ?? !isMobile)
  const [activeTab, setActiveTab] = useState(0)
  return (
    <div className={`az-zone ${className}${open ? '' : ' az-zone--collapsed'}`}>
      <button className="az-zone-hdr" onClick={() => setOpen(!open)}>
        <span className="az-zone-title">{title}</span>
        {sub && <span className="az-zone-sub">{sub}</span>}
        <span className="az-zone-chevron">{open ? '−' : '+'}</span>
      </button>
      {open && (
        <div className="az-zone-body">
          <div className="az-tab-bar">
            {tabs.map((t, i) => (
              <button key={i} className={`az-tab${i === activeTab ? ' az-tab--active' : ''}`}
                onClick={() => setActiveTab(i)}>{t.label}</button>
            ))}
          </div>
          {tabs.map((t, i) => (
            <div key={i} className={`az-tab-panel${i === activeTab ? ' az-tab-panel--active' : ''}`}>
              {t.content}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function Card({ color, title, items }: { color: string; title: string; items: string[] }) {
  return (
    <div className={`az-card az-c-${color}`}>
      <div className="az-card-hdr">{title}</div>
      <ul>{items.map((item, i) => <li key={i}>{item}</li>)}</ul>
    </div>
  )
}

function InfoBox({ color, bg, title, sections }: {
  color: string; bg?: string; title: string
  sections: { heading?: string; items: string[] }[]
}) {
  return (
    <div className={`az-info az-c-${color}${bg ? ` az-bg-${bg}` : ''}`}>
      <div className="az-info-title">{title}</div>
      {sections.map((s, i) => (
        <div key={i}>
          {s.heading && <div className="az-info-sec">{s.heading}</div>}
          <ul>{s.items.map((item, j) => <li key={j}>{item}</li>)}</ul>
        </div>
      ))}
    </div>
  )
}

function Stage({ color, name, desc }: { color: string; name: string; desc: string }) {
  return (
    <div className={`az-stage az-c-${color}`}>
      <div className="az-stage-name">{name}</div>
      <div className="az-stage-desc">{desc}</div>
    </div>
  )
}

function Phase({ color, title, desc, dashed }: { color: string; title: string; desc: string; dashed?: boolean }) {
  return (
    <div className={`az-phase az-c-${color}${dashed ? ' az-phase--dashed' : ''}`}>
      <div className="az-phase-title">{title}</div>
      <div className="az-phase-desc">{desc}</div>
    </div>
  )
}

function KVBox({ title, rows }: { title: string; rows: { label: string; color: string; value: string }[] }) {
  return (
    <div className="az-kv-box">
      <div className="az-kv-title">{title}</div>
      {rows.map((r, i) => (
        <div key={i} className="az-kv-row">
          <span className={`az-kv-k az-kv-k--${r.color}`}>{r.label}:</span>
          <span className="az-kv-v">{r.value}</span>
        </div>
      ))}
    </div>
  )
}

/* ── Tech tags + cloud badges (kept from original) ─────────────────────── */

const TECH_TAGS = ['Go', 'React 19', 'TypeScript', 'PostgreSQL 15', 'Python FastAPI', 'Terraform', 'Docker', 'Weasis DWV']

const CLOUDS = [
  { name: 'Google Cloud', status: 'supported' as const, url: 'https://api.aegisimaging.ai/healthz' },
  { name: 'AWS', status: 'supported' as const, url: 'https://aws.api.aegisimaging.ai/healthz' },
  { name: 'Azure', status: 'supported' as const, url: 'https://azure.api.aegisimaging.ai/healthz' },
]

/* ── Main component ─────────────────────────────────────────────────────── */

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

        {/* ── Native architecture diagram ─────────────────────────────────── */}
        <div className={`az-diagram animate animate--fade-up animate--delay-2 ${isVisible ? 'animate--visible' : ''}`}>

          {/* SENDING SOURCES */}
          <Zone className="az-z-src" title="Sending Sources" sub="External Browser Upload · DICOM Network (C-STORE SCP, port 11112) · Programmatic Batch Ingest">
            <div className="az-g3">
              <Card color="react" title="React Upload Portal (PWA)" items={[
                'Drag-and-drop: files, folders, DICOM directories',
                'Client-side tag de-identification (DICOM PS3.15)',
                'Anonymization preview — before / after tag diff',
                'Multi-study detection with per-study upload cards',
                'Per-project retained-tag anonymization profiles',
                'Auto-retry: 3× exponential backoff per file',
              ]} />
              <Card color="python" title="DIMSE Receiver (pynetdicom · GCE VM)" items={[
                'Compute Engine VM — aegis-prod-dimse-receiver',
                'Static IP · TCP port 11112',
                'C-STORE SCP — receives from PACS systems / scanners',
                'Institution attribution (AE title or IP CIDR)',
                'Calls POST /api/ingest on DICOM association close',
                'Durable retry queue + dead-letter (disk-persistent)',
              ]} />
              <div className="az-privacy">
                <div className="az-privacy-title">Key Privacy Principle</div>
                <div className="az-privacy-main">PHI is stripped in the browser BEFORE data leaves the hospital.</div>
                <ul>
                  <li>Only tag-de-identified DICOM is transmitted</li>
                  <li>HIPAA Safe Harbor — 18 identifier types removed</li>
                  <li>Deterministic UID hashing + date shifting</li>
                  <li>Zero software install required at sending site</li>
                  <li>DIMSE path ingests raw files (internal-only path)</li>
                </ul>
              </div>
            </div>
          </Zone>

          {/* GCP PROJECT */}
          <TabZone className="az-z-gcp" title="GCP Project" sub="aegis-prod-488120 · us-central1 · Live, February 2026" tabs={[
            { label: 'Core', content: <>
              <div className="az-lb-bar">
                <strong>Cloud Armor (DDoS / WAF)</strong>
                <span>+ Global HTTPS Load Balancer</span>
                <span>+ Identity-Aware Proxy (admin routes)</span>
                <span>+ Managed SSL cert v3</span>
                <span>+ Cloud Run (API · Dashboard · Landing · 10 sidecars)</span>
              </div>
              <div className="az-core-grid">
                <Card color="go" title="API Backend (Go / Cloud Run)" items={[
                  'Upload orchestration (sessions, chunked PUT)',
                  'Study / project / institution management',
                  'Routing rules engine → auto-pipeline dispatch',
                  'DICOMweb proxy (QIDO-RS + WADO-RS)',
                  'Export shares: token-auth, ZIP download',
                  'Email digest scheduler + SMTP relay',
                  'DIMSE retry control proxy',
                  'MCP tool backend + batch import CLI',
                  'distroless image — minimal CVE surface',
                ]} />
                <Card color="cyan" title="Admin Dashboard + Weasis DWV (React)" items={[
                  'Behind Identity-Aware Proxy (IAP)',
                  'Study browser: filter, search, paginate, bulk ops',
                  'Weasis DWV — yoked before/after defacing review',
                  '7-stage pipeline visualization per study',
                  'RBAC: admin (write) + viewer (read-only)',
                  'Routing rules, institutions, anon profiles',
                  'Protocol templates, API keys, webhooks',
                  'Audit log + CSV export, share management',
                  'Federation peers, project lifecycle',
                ]} />
                <div className="az-infra-stack">
                  <Card color="amber" title="Cloud SQL (PostgreSQL 15)" items={[
                    'Studies, projects, institutions, admin users',
                    'Routing rules, audit trail, export shares',
                    'Private IP · Secret Manager credentials',
                    'Point-in-time recovery (7-day retention)',
                  ]} />
                  <Card color="gcp-blue" title="Cloud Storage (GCS)" items={[
                    'dicom/raw/{uid}/ — tag-de-identified',
                    'dicom/clean/{uid}/ — defaced + processed',
                    'bids/{uid}/ — NIfTI / BIDS output',
                    'analytics/{uid}/ — segmentation + metrics',
                    'Shared volume (Go API + all 10 sidecars)',
                  ]} />
                </div>
              </div>
            </> },
            { label: 'Sidecars', content: <>
              <div className="az-sidecar-label">Processing Sidecars (Python / Cloud Run) — 9 services dispatched async by Go API</div>
              <div className="az-g5">
                <Card color="python" title="Defacing" items={['Head MRI / PET / CT facial removal', 'mri_reface / DeepDefacer / mri_deface', 'nibabel fallback for dev', 'Writes to clean/ store', 'SSIM-based QA score']} />
                <Card color="python" title="PHI Detection" items={['Burned-in text OCR on pixels', 'Gemini / Vision / Azure / Textract / Tesseract', 'Pixel redaction — detect + mask', 'LLM text scrubbing', 'Per-project confidence config']} />
                <Card color="python" title="QC Automation" items={['File integrity + DICOM tag check', 'Slice consistency', 'SNR estimation', 'Coverage completeness', 'Missing slice detection']} />
                <Card color="python" title="Classification" items={['Fills modality + body_part', 'Heuristic: tags → SOP UID → description', 'Cloud AI fallback', 'Re-evaluates routing rules', 'Confidence threshold gating']} />
                <Card color="python" title="BIDS Conversion" items={['DICOM → NIfTI via dcm2niix', 'BIDS-compliant directory structure', 'sub-{hash8}/anat | func | dwi | perf', 'JSON sidecar metadata', 'UID-hashed subject labels']} />
              </div>
              <div className="az-g4">
                <Card color="python" title="Protocol Check" items={['Verifies TR / TE / flip / thickness', 'Classic + Enhanced DICOM', 'Per-project protocol templates', 'numeric / exact / range match', 'critical / warning / info severity']} />
                <Card color="python" title="Analytics — 18 Backends" items={['Brain: FreeSurfer · SynthSeg · BrainSuite · volBrain', 'Spine: TotalSpineSeg · SPINEPS', 'Whole-body: TotalSegmentator · nnU-Net · MedSAM2', 'Specialized: PETSurfer · QSM · BASIL · FSL · ANTs', 'Longitudinal: TBM-SyN + FreeSurfer Long']} />
                <Card color="python" title="SCT — Spinal Cord Toolbox" items={['Spinal cord segmentation', 'Vertebral labeling', 'CSA per vertebral level', 'Compression metrics: aMCC, aSCOR', 'DTI mapping: FA, MD, AD, RD']} />
                <Card color="python" title="Synth MRI" items={['Synthetic brain MRI generation', 'CPU: Shepp-Logan phantom', 'GPU: MONAI BraTS LDM', 'T1w contrast, Rician noise', 'Seeded + reproducible']} />
              </div>
            </> },
            { label: 'Security & Export', content: <>
              <div className="az-g4">
                <InfoBox color="orange" bg="rose" title="Security Layers" sections={[{ items: [
                  'VPC private subnets + Cloud NAT', 'Cloud Armor DDoS / WAF', 'TLS 1.2+ on all endpoints',
                  'CMEK (Cloud KMS)', 'IAM least privilege', 'Secret Manager (DB creds, API keys)',
                  'distroless containers', 'IAP on all admin routes', 'No PHI in email / audit entries',
                ]}]} />
                <InfoBox color="teal2" bg="mint" title="Export & Sharing" sections={[
                  { heading: 'Export Portal (React)', items: ['Token-authenticated share links', 'Study info, expiry countdown', 'ZIP download of approved DICOM'] },
                  { heading: 'DICOM Forwarding', items: ['DICOMweb STOW-RS destinations', 'DIMSE C-STORE to remote AE Title', 'Cross-cloud: GCP↔AWS STOW-RS live'] },
                  { heading: 'Share Lifecycle', items: ['Extend / revoke anytime', 'Immutable download log', 'Export analytics dashboard'] },
                ]} />
                <InfoBox color="purple" bg="lavender" title="MCP Server + AI Agent Tools" sections={[
                  { heading: 'Model Context Protocol', items: ['96 read tools: studies, audit, stats, routing', '95 write tools (confirm + reason required)', 'Zod-validated schemas'] },
                  { heading: 'Agent Orchestrator', items: ['DICOM tag provenance analysis', 'Diagnostic tools for study triage', 'Cohort report, pipeline funnel'] },
                  { heading: 'Batch Import + Webhooks', items: ['aegis-import CLI — bulk migration', '5 webhook events, HMAC-SHA256 signed', 'Machine-to-machine bearer tokens'] },
                ]} />
                <Card color="react" title="Landing Page (nginx / Cloud Run)" items={[
                  'aegisimaging.ai — public marketing site', 'React + Vite, served by nginx',
                  'Interactive DICOM demo widget', 'Schedule Demo / Contact form',
                  'LB default backend — no IAP required', 'www + apex domains on SSL cert v3',
                ]} />
              </div>
            </> },
          ]} />

          {/* MULTI-CLOUD PARITY */}
          <Zone className="az-z-aws" title="Multi-Cloud Parity" sub="Same Go API, same 9 Python sidecars, same Docker images — only the cloud-native primitives change">
            <div className="az-parity-note">All three clouds run identical application code. The table below shows only what differs per cloud.</div>
            <div className="az-parity-table">
              <div className="az-parity-hdr">
                <span></span><span className="az-parity-cloud az-parity-cloud--gcp">GCP</span><span className="az-parity-cloud az-parity-cloud--aws">AWS</span><span className="az-parity-cloud az-parity-cloud--azure">Azure</span>
              </div>
              <div className="az-parity-row">
                <span className="az-parity-label">Compute</span><span>Cloud Run</span><span>ECS Fargate</span><span>Container Apps</span>
              </div>
              <div className="az-parity-row">
                <span className="az-parity-label">Database</span><span>Cloud SQL</span><span>RDS</span><span>PostgreSQL Flex</span>
              </div>
              <div className="az-parity-row">
                <span className="az-parity-label">Storage</span><span>GCS</span><span>S3</span><span>Azure Blob</span>
              </div>
              <div className="az-parity-row">
                <span className="az-parity-label">Auth</span><span>IAP</span><span>ALB + Cognito</span><span>Easy Auth (AD)</span>
              </div>
              <div className="az-parity-row">
                <span className="az-parity-label">WAF / DDoS</span><span>Cloud Armor</span><span>ALB WAF</span><span>NSG rules</span>
              </div>
              <div className="az-parity-row">
                <span className="az-parity-label">Secrets</span><span>Secret Manager</span><span>Secrets Manager</span><span>Managed Identity</span>
              </div>
              <div className="az-parity-row">
                <span className="az-parity-label">Encryption</span><span>Cloud KMS (CMEK)</span><span>KMS (S3+RDS)</span><span>Key Vault</span>
              </div>
              <div className="az-parity-row">
                <span className="az-parity-label">DIMSE VM</span><span>GCE (e2-small)</span><span>EC2 (t3.small)</span><span>Azure VM (B2s)</span>
              </div>
              <div className="az-parity-row">
                <span className="az-parity-label">CI/CD</span><span>Cloud Build</span><span>GitHub Actions</span><span>GitHub Actions (OIDC)</span>
              </div>
              <div className="az-parity-row">
                <span className="az-parity-label">Registry</span><span>Artifact Registry</span><span>ECR</span><span>ACR</span>
              </div>
              <div className="az-parity-row">
                <span className="az-parity-label">Status</span><span>Live</span><span>Live</span><span>Live</span>
              </div>
            </div>
            <div className="az-cross-cloud">
              <div className="az-cross-cloud-title">Secure Cross-Instance Connectivity</div>
              <div className="az-cross-cloud-desc">
                Every AEGIS instance — cloud or on-premises — can securely route, replicate, and share
                approved studies with any other instance. Connections are configurable per-project via
                routing rules and fire automatically on study approval.
              </div>
              <div className="az-cross-cloud-paths">
                <div className="az-cross-cloud-path">
                  <span className="az-cross-cloud-badge az-cross-cloud-badge--gcp">GCP</span>
                  <span className="az-cross-cloud-arrow">&harr;</span>
                  <span className="az-cross-cloud-badge az-cross-cloud-badge--aws">AWS</span>
                  <span className="az-cross-cloud-method">STOW-RS + DIMSE</span>
                </div>
                <div className="az-cross-cloud-path">
                  <span className="az-cross-cloud-badge az-cross-cloud-badge--gcp">GCP</span>
                  <span className="az-cross-cloud-arrow">&harr;</span>
                  <span className="az-cross-cloud-badge az-cross-cloud-badge--azure">Azure</span>
                  <span className="az-cross-cloud-method">STOW-RS + DIMSE</span>
                </div>
                <div className="az-cross-cloud-path">
                  <span className="az-cross-cloud-badge az-cross-cloud-badge--aws">AWS</span>
                  <span className="az-cross-cloud-arrow">&harr;</span>
                  <span className="az-cross-cloud-badge az-cross-cloud-badge--azure">Azure</span>
                  <span className="az-cross-cloud-method">STOW-RS + DIMSE</span>
                </div>
                <div className="az-cross-cloud-path">
                  <span className="az-cross-cloud-badge az-cross-cloud-badge--onprem">On-Prem</span>
                  <span className="az-cross-cloud-arrow">&harr;</span>
                  <span className="az-cross-cloud-badge az-cross-cloud-badge--any">Any Cloud</span>
                  <span className="az-cross-cloud-method">Planned</span>
                </div>
              </div>
              <ul className="az-cross-cloud-list">
                <li>Routing rules auto-forward approved studies to any registered destination</li>
                <li>DICOMweb STOW-RS (HTTP/TLS) or DIMSE C-STORE (TCP) — configurable per rule</li>
                <li>Federation peers registry enables multi-tenant replication across institutions</li>
                <li>On-premises deployment target planned — same Docker images, local storage mode</li>
              </ul>
            </div>
          </Zone>

          {/* PIPELINE */}
          <Zone className="az-z-pipe" title="Automated Processing Pipeline" sub="routing rules define which steps are required → pipeline dispatches each step in dependency order">
            <div className="az-pipeline">
              <Stage color="violet" name="1. Classify" desc="fills modality / body_part" />
              <Stage color="red" name="2. PHI Scan" desc="OCR pixels, redact burned-in" />
              <Stage color="rust" name="3. Protocol" desc="TR / TE / flip vs template" />
              <Stage color="blue" name="4. Deface" desc="remove face, head imaging" />
              <Stage color="emerald" name="5. QC Check" desc="SNR · coverage · slices" />
              <Stage color="teal" name="6. BIDS" desc="NIfTI + sidecar structure" />
              <Stage color="cyan" name="7. Analytics" desc="18 backends, brain · spine" />
              <Stage color="cyan2" name="8. SCT" desc="cord CSA, compression" />
              <Stage color="gray" name="9. Export Fwd" desc="STOW-RS or DIMSE" />
            </div>
          </Zone>

          {/* NOVEL PLATFORM CAPABILITIES */}
          <Zone className="az-z-novel" title="Novel Platform Capabilities" sub="Differentiating features that set AEGIS apart from legacy PACS and imaging platforms">
            <div className="az-g3">
              <InfoBox color="purple" bg="lavender" title="AEGIS Agent — AI-Powered Operations" sections={[
                { heading: 'Model Context Protocol (MCP)', items: [
                  '191 typed tools (96 read + 95 write)',
                  'Natural language study triage and diagnostics',
                  'Automated pipeline monitoring and alerting',
                  'Cohort analysis and project health briefings',
                ] },
                { heading: 'Agent Capabilities', items: [
                  '"Why is this study stuck?" — root-cause analysis',
                  'Routing simulation and optimization',
                  'Daily ops summaries and anomaly detection',
                  'Batch operations with audit trail and confirmation',
                ] },
              ]} />
              <InfoBox color="cyan" bg="mint" title="Neuroimaging Analytics Pipelines" sections={[
                { heading: 'Brain Imaging (12 backends)', items: [
                  'FreeSurfer recon-all — cortical reconstruction',
                  'SynthSeg — contrast-agnostic segmentation',
                  'FSL / ANTs / SPM — extraction, thickness, DTI',
                  'TBM-SyN + FreeSurfer Long — longitudinal atrophy',
                ] },
                { heading: 'Spine Imaging (4 backends)', items: [
                  'Spinal Cord Toolbox — cord segmentation + CSA',
                  'TotalSpineSeg — vertebrae C1 through sacrum',
                  'SPINEPS — 14 vertebral substructure classes',
                  'MedSAM2 — general-purpose prompted segmentation',
                ] },
              ]} />
              <InfoBox color="teal2" title="Analytics Outputs Database" sections={[
                { heading: 'Structured Results', items: [
                  'Per-study segmentation volumes and ROI metrics',
                  'Cortical thickness, DTI scalars (FA, MD, AD, RD)',
                  'Spinal cord CSA, compression (aMCC, aSCOR)',
                  'QSM susceptibility maps, ASL perfusion CBF',
                ] },
                { heading: 'Longitudinal & Cohort', items: [
                  'Baseline vs follow-up atrophy rates',
                  'AD-signature composite (31-region weighted)',
                  'Per-subject cohort reports + modality coverage',
                  'BIDS-compliant NIfTI export for downstream tools',
                ] },
              ]} />
            </div>
            <div className="az-g3">
              <InfoBox color="orange" title="Zero-Install Browser De-Identification" sections={[
                { items: [
                  'DICOM PS3.15 Annex E tag anonymization runs entirely in the browser',
                  'PHI is stripped before data leaves the hospital network',
                  'HIPAA Safe Harbor: 18 identifier types removed automatically',
                  'Deterministic UID hashing + configurable date shifting',
                  'Pseudonymization with project-scoped salts',
                  'Per-project retained-tag anonymization profiles',
                  'No software install, no VPN — works from any modern browser',
                ] },
              ]} />
              <InfoBox color="blue" title="Configurable 9-Stage Auto-Pipeline" sections={[
                { items: [
                  'Routing rules engine — pattern-match by modality, body part, source, project',
                  'Rules auto-fire on ingest; pipeline dispatches required steps in dependency order',
                  'Phase 0: Classification → Phase 1: PHI + Protocol + Deface (parallel)',
                  'Phase 2: QC + BIDS (post-deface) → Phase 3: Analytics + SCT (post-BIDS)',
                  'Each step idempotent with reset/re-trigger support',
                  'Pipeline failure email alerts + stuck-study SLA monitoring',
                  'Rule simulator for dry-run evaluation before deploying changes',
                ] },
              ]} />
              <InfoBox color="green" title="Event-Driven Integration & Observability" sections={[
                { heading: 'Webhook Subscriptions', items: [
                  '5 events: approved, rejected, PHI flagged, export complete, stuck',
                  'HMAC-SHA256 signed payloads for receiver verification',
                  'Immutable delivery log with per-attempt tracking',
                ] },
                { heading: 'Observability', items: [
                  'Full audit trail — every action, every actor, every timestamp',
                  'Pipeline processing time stats (avg / P95 / min / max)',
                  'Destination health probes with auto-scheduled monitoring',
                  'Per-rule hit analytics to identify stale or unused rules',
                ] },
              ]} />
            </div>
          </Zone>

          {/* PHASES */}
          <Zone className="az-z-phases" title="Implementation Phases">
            <div className="az-g5">
              <Phase color="green" title="M1: Foundation + GCP ✓" desc="Go API · 9 Python sidecars · Upload Portal · Admin Dashboard · PostgreSQL · Terraform · CI/CD" />
              <Phase color="red" title="M2: AWS Live ✓" desc="ECS Fargate · RDS · S3 · ALB + Cognito · Cross-cloud DICOM routing" />
              <Phase color="azure" title="M3: Azure Live ✓" desc="Container Apps · PostgreSQL Flex · Azure Blob · ACR · GitHub Actions OIDC CI/CD" />
              <Phase color="crimson" title="M4: Processing & Security ✓" desc="JPEG2000 / Enhanced DICOM · Pixel redaction · Clinical trial RBAC" />
              <Phase color="cyan2" title="M5: Analytics & MIDI-B ✓" desc="18 analysis tools · SCT · Longitudinal · Biomarker DB · MIDI-B compliance" />
            </div>
            <div className="az-g4">
              <Phase dashed color="purple" title="M6: Private Beta ●" desc="Invite-gated access · Feedback · DIMSE C-MOVE / C-FIND · SLA hardening" />
              <Phase dashed color="amber" title="M7: Compliance ○" desc="BAA template · SOC 2 Type I audit · DIMSE at scale · AWS Marketplace" />
              <Phase dashed color="blue" title="M8: Enterprise Integrations ○" desc="SOC 2 Type II · HL7 FHIR · Federation · DICOM conformance v2" />
              <Phase dashed color="gray" title="M9: Enterprise GA ○" desc="On-premises · Multi-tenant SaaS · PACS/VNA native integration" />
            </div>
          </Zone>

          {/* TECH STACK */}
          <Zone className="az-z-tech" title="Tech Stack & Multi-Cloud">
            <div className="az-g2">
              <KVBox title="Key Open-Source Dependencies" rows={[
                { label: 'Go', color: 'go', value: 'suyashkumar/dicom · pgx · testcontainers-go · testify' },
                { label: 'Browser', color: 'react', value: 'dcmjs · dicomParser · Weasis DWV (MIT) · React 19 · Vite' },
                { label: 'Defacing', color: 'python', value: 'mri_reface · DeepDefacer · mri_deface · dcm2niix · nibabel' },
                { label: 'PHI/OCR', color: 'python', value: 'pytesseract · Google Cloud Vision · AWS Textract · Pillow' },
                { label: 'QC/BIDS', color: 'python', value: 'pydicom · numpy · dcm2niix · pynetdicom (C-STORE SCP)' },
                { label: 'Analytics', color: 'python', value: 'FreeSurfer · FSL · ANTs · SPM · SynthSeg · nnU-Net · TotalSegmentator' },
                { label: 'SCT/Spine', color: 'python', value: 'Spinal Cord Toolbox · TotalSpineSeg · SPINEPS · MedSAM2' },
                { label: 'Infra', color: 'gray', value: 'Terraform Google + AWS + Azure providers · Docker distroless/slim' },
              ]} />
              <KVBox title="Multi-Cloud + Test Coverage" rows={[
                { label: 'GCP', color: 'gcp', value: 'Cloud Run · Cloud SQL · GCS · IAP · Cloud Armor · KMS · Pub/Sub' },
                { label: 'AWS', color: 'orange', value: 'ECS Fargate · RDS · S3 · ALB + Cognito · ACM · Secrets Manager' },
                { label: 'Local', color: 'gray', value: 'Docker Compose · PostgreSQL 15 · Mailpit · local filesystem' },
                { label: 'Auth', color: 'orange', value: 'GCP IAP · Azure AD Easy Auth · AWS ALB+Cognito · dev auto-auth' },
                { label: 'Storage', color: 'go', value: 'STORAGE_MODE=gcs | s3 | azure | local — same Go API' },
                { label: 'Azure', color: 'azure', value: 'Container Apps · PostgreSQL Flex · Azure Blob · ACR · Easy Auth' },
                { label: 'CI/CD', color: 'teal', value: 'GitHub Actions: Go (1,072) · Python (892) · Client (166) · Cloud Build' },
                { label: 'Domains', color: 'python', value: 'aegisimaging.ai · www · api · admin — SSL cert v3' },
              ]} />
            </div>
          </Zone>

        </div>

        {/* ── Tech tags + cloud badges ───────────────────────────────────── */}
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
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
