import { Link, Outlet } from 'react-router-dom'
import '../styles/about.css'

// AboutLayout is the public chrome around /about/*. It deliberately does not
// call /api/auth/me — these routes are reachable without authentication so
// the GCP IAP path-based config can route them through the no-IAP backend
// service (admin_public).
//
// "Sign in" MUST be a plain <a href> (full page navigation), NOT a
// react-router <Link> (client-side navigation). Client-side nav stays
// inside the React app at /about, hits RootRedirect, gets 401 from
// /api/auth/me, and bounces straight back to /about — infinite loop.
// A real <a href="/"> takes the browser through the IAP-gated backend,
// which triggers the Google OAuth flow.
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
        <a href="/" className="about-signin">
          Sign in
        </a>
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
