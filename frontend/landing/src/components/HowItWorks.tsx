import { useScrollAnimation } from '../hooks/useScrollAnimation'
import { CheckIcon, ArrowRightIcon, ShieldCheckIcon } from './icons'

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
  const { ref, isVisible } = useScrollAnimation()

  return (
    <section id="how-it-works" className="section">
      <div className="section__inner" ref={ref}>
        <h2 className={`section__title animate animate--fade-up ${isVisible ? 'animate--visible' : ''}`}>
          How It Works
        </h2>
        <p className={`section__subtitle animate animate--fade-up animate--delay-1 ${isVisible ? 'animate--visible' : ''}`}>
          Two-phase de-identification addresses every known gap
        </p>
        <div className="hiw__phases">
          <div className={`hiw__phase hiw__phase--browser animate animate--fade-up animate--delay-2 ${isVisible ? 'animate--visible' : ''}`}>
            <div className="hiw__phase-header">
              <span className="hiw__phase-number">1</span>
              <div>
                <h3 className="hiw__phase-title">In the browser</h3>
                <p className="hiw__phase-label">Before upload</p>
              </div>
            </div>
            <ul className="hiw__steps">
              {PHASE_1_STEPS.map((s) => (
                <li key={s} className="hiw__step">
                  <CheckIcon size={18} />
                  {s}
                </li>
              ))}
            </ul>
          </div>

          <div className={`hiw__connector animate animate--fade-in animate--delay-3 ${isVisible ? 'animate--visible' : ''}`}>
            <ArrowRightIcon size={32} />
          </div>

          <div className={`hiw__phase hiw__phase--server animate animate--fade-up animate--delay-4 ${isVisible ? 'animate--visible' : ''}`}>
            <div className="hiw__phase-header">
              <span className="hiw__phase-number">2</span>
              <div>
                <h3 className="hiw__phase-title">On the server</h3>
                <p className="hiw__phase-label">After upload</p>
              </div>
            </div>
            <ul className="hiw__steps">
              {PHASE_2_STEPS.map((s) => (
                <li key={s} className="hiw__step">
                  <CheckIcon size={18} />
                  {s}
                </li>
              ))}
            </ul>
          </div>
        </div>
        <p className={`hiw__footer animate animate--fade-up animate--delay-5 ${isVisible ? 'animate--visible' : ''}`}>
          <ShieldCheckIcon size={20} />
          Nothing is shared without human sign-off.
        </p>
      </div>
    </section>
  )
}
