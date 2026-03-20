import { useState, type ReactNode } from 'react'
import { useInviteCode } from '../hooks/useInviteCode'

interface InviteGateProps {
  children: ReactNode
}

/**
 * InviteGate — wraps the upload portal and shows a private-beta access gate
 * when VITE_INVITE_GATE_ENABLED=true is set at build time and the visitor has
 * not provided a valid invite code.
 *
 * Codes are validated server-side via POST /api/invite/validate — no secret
 * token is baked into the client bundle. All CSS values are self-contained
 * (no CSS variable dependencies from the landing page theme).
 */
export function InviteGate({ children }: InviteGateProps) {
  const { admitted, gatingEnabled, autoSubmitting, submitCode } = useInviteCode()
  const [code, setCode] = useState('')
  const [error, setError] = useState(false)
  const [shaking, setShaking] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  if (!gatingEnabled || admitted) return <>{children}</>

  if (autoSubmitting) return (
    <div style={{
      minHeight: '100vh',
      background: 'linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #0f172a 100%)',
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
      background: 'linear-gradient(135deg, #0f172a 0%, #1e293b 50%, #0f172a 100%)',
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      padding: '32px 24px',
      fontFamily: 'system-ui, -apple-system, sans-serif',
    }}>
      {/* Background radial accents */}
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
        {/* Brand */}
        <div style={{ textAlign: 'center' }}>
          <div style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: '10px',
            marginBottom: '16px',
          }}>
            {/* Shield icon inline */}
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="#60a5fa" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
              <polyline points="9 12 11 14 15 10" />
            </svg>
            <span style={{
              fontSize: '22px',
              fontWeight: 700,
              color: '#ffffff',
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
            borderRadius: '9999px',
            padding: '4px 12px',
            marginBottom: '20px',
          }}>
            <span style={{
              width: '6px',
              height: '6px',
              borderRadius: '50%',
              background: '#0d9488',
              flexShrink: 0,
              boxShadow: '0 0 6px #0d9488',
            }} />
            <span style={{ fontSize: '11px', fontWeight: 600, color: '#60a5fa', letterSpacing: '0.08em', textTransform: 'uppercase' as const }}>
              Private Beta
            </span>
          </div>

          <h1 style={{
            fontSize: 'clamp(20px, 4vw, 26px)',
            fontWeight: 700,
            color: '#ffffff',
            lineHeight: '1.3',
            marginBottom: '12px',
          }}>
            Secure DICOM upload portal
          </h1>

          <p style={{
            fontSize: '14px',
            color: '#9ca3af',
            lineHeight: '1.7',
            maxWidth: '360px',
            margin: '0 auto',
          }}>
            Browser-based de-identification and multi-cloud DICOM transfer.
            Access by invitation only.
          </p>
        </div>

        {/* Gate card */}
        <div style={{
          width: '100%',
          background: 'rgba(30, 41, 59, 0.8)',
          border: `1px solid ${error ? 'rgba(234, 88, 12, 0.4)' : 'rgba(59, 130, 246, 0.2)'}`,
          borderRadius: '16px',
          padding: '32px',
          backdropFilter: 'blur(12px)',
          boxShadow: '0 20px 40px rgba(0, 0, 0, 0.3)',
          transition: 'border-color 150ms ease',
          animation: shaking ? 'shake 0.5s ease' : undefined,
        }}>
          <div style={{
            display: 'flex',
            alignItems: 'center',
            gap: '10px',
            marginBottom: '20px',
          }}>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#60a5fa" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
              <path d="M7 11V7a5 5 0 0 1 10 0v4" />
            </svg>
            <span style={{ fontSize: '14px', fontWeight: 600, color: '#ffffff' }}>
              Enter your invite code
            </span>
          </div>

          <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
            <div>
              <input
                type="text"
                value={code}
                onChange={(e) => { setCode(e.target.value); setError(false) }}
                placeholder="XXXX-XXXX-XXXX"
                autoFocus
                autoComplete="off"
                spellCheck={false}
                style={{
                  width: '100%',
                  padding: '12px 16px',
                  background: 'rgba(15, 23, 42, 0.6)',
                  border: `1.5px solid ${error ? 'rgba(234, 88, 12, 0.5)' : 'rgba(71, 85, 105, 0.6)'}`,
                  borderRadius: '8px',
                  color: '#ffffff',
                  fontSize: '16px',
                  fontFamily: 'system-ui, -apple-system, sans-serif',
                  outline: 'none',
                  transition: 'border-color 150ms ease',
                  boxSizing: 'border-box' as const,
                  letterSpacing: '0.04em',
                }}
                onFocus={(e) => { e.target.style.borderColor = '#3b82f6' }}
                onBlur={(e) => { e.target.style.borderColor = error ? 'rgba(234, 88, 12, 0.5)' : 'rgba(71, 85, 105, 0.6)' }}
              />
              {error && (
                <p style={{
                  marginTop: '6px',
                  fontSize: '12px',
                  color: '#fb923c',
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
                background: code.trim() && !submitting
                  ? 'linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%)'
                  : 'rgba(37, 99, 235, 0.3)',
                border: 'none',
                borderRadius: '8px',
                color: '#ffffff',
                fontSize: '14px',
                fontWeight: 600,
                fontFamily: 'system-ui, -apple-system, sans-serif',
                cursor: code.trim() && !submitting ? 'pointer' : 'not-allowed',
                transition: 'all 150ms ease',
                letterSpacing: '0.02em',
              }}
            >
              {submitting ? 'Checking…' : 'Access Upload Portal'}
            </button>
          </form>
        </div>

        {/* Request access */}
        <p style={{
          fontSize: '14px',
          color: '#6b7280',
          textAlign: 'center',
        }}>
          Don't have an invite?{' '}
          <a
            href="mailto:ops@aegisimaging.ai?subject=AEGIS Upload Portal Access Request"
            style={{
              color: '#60a5fa',
              textDecoration: 'none',
              fontWeight: 500,
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
