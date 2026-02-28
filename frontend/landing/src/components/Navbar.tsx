import { useState, useEffect } from 'react'
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
  { id: 'problem', label: 'Problem' },
  { id: 'solution', label: 'Solution' },
  { id: 'how-it-works', label: 'How It Works' },
  { id: 'demo', label: 'Live Demo' },
  { id: 'trust', label: 'Compliance' },
  { id: 'architecture', label: 'Architecture' },
  { id: 'contact', label: 'Contact' },
]

export function Navbar() {
  const [activeSection, setActiveSection] = useState('')
  const [menuOpen, setMenuOpen] = useState(false)
  const [scrolled, setScrolled] = useState(false)
  const { theme, toggle } = useTheme()

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            setActiveSection(entry.target.id)
          }
        }
      },
      { rootMargin: '-50% 0px -50% 0px' }
    )

    const sections = document.querySelectorAll('section[id]')
    sections.forEach((section) => observer.observe(section))
    return () => observer.disconnect()
  }, [])

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 10)
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  const scrollTo = (id: string) => {
    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
    setMenuOpen(false)
  }

  return (
    <nav className={`navbar ${scrolled ? 'navbar--scrolled' : ''}`}>
      <div className="navbar__inner">
        <button className="navbar__brand" onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}>
          <img src="/logo.png" alt="AEGIS" className="navbar__logo" />
          <span className="navbar__wordmark">AEGIS</span>
        </button>

        <div className={`navbar__links ${menuOpen ? 'navbar__links--open' : ''}`}>
          {NAV_LINKS.map((link) => (
            <button
              key={link.id}
              className={`navbar__link ${activeSection === link.id ? 'navbar__link--active' : ''}`}
              onClick={() => scrollTo(link.id)}
            >
              {link.label}
            </button>
          ))}
          <button
            className="navbar__theme-toggle navbar__theme-toggle--mobile"
            onClick={toggle}
            aria-label={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
          >
            {theme === 'light' ? <MoonIcon /> : <SunIcon />}
            <span>{theme === 'light' ? 'Dark mode' : 'Light mode'}</span>
          </button>
          <button className="btn btn--primary btn--sm navbar__cta-mobile" onClick={() => scrollTo('contact')}>
            Schedule Demo
          </button>
        </div>

        <button
          className="navbar__theme-toggle"
          onClick={toggle}
          aria-label={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
          title={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
        >
          {theme === 'light' ? <MoonIcon /> : <SunIcon />}
        </button>

        <button className="btn btn--primary btn--sm navbar__cta" onClick={() => scrollTo('contact')}>
          Schedule Demo
        </button>

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
