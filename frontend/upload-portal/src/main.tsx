import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { App } from './App'
import { InviteGate } from './components/InviteGate'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <InviteGate>
      <App />
    </InviteGate>
  </StrictMode>,
)
