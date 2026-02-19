import { useScrollAnimation } from '../hooks/useScrollAnimation'
import { ShieldCheckIcon, DocumentIcon, KeyIcon, ClipboardIcon, CheckCircleIcon } from './icons'

const COMPLIANCE_BADGES = [
  {
    icon: <ShieldCheckIcon />,
    title: 'HIPAA Compliant',
    description: 'Full compliance with the Health Insurance Portability and Accountability Act Privacy and Security Rules',
  },
  {
    icon: <DocumentIcon />,
    title: 'BAA Available',
    description: 'Business Associate Agreements available for covered entities and their partners',
  },
  {
    icon: <KeyIcon />,
    title: 'Encryption at Rest & in Transit',
    description: 'TLS 1.3 in transit, AES-256 at rest. Cloud KMS managed encryption keys.',
  },
  {
    icon: <ClipboardIcon />,
    title: 'Complete Audit Trail',
    description: 'Every action logged with actor, timestamp, and detail. Immutable audit records for compliance.',
  },
]

const SECURITY_HIGHLIGHTS = [
  'DICOM PS3.15 Annex E Basic Confidentiality Profile',
  'Client-side de-identification before upload',
  'Automated burned-in PHI detection (OCR)',
  'Server-side facial feature removal',
  'Role-based access control (RBAC)',
  'Multi-cloud deployment (GCP, AWS, Azure)',
]

export function Trust() {
  const { ref, isVisible } = useScrollAnimation()

  return (
    <section id="trust" className="section section--alt">
      <div className="section__inner" ref={ref}>
        <h2 className={`section__title animate animate--fade-up ${isVisible ? 'animate--visible' : ''}`}>
          Built for Compliance
        </h2>
        <p className={`section__subtitle animate animate--fade-up animate--delay-1 ${isVisible ? 'animate--visible' : ''}`}>
          AEGIS addresses every layer of medical image privacy
        </p>
        <div className="trust__badges">
          {COMPLIANCE_BADGES.map((badge, i) => (
            <div
              key={badge.title}
              className={`trust__badge-card animate animate--fade-up animate--delay-${i + 1} ${isVisible ? 'animate--visible' : ''}`}
            >
              <div className="trust__badge-icon">{badge.icon}</div>
              <h3 className="trust__badge-title">{badge.title}</h3>
              <p className="trust__badge-desc">{badge.description}</p>
            </div>
          ))}
        </div>
        <div className={`trust__highlights animate animate--fade-up animate--delay-5 ${isVisible ? 'animate--visible' : ''}`}>
          <h3 className="trust__heading">Security Highlights</h3>
          <div className="trust__list">
            {SECURITY_HIGHLIGHTS.map((item) => (
              <div key={item} className="trust__list-item">
                <CheckCircleIcon size={20} />
                <span>{item}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}
