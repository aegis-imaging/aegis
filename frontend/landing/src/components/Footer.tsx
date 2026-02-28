import { Link } from 'react-router-dom'

export function Footer() {
  return (
    <footer className="footer">
      <div className="footer__inner">
        <div className="footer__brand-col">
          <Link to="/" className="footer__brand">
            <img src="/logo.png" alt="AEGIS" className="footer__logo" />
            <span className="footer__wordmark">AEGIS</span>
          </Link>
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
          <Link to="/product" className="footer__link">Features</Link>
          <Link to="/product" className="footer__link">How It Works</Link>
          <Link to="/product" className="footer__link">Compliance</Link>
          <Link to="/technology" className="footer__link">Architecture</Link>
        </div>

        <div className="footer__col">
          <h4 className="footer__heading">Resources</h4>
          <Link to="/demos" className="footer__link">Live Demos</Link>
          <Link to="/contact" className="footer__link">Early Access</Link>
          <a href="mailto:contact@aegisimaging.ai" className="footer__link">Support</a>
        </div>

        <div className="footer__col">
          <h4 className="footer__heading">Company</h4>
          <Link to="/contact" className="footer__link">Contact</Link>
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
