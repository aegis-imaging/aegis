import { Link } from 'react-router-dom'

export function FinalCTA() {
  return (
    <section className="final-cta">
      <div className="final-cta__inner">
        <h2 className="final-cta__title">Ready to secure your imaging data?</h2>
        <p className="final-cta__subtitle">
          AEGIS is live on GCP, AWS, and Azure — ready for pilot deployments.
          Schedule a walkthrough to see real de-identified imaging data in action.
        </p>
        <div className="final-cta__buttons">
          <Link to="/contact" className="btn btn--white btn--lg">
            Schedule Demo
          </Link>
          <a href="mailto:contact@aegisimaging.ai" className="btn btn--ghost-white btn--lg">
            Talk to Us
          </a>
        </div>
      </div>
    </section>
  )
}
