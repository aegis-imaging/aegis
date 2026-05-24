import { Link } from 'react-router-dom'
import '../styles/about.css'

// AboutPage is the public landing surface that visitors see when they hit
// aegisimaging.ai without an authenticated session. It folds the previous
// standalone landing app (frontend/landing/) into the unified admin-dashboard
// bundle, reframed in academic-project terms rather than as a product
// marketing site.
//
// Sections kept from the old landing because they carry research-quality
// content: "The Problem" with NIH / NEJM / European Radiology citations, the
// high-level "How it works" architectural overview. Sections dropped: market
// opportunity, sales CTAs, roadmap. The user is the sole researcher; this is
// not a SaaS funnel.
export function AboutPage() {
  return (
    <div className="about-page">
      <HeroSection />
      <ProblemSection />
      <ApproachSection />
      <ContactSection />
    </div>
  )
}

function HeroSection() {
  return (
    <section className="about-hero">
      <div className="about-container">
        <p className="about-eyebrow">AEGIS</p>
        <h1 className="about-headline">
          Anonymization &amp; Exchange Gateway
          <br />
          <span className="about-headline-accent">for Imaging Studies</span>
        </h1>
        <p className="about-subtitle">
          An academic platform for HIPAA-aware sharing of de-identified medical
          imaging data. Browser-based DICOM anonymization, automated defacing,
          burned-in-PHI detection, and a researcher-facing project / subject /
          study navigation modeled on XNAT.
        </p>
        <div className="about-actions">
          <Link to="/" className="about-btn about-btn-primary">
            Sign in
          </Link>
          <a
            href="https://github.com/aegis-imaging/aegis"
            target="_blank"
            rel="noopener noreferrer"
            className="about-btn about-btn-ghost"
          >
            View on GitHub
          </a>
        </div>
      </div>
    </section>
  )
}

// Citations kept verbatim from the previous landing's Problem.tsx — these are
// the peer-reviewed studies that originally motivated the project and are the
// strongest substantive content the old marketing site carried.
const RESEARCH_PROBLEMS = [
  {
    title: 'De-identification tools often fail',
    body:
      'A 2015 study tested ten free DICOM de-identification tools with default settings. Only one removed all required PHI.',
    citation: 'Aryanto et al., European Radiology, 2015',
  },
  {
    title: 'Faces can be reconstructed',
    body:
      'Automated face-recognition software matched de-identified brain MRI participants to their photographs in 83% of cases.',
    citation: 'Schwarz et al., New England Journal of Medicine, 2019',
  },
  {
    title: 'Burned-in PHI is invisible to tag-only tools',
    body:
      'Patient names, dates, and accession numbers are routinely overlaid into pixel data. Tools that only scrub DICOM tags miss this entirely.',
    citation:
      'Vcelak et al., International Journal of Medical Informatics, 2019',
  },
  {
    title: 'Sharing is now mandatory',
    body:
      'The NIH Data Management and Sharing Policy requires NIH-funded investigators to share scientific data, but institutions lack practical tooling to do it safely.',
    citation: 'NIH NOT-OD-21-013, effective January 2023',
  },
]

function ProblemSection() {
  return (
    <section className="about-section">
      <div className="about-container">
        <h2 className="about-section-title">Why this project exists</h2>
        <p className="about-section-lead">
          Medical image sharing is broken in ways that show up at every stage of
          the pipeline.
        </p>
        <ul className="about-problem-grid">
          {RESEARCH_PROBLEMS.map((p) => (
            <li key={p.title} className="about-problem-card">
              <h3 className="about-problem-title">{p.title}</h3>
              <p className="about-problem-body">{p.body}</p>
              <p className="about-problem-citation">{p.citation}</p>
            </li>
          ))}
        </ul>
      </div>
    </section>
  )
}

function ApproachSection() {
  return (
    <section className="about-section about-section-alt">
      <div className="about-container">
        <h2 className="about-section-title">Approach</h2>
        <div className="about-approach-grid">
          <ApproachItem
            title="Browser-side anonymization"
            body="DICOM tags are scrubbed in the user's browser before upload — the server never sees the original PatientID, PatientName, or BirthDate. Pseudonyms are derived deterministically so the same subject hashes to the same value across uploads."
          />
          <ApproachItem
            title="Automated defacing"
            body="Brain MRI volumes go through a facial-feature removal pass (mri_deface / DeepDefacer) before any sharing. A visual QA score flags volumes where the deface may have under- or over-applied."
          />
          <ApproachItem
            title="Burned-in PHI detection"
            body="OCR scans pixel data for overlaid text — patient names, dates, accession numbers — that tag-only scrubbers miss. Detections gate the study from leaving the project."
          />
          <ApproachItem
            title="Project → Subject → Study navigation"
            body="Researcher UI is modeled on XNAT's familiar hierarchy. Projects scope access via ACLs; subjects group studies by pseudonymized identifier; studies expose the full pipeline state (defacing, QC, BIDS conversion, analytics)."
          />
          <ApproachItem
            title="Cloud-neutral storage"
            body="The same Go API runs on GCP Cloud Run (production today), AWS ECS Fargate, and Azure Container Apps. Storage backend switches between Cloud Storage, S3, and Azure Blob via a single STORAGE_MODE env var."
          />
          <ApproachItem
            title="DICOMweb + BIDS conversion"
            body="Each anonymized study is queryable via DICOMweb and downloadable as a BIDS-compliant NIfTI tree, so it slots into existing neuroimaging workflows (FreeSurfer, FSL, ANTs, SCT) without bespoke loaders."
          />
        </div>
      </div>
    </section>
  )
}

function ApproachItem({ title, body }: { title: string; body: string }) {
  return (
    <div className="about-approach-card">
      <h3 className="about-approach-title">{title}</h3>
      <p className="about-approach-body">{body}</p>
    </div>
  )
}

function ContactSection() {
  return (
    <section className="about-section">
      <div className="about-container about-container-narrow">
        <h2 className="about-section-title">Contact</h2>
        <p className="about-section-lead">
          AEGIS is an academic research project. For collaboration inquiries,
          access requests, or technical questions, reach out at{' '}
          <a href="mailto:contact@aegisimaging.ai">contact@aegisimaging.ai</a>.
        </p>
        <p className="about-section-lead">
          Source code is on{' '}
          <a
            href="https://github.com/aegis-imaging/aegis"
            target="_blank"
            rel="noopener noreferrer"
          >
            GitHub
          </a>
          . Existing collaborators with invite credentials can{' '}
          <Link to="/">sign in</Link> to the application.
        </p>
      </div>
    </section>
  )
}
