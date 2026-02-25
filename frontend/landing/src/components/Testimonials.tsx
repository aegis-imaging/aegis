import { useScrollAnimation } from '../hooks/useScrollAnimation'

const MARKET_CARDS = [
  {
    stat: '$45B+',
    label: 'Medical imaging market',
    detail: 'Global market for medical imaging data management, growing ~8% YoY as research data volumes accelerate.',
    accent: '#3b82f6',
  },
  {
    stat: '2023',
    label: 'NIH data sharing mandate',
    detail: 'NIH Data Management & Sharing Policy now requires all federally funded research to share data — HIPAA-compliant infrastructure is mandatory, not optional.',
    accent: '#8b5cf6',
  },
  {
    stat: '~62%',
    label: 'Imaging studies lack compliant sharing',
    detail: 'Most institutions lack HIPAA-safe pipelines for multi-site collaboration. Manual de-identification is error-prone and does not scale.',
    accent: '#ea580c',
  },
  {
    stat: 'Day 1',
    label: 'Production-ready infrastructure',
    detail: 'AEGIS is live on GCP, AWS, and Azure — microservices architecture (Go API + 7 Python microservices) on all three clouds. Cross-cloud DICOM routing verified. Full audit trail built in.',
    accent: '#059669',
  },
]

export function Testimonials() {
  const { ref, isVisible } = useScrollAnimation()

  return (
    <section id="market" className="section" style={{ background: 'rgba(15,23,42,0.6)' }}>
      <div className="section__inner" ref={ref}>
        <div className={`animate animate--fade-up ${isVisible ? 'animate--visible' : ''}`} style={{ textAlign: 'center', marginBottom: '12px' }}>
          <span style={{
            display: 'inline-block', padding: '4px 16px', borderRadius: '20px',
            background: 'rgba(59,130,246,0.15)', color: '#60a5fa',
            fontSize: '0.77rem', fontWeight: 700, letterSpacing: '0.1em',
            textTransform: 'uppercase', marginBottom: '16px',
          }}>
            Market Opportunity
          </span>
        </div>

        <h2 className={`section__title animate animate--fade-up ${isVisible ? 'animate--visible' : ''}`}>
          The infrastructure gap in medical imaging
        </h2>
        <p className={`section__subtitle animate animate--fade-up animate--delay-1 ${isVisible ? 'animate--visible' : ''}`}>
          Regulatory mandates, massive data growth, and zero production-ready solutions —
          until now.
        </p>

        <div
          className={`animate animate--fade-up animate--delay-2 ${isVisible ? 'animate--visible' : ''}`}
          style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
            gap: '20px',
            marginTop: '40px',
          }}
        >
          {MARKET_CARDS.map((card) => (
            <div
              key={card.label}
              style={{
                background: 'rgba(15,23,42,0.85)',
                border: `1px solid ${card.accent}30`,
                borderTop: `3px solid ${card.accent}`,
                borderRadius: '12px',
                padding: '28px 24px',
                transition: 'transform 0.2s ease, box-shadow 0.2s ease',
              }}
              onMouseOver={e => {
                const el = e.currentTarget
                el.style.transform = 'translateY(-3px)'
                el.style.boxShadow = `0 8px 32px ${card.accent}18`
              }}
              onMouseOut={e => {
                const el = e.currentTarget
                el.style.transform = ''
                el.style.boxShadow = ''
              }}
            >
              <div style={{
                fontSize: '2.2rem', fontWeight: 800,
                color: card.accent, lineHeight: 1,
                marginBottom: '8px',
                fontVariantNumeric: 'tabular-nums',
              }}>
                {card.stat}
              </div>
              <div style={{
                fontSize: '0.9rem', fontWeight: 700,
                color: '#e2e8f0', marginBottom: '10px',
                lineHeight: 1.3,
              }}>
                {card.label}
              </div>
              <p style={{
                fontSize: '0.82rem', color: '#64748b',
                lineHeight: 1.6, margin: 0,
              }}>
                {card.detail}
              </p>
            </div>
          ))}
        </div>

        <div
          className={`animate animate--fade-up animate--delay-3 ${isVisible ? 'animate--visible' : ''}`}
          style={{
            marginTop: '48px',
            padding: '28px 32px',
            background: 'linear-gradient(135deg, rgba(59,130,246,0.08), rgba(139,92,246,0.08))',
            border: '1px solid rgba(59,130,246,0.2)',
            borderRadius: '14px',
            display: 'flex',
            flexWrap: 'wrap',
            gap: '20px',
            alignItems: 'center',
            justifyContent: 'space-between',
          }}
        >
          <div>
            <div style={{ fontSize: '1.1rem', fontWeight: 700, color: '#f1f5f9', marginBottom: '6px' }}>
              Ready to see the platform in action?
            </div>
            <div style={{ fontSize: '0.87rem', color: '#64748b' }}>
              Schedule a 30-minute walkthrough with real de-identified imaging data.
            </div>
          </div>
          <button
            onClick={() => document.getElementById('contact')?.scrollIntoView({ behavior: 'smooth' })}
            style={{
              padding: '12px 28px', borderRadius: '8px',
              background: 'var(--gradient-cta)', color: '#fff',
              border: 'none', cursor: 'pointer',
              fontWeight: 700, fontSize: '0.95rem',
              whiteSpace: 'nowrap', flexShrink: 0,
            }}
          >
            Schedule Demo →
          </button>
        </div>
      </div>
    </section>
  )
}
