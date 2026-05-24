interface UploadState {
  uploaded: number
  total: number
  currentFile: string
  finished: boolean
  error: string | null
}

interface Props {
  state: UploadState
  onDone: () => void
}

export function UploadProgress({ state, onDone }: Props) {
  const pct = state.total === 0 ? 0 : Math.round((state.uploaded / state.total) * 100)

  return (
    <div style={{ marginTop: 16 }}>
      <h2 style={{ fontSize: 16, marginBottom: 8 }}>
        {state.finished ? 'Upload complete' : state.error ? 'Upload failed' : 'Uploading…'}
      </h2>
      <div
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={pct}
        style={{ width: '100%', background: '#e5e7eb', borderRadius: 4, overflow: 'hidden', height: 16 }}
      >
        <div
          style={{
            width: `${pct}%`,
            height: '100%',
            background: state.error ? '#ea580c' : '#0d9488',
            transition: 'width 200ms ease-out',
          }}
        />
      </div>
      <p style={{ marginTop: 8, fontSize: 13, color: '#374151' }}>
        {state.uploaded} of {state.total} files
        {state.currentFile && !state.finished && (
          <span style={{ color: '#6b7280', marginLeft: 8 }}>(current: {state.currentFile})</span>
        )}
      </p>
      {state.error && (
        <div role="alert" style={{ background: '#ffedd5', color: '#9a3412', padding: 12, borderRadius: 6, marginTop: 8 }}>
          {state.error}
        </div>
      )}
      {(state.finished || state.error) && (
        <button onClick={onDone} style={{ marginTop: 12, padding: '8px 16px' }}>
          Upload another study
        </button>
      )}
    </div>
  )
}
