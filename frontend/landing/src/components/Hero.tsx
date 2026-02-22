import { ShieldCheckIcon, LockIcon, CloudIcon, ServerIcon } from './icons'

export function Hero() {
  const scrollTo = (id: string) => {
    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
  }

  return (
    <section id="hero" className="hero">
      <div className="hero__inner">
        <div className="hero__badge">
          <span className="hero__badge-dot" />
          HIPAA-Compliant Medical Image Sharing
        </div>

        <h1 className="hero__headline">
          Secure medical image sharing
          <br />
          <span className="hero__headline-accent">for research and clinical care</span>
        </h1>

        <p className="hero__subtitle">
          Browser-based DICOM de-identification, automated defacing, and burned-in
          PHI detection. Multi-cloud. All modalities. No software to install.
        </p>

        <div className="hero__ctas">
          <button className="btn btn--white btn--lg" onClick={() => scrollTo('demo')}>
            Try Live Demo
          </button>
          <button className="btn btn--ghost btn--lg" onClick={() => scrollTo('contact')}>
            Schedule a Demo →
          </button>
        </div>

        <div className="hero__trust-badges">
          <div className="hero__trust-badge">
            <ShieldCheckIcon size={16} />
            <span>HIPAA Compliant</span>
          </div>
          <div className="hero__trust-badge">
            <LockIcon size={16} />
            <span>BAA Available</span>
          </div>
          <div className="hero__trust-badge">
            <CloudIcon size={16} />
            <span>Multi-Cloud</span>
          </div>
          <div className="hero__trust-badge">
            <ServerIcon size={16} />
            <span>7 Processing Services</span>
          </div>
        </div>
      </div>
    </section>
  )
}
