import { useEffect, useRef, useState } from 'react'
import { useScrollAnimation } from '../hooks/useScrollAnimation'

const DIAGRAM_INTRINSIC_WIDTH = 1875

const TECH_TAGS = [
  'Go',
  'React 19',
  'TypeScript',
  'PostgreSQL 15',
  'Python FastAPI',
  'Terraform',
  'Docker',
  'Weasis DWV',
]

const CLOUDS = [
  { name: 'Google Cloud', status: 'supported' as const, url: 'https://api.aegisimaging.ai/healthz' },
  { name: 'AWS', status: 'supported' as const, url: 'https://aws.api.aegisimaging.ai/healthz' },
  { name: 'Azure', status: 'supported' as const, url: 'https://azure.api.aegisimaging.ai/healthz' },
]

export function Architecture() {
  const { ref, isVisible } = useScrollAnimation()
  const wrapperRef = useRef<HTMLDivElement>(null)
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const [scale, setScale] = useState(1)
  const [contentHeight, setContentHeight] = useState(1600)

  useEffect(() => {
    const el = wrapperRef.current
    if (!el) return
    const ro = new ResizeObserver(([entry]) => {
      const w = entry.contentRect.width
      setScale(Math.min(1, w / DIAGRAM_INTRINSIC_WIDTH))
    })
    ro.observe(el)
    return () => ro.disconnect()
  }, [])

  const handleIframeLoad = () => {
    try {
      const h = iframeRef.current?.contentDocument?.documentElement.scrollHeight
      if (h && h > 0) setContentHeight(h)
    } catch { /* cross-origin fallback: keep default */ }
  }

  return (
    <section id="architecture" className="section">
      <div className="section__inner" ref={ref}>
        <h2 className={`section__title animate animate--fade-up ${isVisible ? 'animate--visible' : ''}`}>
          Architecture
        </h2>
        <p className={`section__subtitle animate animate--fade-up animate--delay-1 ${isVisible ? 'animate--visible' : ''}`}>
          Microservices architecture, multi-cloud, built on established standards
        </p>

        <div
          ref={wrapperRef}
          className={`arch__diagram-wrapper animate animate--scale-in animate--delay-2 ${isVisible ? 'animate--visible' : ''}`}
          style={{ height: `${contentHeight * scale}px`, overflow: 'hidden' }}
        >
          <iframe
            ref={iframeRef}
            src="/architecture.html"
            title="AEGIS system architecture diagram"
            className="arch__diagram-iframe"
            loading="lazy"
            onLoad={handleIframeLoad}
            style={{
              width: `${DIAGRAM_INTRINSIC_WIDTH}px`,
              height: `${contentHeight}px`,
              transformOrigin: 'top left',
              transform: `scale(${scale})`,
            }}
          />
        </div>

        <div className={`arch__details animate animate--fade-up animate--delay-3 ${isVisible ? 'animate--visible' : ''}`}>
          <div className="arch__tech">
            <h3 className="arch__heading">Tech Stack</h3>
            <div className="arch__tags">
              {TECH_TAGS.map((tag) => (
                <span key={tag} className="arch__tag">{tag}</span>
              ))}
            </div>
          </div>

          <div className="arch__clouds">
            <h3 className="arch__heading">Cloud Support</h3>
            <div className="arch__cloud-badges">
              {CLOUDS.map((c) => (
                c.url ? (
                  <a
                    key={c.name}
                    href={c.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className={`arch__cloud-badge arch__cloud-badge--${c.status}`}
                    title={`View live ${c.name} deployment`}
                  >
                    {c.name} ↗
                  </a>
                ) : (
                  <span
                    key={c.name}
                    className={`arch__cloud-badge arch__cloud-badge--${c.status}`}
                  >
                    {c.name}
                    <span className="arch__cloud-planned"> (planned)</span>
                  </span>
                )
              ))}
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
