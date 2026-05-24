import { useEffect, useState } from 'react'
import { invoke } from '@tauri-apps/api/core'

interface PairResponse {
  api_key: string
  server_url: string
}

interface Props {
  onPaired: (cred: { server_url: string; api_key: string }) => void
  onCancel: () => void
}

/**
 * Runs automatically on first launch when the installer filename contained
 * a `-pair-{token}` segment. The Rust side returns the token via
 * get_pairing_token(); we POST it to /api/install/pair and persist the
 * resulting api_key + server_url to the OS keychain.
 *
 * The server URL the request goes TO is initially read from a hard-coded
 * default (the prod AEGIS landing) — the response carries the canonical
 * server_url back, so dev installers can repoint by overriding the default.
 */
export function FirstRunPairing({ onPaired, onCancel }: Props) {
  const [status, setStatus] = useState<'pairing' | 'error'>('pairing')
  const [error, setError] = useState<string>('')
  const [serverUrl, setServerUrl] = useState('https://api.aegisimaging.ai')

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const token = await invoke<string | null>('get_pairing_token')
        if (cancelled) return
        if (!token) {
          setError('No pairing token in installer filename.')
          setStatus('error')
          return
        }
        const pair = await invoke<PairResponse>('exchange_pairing', {
          serverUrl,
          pairingToken: token,
        })
        if (cancelled) return
        await invoke('store_credential', {
          serverUrl: pair.server_url || serverUrl,
          apiKey: pair.api_key,
        })
        onPaired({ server_url: pair.server_url || serverUrl, api_key: pair.api_key })
      } catch (e) {
        if (cancelled) return
        setError(e instanceof Error ? e.message : String(e))
        setStatus('error')
      }
    })()
    return () => {
      cancelled = true
    }
  }, [onPaired, serverUrl])

  return (
    <div style={{ maxWidth: 480, margin: '64px auto', padding: 24 }}>
      <h2 style={{ marginTop: 0 }}>Pairing with AEGIS</h2>
      {status === 'pairing' && (
        <>
          <p>Exchanging your installer's pairing token for an API key.</p>
          <p style={{ color: '#6b7280', fontSize: 13 }}>Server: {serverUrl}</p>
        </>
      )}
      {status === 'error' && (
        <>
          <div role="alert" style={{ background: '#ffedd5', color: '#9a3412', padding: 12, borderRadius: 6, marginBottom: 12 }}>
            Pairing failed: {error}
          </div>
          <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
            <input
              type="url"
              value={serverUrl}
              onChange={e => setServerUrl(e.target.value)}
              style={{ flex: 1, padding: 6 }}
            />
            <button
              onClick={() => {
                setStatus('pairing')
                setError('')
              }}
            >
              Retry
            </button>
          </div>
          <button onClick={onCancel}>Enter server URL and API key manually</button>
        </>
      )}
    </div>
  )
}
