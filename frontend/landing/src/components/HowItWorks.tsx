const PHASE_1_STEPS = [
  'DICOM tags stripped per PS3.15 Basic Confidentiality Profile',
  '18 HIPAA Safe Harbor identifiers addressed',
  'Before-and-after tag comparison preview',
  'Only de-identified data is transmitted',
]

const PHASE_2_STEPS = [
  'OCR scan for burned-in pixel PHI',
  'Automated defacing for head imaging',
  'Acquisition protocol compliance check',
  'Quality control and BIDS conversion',
  'Administrator review and approval',
]

export function HowItWorks() {
  return (
    <section id="how-it-works" className="section">
      <div className="section__inner">
        <h2 className="section__title">How It Works</h2>
        <p className="section__subtitle">
          Two-phase de-identification addresses every known gap
        </p>
        <div className="hiw__phases">
          <div className="hiw__phase hiw__phase--browser">
            <div className="hiw__phase-header">
              <span className="hiw__phase-number">1</span>
              <div>
                <h3 className="hiw__phase-title">In the browser</h3>
                <p className="hiw__phase-label">Before upload</p>
              </div>
            </div>
            <ul className="hiw__steps">
              {PHASE_1_STEPS.map((s) => (
                <li key={s} className="hiw__step">{s}</li>
              ))}
            </ul>
          </div>

          <div className="hiw__arrow">&#x2192;</div>

          <div className="hiw__phase hiw__phase--server">
            <div className="hiw__phase-header">
              <span className="hiw__phase-number">2</span>
              <div>
                <h3 className="hiw__phase-title">On the server</h3>
                <p className="hiw__phase-label">After upload</p>
              </div>
            </div>
            <ul className="hiw__steps">
              {PHASE_2_STEPS.map((s) => (
                <li key={s} className="hiw__step">{s}</li>
              ))}
            </ul>
          </div>
        </div>
        <p className="hiw__footer">
          Nothing is shared without human sign-off.
        </p>
      </div>
    </section>
  )
}
