import { useScrollAnimation } from '../hooks/useScrollAnimation'
import { QuoteIcon } from './icons'

export function Testimonials() {
  const { ref, isVisible } = useScrollAnimation()

  return (
    <section id="testimonials" className="section">
      <div className="section__inner" ref={ref}>
        <h2 className={`section__title animate animate--fade-up ${isVisible ? 'animate--visible' : ''}`}>
          Trusted by Research Teams
        </h2>
        <p className={`section__subtitle animate animate--fade-up animate--delay-1 ${isVisible ? 'animate--visible' : ''}`}>
          AEGIS is in active development with early adopters in medical imaging research
        </p>
        <div className={`testimonials__card animate animate--scale-in animate--delay-2 ${isVisible ? 'animate--visible' : ''}`}>
          <div className="testimonials__icon">
            <QuoteIcon size={40} />
          </div>
          <p className="testimonials__quote">
            Case studies and testimonials coming soon as our early access program expands.
          </p>
          <div>
            <span className="testimonials__author-name">Early Access Program</span>
            <span className="testimonials__author-role">Now accepting applications</span>
          </div>
        </div>
      </div>
    </section>
  )
}
