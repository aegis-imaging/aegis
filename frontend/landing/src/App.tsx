import type { ReactNode } from 'react'
import './styles/landing.css'

export const APP_URL: string = import.meta.env.VITE_APP_URL || 'https://app.aegisimaging.ai'
const GITHUB_URL = 'https://github.com/aegis-imaging/aegis'
const CONTACT_EMAIL = 'contact@aegisimaging.ai'

const NAV = [
  { href: '#problem', label: 'Problem' },
  { href: '#approach', label: 'Approach' },
  { href: '#architecture', label: 'Architecture' },
  { href: '#stack', label: 'Stack' },
  { href: '#contact', label: 'Contact' },
]

export function App() {
  return (
    <div className="lp-shell">
      <a className="lp-skip" href="#main">
        Skip to content
      </a>
      <TopBar />
      <main id="main">
        <Hero />
        <Problem />
        <Approach />
        <Architecture />
        <Stack />
        <Contact />
      </main>
      <Footer />
    </div>
  )
}

function ExternalLink({ href, className, children }: { href: string; className?: string; children: ReactNode }) {
  return (
    <a href={href} className={className} target="_blank" rel="noopener noreferrer">
      {children}
    </a>
  )
}

function TopBar() {
  return (
    <header className="lp-topbar">
      <div className="lp-container lp-topbar-row">
        <a href="#" className="lp-brand">
          <img src="/logo.png" alt="" width="18" height="32" />
          <span>AEGIS</span>
        </a>
        <nav className="lp-nav" aria-label="Sections">
          {NAV.map((item) => (
            <a key={item.href} href={item.href}>
              {item.label}
            </a>
          ))}
        </nav>
        <div className="lp-topbar-actions">
          <ExternalLink href={GITHUB_URL} className="lp-btn lp-btn-ghost lp-btn-sm">
            GitHub
          </ExternalLink>
          <a href={APP_URL} className="lp-btn lp-btn-primary lp-btn-sm">
            Sign in
          </a>
        </div>
      </div>
    </header>
  )
}

const FACTS = [
  { title: 'Three clouds', body: 'GCP Cloud Run, AWS ECS Fargate, Azure Container Apps — one codebase' },
  { title: 'Any DICOM modality', body: 'MRI, CT, PET, ultrasound, X-ray, mammography and more' },
  { title: 'Ingest anywhere', body: 'Browser upload, DIMSE C-STORE from PACS, DICOMweb STOW-RS' },
  { title: 'Analysis-ready output', body: 'BIDS / NIfTI trees, DICOMweb access, token-gated exports' },
]

function Hero() {
  return (
    <section className="lp-hero">
      <div className="lp-container">
        <p className="lp-eyebrow">Research infrastructure · Medical imaging</p>
        <h1 className="lp-headline">
          Anonymization &amp; Exchange Gateway
          <br />
          <span className="lp-accent">for Imaging Studies</span>
        </h1>
        <p className="lp-subtitle">
          A cloud-neutral platform for HIPAA-aware sharing of de-identified medical imaging.
          Browser-side DICOM anonymization, automated defacing, burned-in-PHI detection, and an
          XNAT-style project / subject / study workspace — built once, deployed to GCP, AWS, and
          Azure.
        </p>
        <div className="lp-actions">
          <ExternalLink href={GITHUB_URL} className="lp-btn lp-btn-primary">
            View the source
          </ExternalLink>
          <a href={APP_URL} className="lp-btn lp-btn-ghost">
            Sign in to the app
          </a>
        </div>
        <ul className="lp-facts" aria-label="At a glance">
          {FACTS.map((f) => (
            <li key={f.title}>
              <strong>{f.title}</strong>
              <span>{f.body}</span>
            </li>
          ))}
        </ul>
      </div>
    </section>
  )
}

const RESEARCH_PROBLEMS = [
  {
    title: 'De-identification tools often fail',
    body: 'A 2015 study tested ten free DICOM de-identification tools with default settings. Only one removed all required PHI.',
    citation: 'Aryanto et al., European Radiology, 2015',
  },
  {
    title: 'Faces can be reconstructed',
    body: 'Automated face-recognition software matched de-identified brain MRI participants to their photographs in 83% of cases.',
    citation: 'Schwarz et al., New England Journal of Medicine, 2019',
  },
  {
    title: 'Burned-in PHI is invisible to tag-only tools',
    body: 'Patient names, dates, and accession numbers are routinely overlaid into pixel data. Tools that only scrub DICOM tags miss this entirely.',
    citation: 'Vcelak et al., International Journal of Medical Informatics, 2019',
  },
  {
    title: 'Sharing is now mandatory',
    body: 'The NIH Data Management and Sharing Policy requires NIH-funded investigators to share scientific data, but institutions lack practical tooling to do it safely.',
    citation: 'NIH NOT-OD-21-013, effective January 2023',
  },
]

function Problem() {
  return (
    <section id="problem" className="lp-section">
      <div className="lp-container">
        <h2 className="lp-section-title">Why this project exists</h2>
        <p className="lp-lead">
          Medical image sharing is broken in ways that show up at every stage of the pipeline.
        </p>
        <ul className="lp-grid lp-grid-4">
          {RESEARCH_PROBLEMS.map((p) => (
            <li key={p.title} className="lp-card">
              <h3>{p.title}</h3>
              <p>{p.body}</p>
              <p className="lp-citation">{p.citation}</p>
            </li>
          ))}
        </ul>
      </div>
    </section>
  )
}

const APPROACH = [
  {
    title: 'Browser-side anonymization',
    body: "DICOM tags are scrubbed in the user's browser before upload — the server never sees the original PatientID, PatientName, or BirthDate. Pseudonyms are derived deterministically so the same subject hashes to the same value across uploads.",
  },
  {
    title: 'Automated defacing',
    body: 'Brain MRI volumes go through a facial-feature removal pass (mri_deface / DeepDefacer) before any sharing. A visual QA score flags volumes where the deface may have under- or over-applied.',
  },
  {
    title: 'Burned-in PHI detection',
    body: 'OCR scans pixel data for overlaid text — patient names, dates, accession numbers — that tag-only scrubbers miss. Detections gate the study from leaving the project.',
  },
  {
    title: 'Project → Subject → Study navigation',
    body: "Researcher UI is modeled on XNAT's familiar hierarchy. Projects scope access via ACLs; subjects group studies by pseudonymized identifier; studies expose the full pipeline state (defacing, QC, BIDS conversion, analytics).",
  },
  {
    title: 'Cloud-neutral storage',
    body: 'The same Go API runs on GCP Cloud Run, AWS ECS Fargate, and Azure Container Apps. Storage backend switches between Cloud Storage, S3, and Azure Blob via a single STORAGE_MODE env var.',
  },
  {
    title: 'DICOMweb + BIDS conversion',
    body: 'Each anonymized study is queryable via DICOMweb and downloadable as a BIDS-compliant NIfTI tree, so it slots into existing neuroimaging workflows (FreeSurfer, FSL, ANTs, SCT) without bespoke loaders.',
  },
]

function Approach() {
  return (
    <section id="approach" className="lp-section lp-section-alt">
      <div className="lp-container">
        <h2 className="lp-section-title">Approach</h2>
        <p className="lp-lead">
          Every study is de-identified at three layers — metadata, image geometry, and pixel content
          — before a human decides whether it can leave the project.
        </p>
        <ul className="lp-grid">
          {APPROACH.map((a) => (
            <li key={a.title} className="lp-card lp-plain">
              <h3>{a.title}</h3>
              <p>{a.body}</p>
            </li>
          ))}
        </ul>
      </div>
    </section>
  )
}

const PIPELINE = [
  {
    phase: 'Ingest',
    title: 'Studies arrive already de-identified',
    body: 'Browser upload scrubs DICOM tags before data leaves the sending site. Hospitals can also push via DIMSE C-STORE or DICOMweb STOW-RS into the same pipeline.',
  },
  {
    phase: 'Phase 0',
    title: 'Classification',
    body: 'Modality and body part are detected and routing rules re-evaluated, so non-head studies skip defacing automatically.',
  },
  {
    phase: 'Phase 1',
    title: 'PHI scan · protocol check · defacing',
    body: 'OCR looks for burned-in text, acquisition parameters are checked against site templates, and head scans have facial features removed — all in parallel.',
  },
  {
    phase: 'Phase 2',
    title: 'QC · BIDS conversion',
    body: 'Automated quality checks on the defaced volume, then dcm2niix produces a BIDS-compliant NIfTI tree.',
  },
  {
    phase: 'Phase 3',
    title: 'Analytics',
    body: 'FreeSurfer, FSL, ANTs, SynthSeg, TotalSegmentator, Spinal Cord Toolbox and other backends run on the BIDS output.',
  },
  {
    phase: 'Release',
    title: 'Human sign-off',
    body: 'Reviewers compare before and after in a web DICOM viewer. Routing rules and token-authenticated shares govern what leaves the project, and every action is audited.',
  },
]

const PLATFORM = [
  {
    title: 'Go API + PostgreSQL',
    body: 'Upload orchestration, DICOMweb proxy, priority-ordered routing rules, audit trail, and multi-provider auth (GCP IAP, AWS Cognito, Azure AD, API keys).',
  },
  {
    title: 'Four React apps',
    body: 'Upload portal for sending sites, researcher / admin workspace, export portal for share recipients, and this site — React 19, TypeScript, Vite.',
  },
  {
    title: 'Python sidecars',
    body: 'FastAPI services for defacing, PHI detection, QC, BIDS, classification, protocol compliance, analytics, spinal cord, synthetic data, and DIMSE receive — each enabled by a single env var.',
  },
  {
    title: 'Infrastructure as code',
    body: 'Terraform modules for GCP, AWS, and Azure; Docker Compose for local development; GitHub Actions and Cloud Build for CI/CD.',
  },
  {
    title: 'Agent-ready',
    body: 'An MCP server exposes studies, pipeline state, and routing to LLM agents in natural language.',
  },
]

function Architecture() {
  return (
    <section id="architecture" className="lp-section">
      <div className="lp-container">
        <h2 className="lp-section-title">How a study moves through AEGIS</h2>
        <p className="lp-lead">
          Pipeline phases are dispatched automatically in dependency order; each step is a
          separate service that can be enabled, disabled, or swapped per deployment.
        </p>
        <ol className="lp-flow">
          {PIPELINE.map((step) => (
            <li key={step.phase}>
              <p className="lp-phase">{step.phase}</p>
              <h3>{step.title}</h3>
              <p>{step.body}</p>
            </li>
          ))}
        </ol>
        <div className="lp-platform">
          <h3>What the platform is made of</h3>
          <ul className="lp-grid">
            {PLATFORM.map((p) => (
              <li key={p.title} className="lp-card lp-plain">
                <h3>{p.title}</h3>
                <p>{p.body}</p>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  )
}

const STACK = [
  'Go',
  'PostgreSQL',
  'React 19',
  'TypeScript',
  'Vite',
  'Python',
  'FastAPI',
  'pydicom',
  'dcm2niix',
  'DICOMweb',
  'DIMSE',
  'BIDS / NIfTI',
  'FreeSurfer',
  'FSL',
  'ANTs',
  'Spinal Cord Toolbox',
  'Terraform',
  'Docker',
  'GCP Cloud Run',
  'AWS ECS Fargate',
  'Azure Container Apps',
  'GitHub Actions',
  'MCP',
]

function Stack() {
  return (
    <section id="stack" className="lp-section lp-section-alt">
      <div className="lp-container">
        <h2 className="lp-section-title">Stack</h2>
        <p className="lp-lead">
          Standard-library Go, functional React with strict TypeScript, and Python sidecars in slim
          containers — chosen so the same images run unchanged on every cloud.
        </p>
        <ul className="lp-chips" aria-label="Technologies">
          {STACK.map((s) => (
            <li key={s}>{s}</li>
          ))}
        </ul>
      </div>
    </section>
  )
}

function Contact() {
  return (
    <section id="contact" className="lp-section">
      <div className="lp-container lp-container-narrow">
        <h2 className="lp-section-title">Contact</h2>
        <p className="lp-lead">
          AEGIS is designed and built by Matthew L. Senjem (AEGIS Imaging LLC). For collaboration
          inquiries, access requests, or technical questions, write to{' '}
          <a href={`mailto:${CONTACT_EMAIL}`}>{CONTACT_EMAIL}</a>.
        </p>
        <p className="lp-lead">
          Source code is on <ExternalLink href={GITHUB_URL}>GitHub</ExternalLink>. Collaborators with
          invite credentials can <a href={APP_URL}>sign in</a> to the application.
        </p>
      </div>
    </section>
  )
}

function Footer() {
  return (
    <footer className="lp-footer">
      <div className="lp-container lp-footer-row">
        <p>
          © {new Date().getFullYear()} AEGIS Imaging LLC · Anonymization &amp; Exchange Gateway for
          Imaging Studies
        </p>
        <p>
          <ExternalLink href={GITHUB_URL}>GitHub</ExternalLink> · <a href={APP_URL}>Sign in</a>
        </p>
      </div>
    </footer>
  )
}
