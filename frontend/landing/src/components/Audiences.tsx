const AUDIENCES = [
  {
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
  return (
    <section id="audiences" className="section section--alt">
      <div className="section__inner">
        <h2 className="section__title">Who It&rsquo;s For</h2>
        <p className="section__subtitle">
          One platform, two markets
        </p>
        <div className="audiences__grid">
          {AUDIENCES.map((a) => (
            <div key={a.title} className={`audiences__card ${a.className}`}>
              <h3 className="audiences__card-title">{a.title}</h3>
              <ul className="audiences__list">
                {a.points.map((p) => (
                  <li key={p} className="audiences__item">{p}</li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
