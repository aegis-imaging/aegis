import { Link, Outlet } from 'react-router-dom'
import '../styles/about.css'

// AboutLayout is the public chrome around /about/*. It deliberately does not
// call /api/auth/me — these routes are reachable without authentication so
// the GCP IAP path-based config can route them through the no-IAP backend
// service (admin_public). The "Sign in" button points at the apex root,
// which falls through to the IAP-gated backend and triggers the standard
// Google login flow.
export function AboutLayout() {
  return (
    <div className="about-shell">
      <header className="about-topbar">
        <Link to="/about" className="about-brand">
          AEGIS
        </Link>
        <nav className="about-topnav" aria-label="About">
          <Link to="/about">About</Link>
          <a
            href="https://github.com/aegis-imaging/aegis"
            target="_blank"
            rel="noopener noreferrer"
          >
            Source
          </a>
        </nav>
        <Link to="/" className="about-signin">
          Sign in
        </Link>
      </header>
      <main>
        <Outlet />
      </main>
      <footer className="about-footer">
        <div className="about-container">
          <p>
            AEGIS — Anonymization &amp; Exchange Gateway for Imaging Studies.
            Academic research platform.
          </p>
        </div>
      </footer>
    </div>
  )
}
