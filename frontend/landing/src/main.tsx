import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { App, APP_URL } from './App'

// Invite emails issued while the app lived on the apex link to
// https://aegisimaging.ai/?invite=CODE — the app redeems those, not this site.
if (new URLSearchParams(window.location.search).has('invite')) {
  window.location.replace(`${APP_URL}/${window.location.search}`)
} else {
  createRoot(document.getElementById('root')!).render(
    <StrictMode>
      <App />
    </StrictMode>,
  )
}
