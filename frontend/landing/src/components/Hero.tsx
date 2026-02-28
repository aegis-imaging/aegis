import { Link } from 'react-router-dom'
import { ShieldCheckIcon, LockIcon, CloudIcon, ServerIcon } from './icons'

export function Hero() {
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
          <Link to="/demos" className="btn btn--white btn--lg">
            Try Live Demo
          </Link>
          <Link to="/contact" className="btn btn--ghost btn--lg">
            Schedule a Demo →
          </Link>
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
            <span>18 Analysis Tools</span>
          </div>
        </div>
      </div>
    </section>
  )
}
