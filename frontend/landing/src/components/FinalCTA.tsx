export function FinalCTA() {
  const scrollTo = (id: string) => {
    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
  }

  return (
    <section className="final-cta">
      <div className="final-cta__inner">
        <h2 className="final-cta__title">Ready to secure your imaging data?</h2>
        <p className="final-cta__subtitle">
          AEGIS is live on GCP and ready for pilot deployments.
          Schedule a walkthrough to see real de-identified imaging data in action.
        </p>
        <div className="final-cta__buttons">
          <button className="btn btn--white btn--lg" onClick={() => scrollTo('contact')}>
            Schedule Demo
          </button>
          <a href="mailto:contact@aegisimaging.ai" className="btn btn--ghost-white btn--lg">
            Talk to Us
          </a>
        </div>
      </div>
    </section>
  )
}
