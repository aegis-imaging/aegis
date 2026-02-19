export function Footer() {
  const scrollTo = (id: string) => {
    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
  }

  return (
    <footer className="footer">
      <div className="footer__inner">
        <div className="footer__brand-col">
          <div className="footer__brand">
            <img src="/logo.png" alt="AEGIS" className="footer__logo" />
            <span className="footer__wordmark">AEGIS</span>
          </div>
          <p className="footer__tagline">
            Anonymization &amp; Exchange Gateway for Imaging Studies
          </p>
          <p className="footer__mythology">
            In Greek mythology, the aegis was the divine shield of Zeus and
            Athena&mdash;a symbol of protection.
          </p>
        </div>

        <div className="footer__col">
          <h4 className="footer__heading">Product</h4>
          <button className="footer__link" onClick={() => scrollTo('solution')}>Features</button>
          <button className="footer__link" onClick={() => scrollTo('how-it-works')}>How It Works</button>
          <button className="footer__link" onClick={() => scrollTo('trust')}>Compliance</button>
          <button className="footer__link" onClick={() => scrollTo('architecture')}>Architecture</button>
        </div>

        <div className="footer__col">
          <h4 className="footer__heading">Resources</h4>
          <button className="footer__link" onClick={() => scrollTo('audiences')}>Use Cases</button>
          <button className="footer__link" onClick={() => scrollTo('contact')}>Early Access</button>
          <a href="mailto:contact@aegisimaging.ai" className="footer__link">Support</a>
        </div>

        <div className="footer__col">
          <h4 className="footer__heading">Company</h4>
          <a href="mailto:contact@aegisimaging.ai" className="footer__link">Contact</a>
          <a href="https://aegisimaging.ai" className="footer__link">aegisimaging.ai</a>
        </div>
      </div>

      <div className="footer__bottom">
        <p className="footer__copyright">
          &copy; 2026 AEGIS Imaging LLC. All rights reserved.
        </p>
        <div className="footer__legal">
          <span className="footer__legal-link">Privacy Policy</span>
          <span className="footer__legal-link">Terms of Service</span>
        </div>
      </div>
    </footer>
  )
}
