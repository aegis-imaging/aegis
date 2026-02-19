import { useScrollAnimation } from '../hooks/useScrollAnimation'
import { MicroscopeIcon, HospitalIcon, CheckIcon } from './icons'

const AUDIENCES = [
  {
    icon: <MicroscopeIcon />,
    title: 'Research Teams',
    className: 'audiences__card--research',
    points: [
      'Browser upload link sent to external sites\u2014zero install',
      'NIH Data Management and Sharing Policy compliance',
      'Automated defacing, protocol QC, and BIDS conversion',
      'Per-project anonymization profiles and routing rules',
    ],
  },
  {
    icon: <HospitalIcon />,
    title: 'Radiology Departments',
    className: 'audiences__card--radiology',
    points: [
      'HIPAA-compliant de-identification gateway',
      'Batch import from PACS and network storage',
      'Routing rules, export shares, and audit trail',
      'Metadata classification and quality control',
    ],
  },
]

export function Audiences() {
  const { ref, isVisible } = useScrollAnimation()

  return (
    <section id="audiences" className="section section--alt">
      <div className="section__inner" ref={ref}>
        <h2 className={`section__title animate animate--fade-up ${isVisible ? 'animate--visible' : ''}`}>
          Who It&rsquo;s For
        </h2>
        <p className={`section__subtitle animate animate--fade-up animate--delay-1 ${isVisible ? 'animate--visible' : ''}`}>
          One platform, two markets
        </p>
        <div className="audiences__grid">
          {AUDIENCES.map((a, i) => (
            <div
              key={a.title}
              className={`audiences__card ${a.className} animate animate--fade-up animate--delay-${i + 2} ${isVisible ? 'animate--visible' : ''}`}
            >
              <div className="audiences__card-header">
                <div className="audiences__card-icon">{a.icon}</div>
                <h3 className="audiences__card-title">{a.title}</h3>
              </div>
              <ul className="audiences__list">
                {a.points.map((p) => (
                  <li key={p} className="audiences__item">
                    <CheckIcon size={18} />
                    {p}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
