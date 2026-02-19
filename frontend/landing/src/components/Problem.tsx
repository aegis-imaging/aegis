import { useScrollAnimation } from '../hooks/useScrollAnimation'
import { WarningIcon, FaceIcon, EyeIcon, DocumentIcon } from './icons'

const PROBLEMS = [
  {
    icon: <WarningIcon />,
    title: 'De-identification tools fail',
    body: 'A 2015 study tested 10 free DICOM de-identification tools with default settings. Only 1 of 10 removed all required PHI.',
    citation: 'Aryanto et al., European Radiology, 2015',
  },
  {
    icon: <FaceIcon />,
    title: 'Faces can be reconstructed',
    body: 'Automated face-recognition software matched de-identified brain MRI participants to photographs in 83% of cases.',
    citation: 'Schwarz et al., New England Journal of Medicine, 2019',
  },
  {
    icon: <EyeIcon />,
    title: 'Burned-in PHI is invisible',
    body: 'Patient names, dates, and accession numbers are routinely overlaid on image pixels. Tag-level tools miss this entirely.',
    citation: 'Vcelak et al., International Journal of Medical Informatics, 2019',
  },
  {
    icon: <DocumentIcon />,
    title: 'Sharing is now mandatory',
    body: 'The NIH Data Management and Sharing Policy requires all NIH-funded investigators to share scientific data\u2014but institutions lack the tools to do it safely.',
    citation: 'NIH NOT-OD-21-013, effective January 2023',
  },
]

export function Problem() {
  const { ref, isVisible } = useScrollAnimation()

  return (
    <section id="problem" className="section">
      <div className="section__inner" ref={ref}>
        <h2 className={`section__title animate animate--fade-up ${isVisible ? 'animate--visible' : ''}`}>
          The Problem
        </h2>
        <p className={`section__subtitle animate animate--fade-up animate--delay-1 ${isVisible ? 'animate--visible' : ''}`}>
          Medical image sharing is broken
        </p>
        <div className="problem__grid">
          {PROBLEMS.map((p, i) => (
            <div
              key={p.title}
              className={`problem__card animate animate--fade-up animate--delay-${i + 1} ${isVisible ? 'animate--visible' : ''}`}
            >
              <div className="problem__card-icon">{p.icon}</div>
              <h3 className="problem__card-title">{p.title}</h3>
              <p className="problem__card-body">{p.body}</p>
              <p className="problem__card-citation">{p.citation}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
