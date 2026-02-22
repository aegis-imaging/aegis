import './App.css'
import { Navbar } from './components/Navbar'
import { Hero } from './components/Hero'
import { Stats } from './components/Stats'
import { Problem } from './components/Problem'
import { Solution } from './components/Solution'
import { HowItWorks } from './components/HowItWorks'
import Demo from './components/Demo'
import { Trust } from './components/Trust'
import { Audiences } from './components/Audiences'
import { Testimonials } from './components/Testimonials'
import { Architecture } from './components/Architecture'
import { Contact } from './components/Contact'
import { FinalCTA } from './components/FinalCTA'
import { Footer } from './components/Footer'

export function App() {
  return (
    <>
      <Navbar />
      <Hero />
      <Stats />
      <Problem />
      <Solution />
      <HowItWorks />
      <Demo />
      <Trust />
      <Audiences />
      <Testimonials />
      <Architecture />
      <Contact />
      <FinalCTA />
      <Footer />
    </>
  )
}
