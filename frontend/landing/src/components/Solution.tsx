import { useScrollAnimation } from '../hooks/useScrollAnimation'
import { GlobeIcon, ShieldLockIcon, BrainIcon, ScanIcon, ImageIcon, ClipboardIcon } from './icons'

const FEATURES = [
  {
    icon: <GlobeIcon />,
    title: 'No software to install',
    body: 'The upload portal runs entirely in the browser. No desktop app, no plugins, no IT approval at sending sites.',
  },
  {
    icon: <ShieldLockIcon />,
    title: 'Tag-level anonymization',
    body: 'DICOM PS3.15 Annex E Basic Confidentiality Profile. PHI is stripped before data leaves the hospital network.',
  },
  {
    icon: <BrainIcon />,
    title: 'Automated defacing',
    body: 'Head MRI, CT, and PET undergo server-side facial feature removal. Before-and-after review by administrators.',
  },
  {
    icon: <ScanIcon />,
    title: 'Burned-in PHI detection',
    body: 'OCR scans image pixels for overlaid text. Studies with detected pixel-level PHI are flagged for review.',
  },
  {
    icon: <ImageIcon />,
    title: 'All DICOM modalities',
    body: 'MRI, CT, PET, ultrasound, X-ray, mammography, nuclear medicine, and more\u2014from a single platform.',
  },
  {
    icon: <ClipboardIcon />,
    title: 'Centralized audit trail',
    body: 'Every upload, approval, rejection, and export is logged. Full traceability for HIPAA compliance.',
  },
]

export function Solution() {
  const { ref, isVisible } = useScrollAnimation()

  return (
    <section id="solution" className="section section--alt">
      <div className="section__inner" ref={ref}>
        <h2 className={`section__title animate animate--fade-up ${isVisible ? 'animate--visible' : ''}`}>
          The Solution
        </h2>
        <p className={`section__subtitle animate animate--fade-up animate--delay-1 ${isVisible ? 'animate--visible' : ''}`}>
          AEGIS makes secure image sharing as easy as uploading a file
        </p>
        <div className="solution__grid">
          {FEATURES.map((f, i) => (
            <div
              key={f.title}
              className={`solution__card animate animate--fade-up animate--delay-${i + 1} ${isVisible ? 'animate--visible' : ''}`}
            >
              <div className="solution__card-icon">{f.icon}</div>
              <h3 className="solution__card-title">{f.title}</h3>
              <p className="solution__card-body">{f.body}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
