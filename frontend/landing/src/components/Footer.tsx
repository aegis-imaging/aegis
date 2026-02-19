export function Footer() {
  return (
    <footer className="footer">
      <div className="footer__inner">
        <div className="footer__brand">
          <img src="/logo.png" alt="AEGIS" className="footer__logo" />
          <p className="footer__tagline">
            Anonymization &amp; Exchange Gateway for Imaging Studies
          </p>
        </div>

        <div className="footer__links">
          <h4 className="footer__heading">Links</h4>
          <a href="https://github.com/msenjem/AEGIS" className="footer__link" target="_blank" rel="noopener noreferrer">
            GitHub
          </a>
          <a href="mailto:contact@aegisimaging.ai" className="footer__link">
            Contact
          </a>
          <a href="https://aegisimaging.ai" className="footer__link">
            aegisimaging.ai
          </a>
        </div>

        <div className="footer__info">
          <h4 className="footer__heading">About</h4>
          <p className="footer__text">
            In Greek mythology, the aegis was the divine shield of Zeus and
            Athena&mdash;a symbol of protection.
          </p>
        </div>
      </div>

      <div className="footer__bottom">
        <p className="footer__copyright">
          &copy; 2026 AEGIS Imaging LLC. All rights reserved.
        </p>
      </div>
    </footer>
  )
}
