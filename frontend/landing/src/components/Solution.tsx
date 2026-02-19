const FEATURES = [
  {
    icon: '\u{1F310}', // globe
    title: 'No software to install',
    body: 'The upload portal runs entirely in the browser. No desktop app, no plugins, no IT approval at sending sites.',
  },
  {
    icon: '\u{1F512}', // lock
    title: 'Tag-level anonymization',
    body: 'DICOM PS3.15 Annex E Basic Confidentiality Profile. PHI is stripped before data leaves the hospital network.',
  },
  {
    icon: '\u{1F9E0}', // brain
    title: 'Automated defacing',
    body: 'Head MRI, CT, and PET undergo server-side facial feature removal. Before-and-after review by administrators.',
  },
  {
    icon: '\u{1F50D}', // magnifying glass
    title: 'Burned-in PHI detection',
    body: 'OCR scans image pixels for overlaid text. Studies with detected pixel-level PHI are flagged for review.',
  },
  {
    icon: '\u{1F3E5}', // hospital
    title: 'All DICOM modalities',
    body: 'MRI, CT, PET, ultrasound, X-ray, mammography, nuclear medicine, and more\u2014from a single platform.',
  },
  {
    icon: '\u{1F4CB}', // clipboard
    title: 'Centralized audit trail',
    body: 'Every upload, approval, rejection, and export is logged. Full traceability for HIPAA compliance.',
  },
]

export function Solution() {
  return (
    <section id="solution" className="section section--alt">
      <div className="section__inner">
        <h2 className="section__title">The Solution</h2>
        <p className="section__subtitle">
          AEGIS makes secure image sharing as easy as uploading a file
        </p>
        <div className="solution__grid">
          {FEATURES.map((f) => (
            <div key={f.title} className="solution__card">
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
