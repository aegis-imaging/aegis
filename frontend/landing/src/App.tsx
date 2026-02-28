import { useEffect } from 'react'
import { Routes, Route, useLocation } from 'react-router-dom'
import './App.css'
import { Navbar } from './components/Navbar'
import { Footer } from './components/Footer'
import { InviteGate } from './components/InviteGate'
import { HomePage } from './pages/HomePage'
import { ProductPage } from './pages/ProductPage'
import { DemosPage } from './pages/DemosPage'
import { TechnologyPage } from './pages/TechnologyPage'
import { ContactPage } from './pages/ContactPage'

function ScrollToTop() {
  const { pathname } = useLocation()
  useEffect(() => {
    window.scrollTo(0, 0)
  }, [pathname])
  return null
}

export function App() {
  return (
    <InviteGate>
      <Navbar />
      <ScrollToTop />
      <main>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/product" element={<ProductPage />} />
          <Route path="/demos" element={<DemosPage />} />
          <Route path="/technology" element={<TechnologyPage />} />
          <Route path="/contact" element={<ContactPage />} />
        </Routes>
      </main>
      <Footer />
    </InviteGate>
  )
}
