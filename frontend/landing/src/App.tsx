import { useState } from 'react'
import './App.css'
import { Navbar } from './components/Navbar'
import { Hero } from './components/Hero'
import { Stats } from './components/Stats'
import { Problem } from './components/Problem'
import { Solution } from './components/Solution'
import { HowItWorks } from './components/HowItWorks'
import Demo from './components/Demo'
import PipelineDemo from './components/PipelineDemo'
import { Trust } from './components/Trust'
import { Audiences } from './components/Audiences'
import { Testimonials } from './components/Testimonials'
import { Architecture } from './components/Architecture'
import { Roadmap } from './components/Roadmap'
import { Contact } from './components/Contact'
import { FinalCTA } from './components/FinalCTA'
import { Footer } from './components/Footer'
import { InviteGate } from './components/InviteGate'
import { CollapsibleSection } from './components/CollapsibleSection'

export function App() {
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})
  const toggle = (id: string) =>
    setCollapsed(prev => ({ ...prev, [id]: !prev[id] }))

  return (
    <InviteGate>
      <Navbar />
      <Hero />
      <Stats />
      <Problem />
      <CollapsibleSection id="solution" label="Features" collapsed={!!collapsed.solution} onToggle={() => toggle('solution')}>
        <Solution />
      </CollapsibleSection>
      <CollapsibleSection id="hiw" label="How It Works" collapsed={!!collapsed.hiw} onToggle={() => toggle('hiw')}>
        <HowItWorks />
      </CollapsibleSection>
      <CollapsibleSection id="demo" label="Live Demo" collapsed={!!collapsed.demo} onToggle={() => toggle('demo')}>
        <Demo />
      </CollapsibleSection>
      <CollapsibleSection id="pipeline" label="Processing Pipeline" collapsed={!!collapsed.pipeline} onToggle={() => toggle('pipeline')}>
        <PipelineDemo />
      </CollapsibleSection>
      <CollapsibleSection id="trust" label="Security & Compliance" collapsed={!!collapsed.trust} onToggle={() => toggle('trust')}>
        <Trust />
      </CollapsibleSection>
      <CollapsibleSection id="audiences" label="Who It's For" collapsed={!!collapsed.audiences} onToggle={() => toggle('audiences')}>
        <Audiences />
      </CollapsibleSection>
      <CollapsibleSection id="testimonials" label="Testimonials" collapsed={!!collapsed.testimonials} onToggle={() => toggle('testimonials')}>
        <Testimonials />
      </CollapsibleSection>
      <CollapsibleSection id="architecture" label="Architecture" collapsed={!!collapsed.architecture} onToggle={() => toggle('architecture')}>
        <Architecture />
      </CollapsibleSection>
      <CollapsibleSection id="roadmap" label="Roadmap" collapsed={!!collapsed.roadmap} onToggle={() => toggle('roadmap')}>
        <Roadmap />
      </CollapsibleSection>
      <Contact />
      <FinalCTA />
      <Footer />
    </InviteGate>
  )
}
