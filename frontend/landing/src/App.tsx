import './App.css'
import { Navbar } from './components/Navbar'
import { Hero } from './components/Hero'
import { Problem } from './components/Problem'
import { Solution } from './components/Solution'
import { HowItWorks } from './components/HowItWorks'
import { Audiences } from './components/Audiences'
import { Architecture } from './components/Architecture'
import { Contact } from './components/Contact'
import { Footer } from './components/Footer'

export function App() {
  return (
    <>
      <Navbar />
      <Hero />
      <Problem />
      <Solution />
      <HowItWorks />
      <Audiences />
      <Architecture />
      <Contact />
      <Footer />
    </>
  )
}
