import { Hero } from '../components/Hero'
import { Stats } from '../components/Stats'
import { Problem } from '../components/Problem'
import { Audiences } from '../components/Audiences'
import { FinalCTA } from '../components/FinalCTA'

export function HomePage() {
  return (
    <>
      <Hero />
      <Stats />
      <Problem />
      <Audiences />
      <FinalCTA />
    </>
  )
}
