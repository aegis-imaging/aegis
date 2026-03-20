import { useState, useEffect } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { useTheme } from '../hooks/useTheme'

function MoonIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/>
    </svg>
  )
}

function SunIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="5"/>
      <line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/>
      <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/>
      <line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/>
      <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/>
    </svg>
  )
}

const NAV_LINKS = [
  { path: '/', label: 'Home' },
  { path: '/product', label: 'Product' },
  { path: '/demos', label: 'Demos' },
  { path: '/technology', label: 'Technology' },
  { path: '/contact', label: 'Contact' },
]

const UPLOAD_PORTAL_URL = import.meta.env.VITE_UPLOAD_PORTAL_URL || 'https://upload.aegisimaging.ai'

export function Navbar() {
  const [menuOpen, setMenuOpen] = useState(false)
  const [scrolled, setScrolled] = useState(false)
  const { theme, toggle } = useTheme()
  const { pathname } = useLocation()

  // Close mobile menu on route change
  useEffect(() => {
    setMenuOpen(false)
  }, [pathname])

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 10)
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  const isActive = (path: string) => {
    if (path === '/') return pathname === '/'
    return pathname.startsWith(path)
  }

  return (
    <nav className={`navbar ${scrolled || pathname !== '/' ? 'navbar--scrolled' : ''}`}>
      <div className="navbar__inner">
        <Link to="/" className="navbar__brand">
          <img src="/logo.png" alt="AEGIS" className="navbar__logo" />
          <span className="navbar__wordmark">AEGIS</span>
        </Link>

        <div className={`navbar__links ${menuOpen ? 'navbar__links--open' : ''}`}>
          {NAV_LINKS.map((link) => (
            <Link
              key={link.path}
              to={link.path}
              className={`navbar__link ${isActive(link.path) ? 'navbar__link--active' : ''}`}
            >
              {link.label}
            </Link>
          ))}
          <a
            href={UPLOAD_PORTAL_URL}
            className="btn btn--secondary btn--sm navbar__cta-mobile"
            target="_blank"
            rel="noopener noreferrer"
          >
            Submit Data ↗
          </a>
          <button
            className="navbar__theme-toggle navbar__theme-toggle--mobile"
            onClick={toggle}
            aria-label={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
          >
            {theme === 'light' ? <MoonIcon /> : <SunIcon />}
            <span>{theme === 'light' ? 'Dark mode' : 'Light mode'}</span>
          </button>
          <Link to="/contact" className="btn btn--primary btn--sm navbar__cta-mobile">
            Schedule Demo
          </Link>
        </div>

        <button
          className="navbar__theme-toggle"
          onClick={toggle}
          aria-label={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
          title={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
        >
          {theme === 'light' ? <MoonIcon /> : <SunIcon />}
        </button>

        <a
          href={UPLOAD_PORTAL_URL}
          className="btn btn--secondary btn--sm navbar__cta"
          target="_blank"
          rel="noopener noreferrer"
        >
          Submit Data ↗
        </a>

        <Link to="/contact" className="btn btn--primary btn--sm navbar__cta">
          Schedule Demo
        </Link>

        <button
          className="navbar__hamburger"
          onClick={() => setMenuOpen(!menuOpen)}
          aria-label="Toggle menu"
        >
          <span className={`navbar__hamburger-line ${menuOpen ? 'navbar__hamburger-line--open' : ''}`} />
          <span className={`navbar__hamburger-line ${menuOpen ? 'navbar__hamburger-line--open' : ''}`} />
          <span className={`navbar__hamburger-line ${menuOpen ? 'navbar__hamburger-line--open' : ''}`} />
        </button>
      </div>
    </nav>
  )
}
