import { useEffect, useState } from 'react'

// useDarkMode is a single source of truth for the light/dark theme across
// the whole app. It used to live as inline useState inside App.tsx, which
// meant only routes that rendered <App/> (i.e. /admin/*) could read or
// flip the theme. Now any component — TopBar, DashboardLayout, HomePage,
// etc. — can call useDarkMode() and they all stay in sync via the
// localStorage `aegis_theme` key and a `data-theme` attribute on
// document.documentElement.
//
// The attribute on documentElement is what nav.css's
// `:root[data-theme="dark"]` selector keys off of, so flipping it instantly
// re-theme'd every aegis-* CSS variable. Initial mount honors a saved
// preference; if none, falls back to the OS-level prefers-color-scheme.
export function useDarkMode(): [boolean, () => void] {
  const [darkMode, setDarkMode] = useState<boolean>(() => {
    if (typeof window === 'undefined') return false
    const saved = localStorage.getItem('aegis_theme')
    if (saved !== null) return saved === 'dark'
    return window.matchMedia?.('(prefers-color-scheme: dark)').matches ?? false
  })

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', darkMode ? 'dark' : 'light')
    localStorage.setItem('aegis_theme', darkMode ? 'dark' : 'light')
  }, [darkMode])

  // Listen for changes from *other* mounts of the hook (e.g. when TopBar
  // toggles the theme but the admin sidebar's button is also on screen).
  // Storage events fire on every same-origin tab/iframe — including this
  // one's other React subtrees — so we stay in sync without a context.
  useEffect(() => {
    const onStorage = (e: StorageEvent) => {
      if (e.key === 'aegis_theme' && e.newValue) {
        setDarkMode(e.newValue === 'dark')
      }
    }
    window.addEventListener('storage', onStorage)
    return () => window.removeEventListener('storage', onStorage)
  }, [])

  const toggle = () => setDarkMode(d => !d)
  return [darkMode, toggle]
}
