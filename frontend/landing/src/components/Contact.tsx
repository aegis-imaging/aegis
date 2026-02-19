import { useState } from 'react'
import { useScrollAnimation } from '../hooks/useScrollAnimation'

const ROLES = [
  '',
  'Researcher',
  'Radiologist',
  'IT / Infrastructure',
  'Executive',
  'Other',
]

export function Contact() {
  const [name, setName] = useState('')
  const [email, setEmail] = useState('')
  const [organization, setOrganization] = useState('')
  const [role, setRole] = useState('')
  const [message, setMessage] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const [error, setError] = useState('')
  const { ref, isVisible } = useScrollAnimation()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    setSubmitting(true)
    try {
      const res = await fetch('/api/contact', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, email, organization, role, message }),
      })
      if (res.ok) {
        setSubmitted(true)
      } else {
        setError('Something went wrong. Please try again or email contact@aegisimaging.ai.')
      }
    } catch {
      setError('Network error. Please try again or email contact@aegisimaging.ai.')
    }
    setSubmitting(false)
  }

  if (submitted) {
    return (
      <section id="contact" className="section section--alt">
        <div className="section__inner">
          <div className="contact__success">
            <h2 className="section__title">Thank you!</h2>
            <p className="section__subtitle">
              We&rsquo;ll be in touch. In the meantime, feel free to reach out
              at <a href="mailto:contact@aegisimaging.ai">contact@aegisimaging.ai</a>.
            </p>
          </div>
        </div>
      </section>
    )
  }

  return (
    <section id="contact" className="section section--alt">
      <div className="section__inner" ref={ref}>
        <h2 className={`section__title animate animate--fade-up ${isVisible ? 'animate--visible' : ''}`}>
          Get Early Access
        </h2>
        <p className={`section__subtitle animate animate--fade-up animate--delay-1 ${isVisible ? 'animate--visible' : ''}`}>
          AEGIS is in active development. Sign up for updates or reach out to
          discuss your use case.
        </p>

        <form className={`contact__form animate animate--fade-up animate--delay-2 ${isVisible ? 'animate--visible' : ''}`} onSubmit={handleSubmit}>
          {error && <div className="contact__error">{error}</div>}

          <div className="contact__row">
            <label className="contact__label">
              Name *
              <input
                type="text"
                className="contact__input"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </label>
            <label className="contact__label">
              Email *
              <input
                type="email"
                className="contact__input"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </label>
          </div>

          <div className="contact__row">
            <label className="contact__label">
              Organization
              <input
                type="text"
                className="contact__input"
                value={organization}
                onChange={(e) => setOrganization(e.target.value)}
              />
            </label>
            <label className="contact__label">
              Role
              <select
                className="contact__input"
                value={role}
                onChange={(e) => setRole(e.target.value)}
              >
                {ROLES.map((r) => (
                  <option key={r} value={r}>
                    {r || 'Select a role...'}
                  </option>
                ))}
              </select>
            </label>
          </div>

          <label className="contact__label">
            Message
            <textarea
              className="contact__input contact__textarea"
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              rows={4}
            />
          </label>

          <button
            type="submit"
            className="btn btn--primary btn--lg contact__submit"
            disabled={submitting}
          >
            {submitting ? 'Sending...' : 'Join the Waitlist'}
          </button>
        </form>

        <p className="contact__direct">
          Or email us directly at{' '}
          <a href="mailto:contact@aegisimaging.ai">contact@aegisimaging.ai</a>
        </p>
      </div>
    </section>
  )
}
