export function Hero() {
  const scrollTo = (id: string) => {
    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
  }

  return (
    <section id="hero" className="hero">
      <div className="hero__inner">
        <img src="/logo.png" alt="AEGIS — Anonymization & Exchange Gateway for Imaging Studies" className="hero__logo" />
        <h1 className="hero__headline">
          Secure medical image sharing for research and clinical care
        </h1>
        <p className="hero__subtitle">
          Browser-based DICOM de-identification, automated defacing, and burned-in
          PHI detection&mdash;HIPAA-compliant, multi-cloud, all modalities.
        </p>
        <p className="hero__etymology">
          In Greek mythology, the aegis was the divine shield of Zeus and
          Athena&mdash;a symbol of protection.
        </p>
        <div className="hero__ctas">
          <button className="btn btn--primary btn--lg" onClick={() => scrollTo('contact')}>
            Get Early Access
          </button>
          <button className="btn btn--secondary btn--lg" onClick={() => scrollTo('architecture')}>
            View Architecture
          </button>
        </div>
      </div>
    </section>
  )
}
