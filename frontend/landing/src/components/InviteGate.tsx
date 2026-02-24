import { useState, type ReactNode } from 'react'
import { ShieldCheckIcon } from './icons'
import { useInviteCode } from '../hooks/useInviteCode'

interface InviteGateProps {
  children: ReactNode
}

/**
 * InviteGate — wraps the full landing page content and shows a private-beta
 * access gate when VITE_INVITE_GATE_ENABLED=true is set at build time and the
 * visitor has not provided a valid invite code.
 *
 * Codes are validated server-side via POST /api/invite/validate — no secret
 * token is baked into the client bundle.
 *
 * If VITE_INVITE_GATE_ENABLED is not 'true' the gate is transparent (dev / open mode).
 */
export function InviteGate({ children }: InviteGateProps) {
  const { admitted, gatingEnabled, autoSubmitting, submitCode } = useInviteCode()
  const [code, setCode] = useState('')
  const [error, setError] = useState(false)
  const [shaking, setShaking] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  // Transparent when gating is disabled or visitor is already admitted.
  if (!gatingEnabled || admitted) return <>{children}</>

  // Show a brief spinner while auto-submitting a ?invite= URL param.
  if (autoSubmitting) return (
    <div style={{
      minHeight: '100vh', background: 'var(--gradient-hero)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
    }}>
      <div style={{
        width: '40px', height: '40px', borderRadius: '50%',
        border: '3px solid rgba(59,130,246,0.2)',
        borderTop: '3px solid #3b82f6',
        animation: 'spin 1s linear infinite',
      }} />
      <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
    </div>
  )

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!code.trim() || submitting) return
    setSubmitting(true)
    const valid = await submitCode(code)
    setSubmitting(false)
    if (valid) {
      setError(false)
    } else {
      setError(true)
      setShaking(true)
      setCode('')
      setTimeout(() => setShaking(false), 600)
    }
  }

  return (
    <div style={{
      minHeight: '100vh',
      background: 'var(--gradient-hero)',
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '32px 24px',
      fontFamily: 'var(--font-sans)',
    }}>
      {/* Background grid pattern */}
      <div style={{
        position: 'fixed',
        inset: 0,
        backgroundImage: 'radial-gradient(circle at 25% 25%, rgba(59, 130, 246, 0.08) 0%, transparent 50%), radial-gradient(circle at 75% 75%, rgba(37, 99, 235, 0.06) 0%, transparent 50%)',
        pointerEvents: 'none',
      }} />

      <div style={{
        position: 'relative',
        width: '100%',
        maxWidth: '440px',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: '32px',
      }}>
        {/* Logo + Brand */}
        <div style={{ textAlign: 'center' }}>
          <div style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: '10px',
            marginBottom: '16px',
          }}>
            <img src="/logo.png" alt="AEGIS" style={{ height: '36px', width: '36px', objectFit: 'contain' }} />
            <span style={{
              fontSize: '22px',
              fontWeight: 'var(--font-bold)',
              color: 'var(--color-white)',
              letterSpacing: '0.06em',
            }}>
              AEGIS
            </span>
          </div>

          <div style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: '6px',
            background: 'rgba(59, 130, 246, 0.12)',
            border: '1px solid rgba(59, 130, 246, 0.25)',
            borderRadius: 'var(--radius-full)',
            padding: '4px 12px',
            marginBottom: '20px',
          }}>
            <span style={{
              width: '6px',
              height: '6px',
              borderRadius: '50%',
              background: '#22c55e',
              flexShrink: 0,
              boxShadow: '0 0 6px #22c55e',
            }} />
            <span style={{ fontSize: '11px', fontWeight: 'var(--font-semibold)', color: 'var(--color-blue-400)', letterSpacing: '0.08em', textTransform: 'uppercase' }}>
              Private Beta
            </span>
          </div>

          <h1 style={{
            fontSize: 'clamp(22px, 4vw, 28px)',
            fontWeight: 'var(--font-bold)',
            color: 'var(--color-white)',
            lineHeight: '1.3',
            marginBottom: '12px',
          }}>
            HIPAA-compliant medical imaging,<br />
            <span style={{ color: 'var(--color-blue-400)' }}>built for research teams</span>
          </h1>

          <p style={{
            fontSize: 'var(--text-sm)',
            color: 'var(--color-gray-400)',
            lineHeight: '1.7',
            maxWidth: '360px',
            margin: '0 auto',
          }}>
            Browser-based de-identification, automated defacing, and multi-cloud DICOM sharing.
            Currently in private beta — access by invitation.
          </p>
        </div>

        {/* Gate card */}
        <div style={{
          width: '100%',
          background: 'rgba(30, 41, 59, 0.8)',
          border: `1px solid ${error ? 'rgba(239, 68, 68, 0.4)' : 'rgba(59, 130, 246, 0.2)'}`,
          borderRadius: 'var(--radius-xl)',
          padding: '32px',
          backdropFilter: 'blur(12px)',
          boxShadow: '0 20px 40px rgba(0, 0, 0, 0.3)',
          transition: 'border-color var(--transition-fast)',
          animation: shaking ? 'shake 0.5s ease' : undefined,
        }}>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            gap: '10px',
            marginBottom: '20px',
          }}>
            <span style={{ color: 'var(--color-blue-400)', display: 'flex' }}><ShieldCheckIcon size={18} /></span>
            <span style={{ fontSize: 'var(--text-sm)', fontWeight: 'var(--font-semibold)', color: 'var(--color-white)' }}>
              Enter your invite code
            </span>
          </div>

          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div>
              <input
                type="text"
                value={code}
                onChange={(e) => { setCode(e.target.value); setError(false) }}
                placeholder="Enter invite code"
                autoFocus
                autoComplete="off"
                spellCheck={false}
                style={{
                  width: '100%',
                  padding: '12px 16px',
                  background: 'rgba(15, 23, 42, 0.6)',
                  border: `1.5px solid ${error ? 'rgba(239, 68, 68, 0.5)' : 'rgba(71, 85, 105, 0.6)'}`,
                  borderRadius: 'var(--radius-md)',
                  color: 'var(--color-white)',
                  fontSize: 'var(--text-base)',
                  fontFamily: 'var(--font-sans)',
                  outline: 'none',
                  transition: 'border-color var(--transition-fast)',
                  boxSizing: 'border-box',
                  letterSpacing: '0.04em',
                }}
                onFocus={(e) => { e.target.style.borderColor = 'var(--color-blue-500)' }}
                onBlur={(e) => { e.target.style.borderColor = error ? 'rgba(239, 68, 68, 0.5)' : 'rgba(71, 85, 105, 0.6)' }}
              />
              {error && (
                <p style={{
                  marginTop: '6px',
                  fontSize: 'var(--text-xs)',
                  color: '#f87171',
                  letterSpacing: '0.01em',
                }}>
                  Invalid invite code. Please check the code from your invitation email.
                </p>
              )}
            </div>

            <button
              type="submit"
              disabled={!code.trim() || submitting}
              style={{
                width: '100%',
                padding: '12px 24px',
                background: code.trim() && !submitting ? 'var(--gradient-cta)' : 'rgba(37, 99, 235, 0.3)',
                border: 'none',
                borderRadius: 'var(--radius-md)',
                color: 'var(--color-white)',
                fontSize: 'var(--text-sm)',
                fontWeight: 'var(--font-semibold)',
                fontFamily: 'var(--font-sans)',
                cursor: code.trim() && !submitting ? 'pointer' : 'not-allowed',
                transition: 'all var(--transition-fast)',
                letterSpacing: '0.02em',
              }}
            >
              {submitting ? 'Checking…' : 'Access Site'}
            </button>
          </form>
        </div>

        {/* Request access link */}
        <p style={{
          fontSize: 'var(--text-sm)',
          color: 'var(--color-gray-500)',
          textAlign: 'center',
        }}>
          Don't have an invite?{' '}
          <a
            href="mailto:ops@aegisimaging.ai?subject=AEGIS Beta Access Request"
            style={{
              color: 'var(--color-blue-400)',
              textDecoration: 'none',
              fontWeight: 'var(--font-medium)',
            }}
            onMouseEnter={(e) => { (e.target as HTMLAnchorElement).style.textDecoration = 'underline' }}
            onMouseLeave={(e) => { (e.target as HTMLAnchorElement).style.textDecoration = 'none' }}
          >
            Request access
          </a>
        </p>
      </div>

      <style>{`
        @keyframes shake {
          0%, 100% { transform: translateX(0); }
          20%       { transform: translateX(-8px); }
          40%       { transform: translateX(8px); }
          60%       { transform: translateX(-6px); }
          80%       { transform: translateX(6px); }
        }
      `}</style>
    </div>
  )
}
