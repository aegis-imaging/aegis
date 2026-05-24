import { useState } from 'react'

interface Props {
  onSubmit: (serverUrl: string, apiKey: string) => void | Promise<void>
  initialError: string | null
}

/**
 * Manual pairing fallback when the installer wasn't downloaded via the
 * personalised invite link. Operators paste a server URL + an API key minted
 * by an admin through `/api/api-keys` and we store both in the OS keychain.
 */
export function ServerSettings({ onSubmit, initialError }: Props) {
  const [serverUrl, setServerUrl] = useState('https://api.aegisimaging.ai')
  const [apiKey, setApiKey] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [err, setErr] = useState<string | null>(initialError)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setErr(null)
    if (!serverUrl.trim() || !apiKey.trim()) {
      setErr('Server URL and API key are both required.')
      return
    }
    setSubmitting(true)
    try {
      await onSubmit(serverUrl.trim().replace(/\/+$/, ''), apiKey.trim())
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} style={{ maxWidth: 480, margin: '64px auto', padding: 24 }}>
      <h2 style={{ marginTop: 0 }}>Connect to AEGIS</h2>
      <p style={{ color: '#374151' }}>
        Enter your AEGIS server URL and the API key your administrator gave you.
      </p>
      {err && (
        <div role="alert" style={{ background: '#ffedd5', color: '#9a3412', padding: 12, borderRadius: 6, marginBottom: 12 }}>
          {err}
        </div>
      )}
      <label style={{ display: 'block', marginBottom: 12 }}>
        <span style={{ display: 'block', fontSize: 12, color: '#374151', marginBottom: 4 }}>Server URL</span>
        <input
          type="url"
          value={serverUrl}
          onChange={e => setServerUrl(e.target.value)}
          placeholder="https://api.aegisimaging.ai"
          required
          style={{ width: '100%', padding: 6 }}
        />
      </label>
      <label style={{ display: 'block', marginBottom: 12 }}>
        <span style={{ display: 'block', fontSize: 12, color: '#374151', marginBottom: 4 }}>API key</span>
        <input
          type="password"
          value={apiKey}
          onChange={e => setApiKey(e.target.value)}
          placeholder="aegis_…"
          required
          autoComplete="off"
          spellCheck={false}
          style={{ width: '100%', padding: 6, fontFamily: 'monospace' }}
        />
      </label>
      <button type="submit" disabled={submitting} style={{ padding: '8px 16px' }}>
        {submitting ? 'Saving…' : 'Connect'}
      </button>
    </form>
  )
}
