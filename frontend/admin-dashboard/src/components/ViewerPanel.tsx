import { useState, useEffect } from 'react'

const DWV_BASE  = import.meta.env.VITE_DWV_BASE_URL  || 'http://localhost:3005'
const OHIF_BASE = import.meta.env.VITE_OHIF_BASE_URL || 'http://localhost:3006'

type ViewerEngine = 'dwv' | 'ohif'

function useViewerEngine(): [ViewerEngine, (e: ViewerEngine) => void] {
  const [engine, setEngine] = useState<ViewerEngine>(() => {
    try {
      return (localStorage.getItem('aegis_viewer_engine') as ViewerEngine) || 'dwv'
    } catch {
      return 'dwv'
    }
  })
  const set = (e: ViewerEngine) => {
    setEngine(e)
    try { localStorage.setItem('aegis_viewer_engine', e) } catch {}
  }
  return [engine, set]
}

export function ViewerPanel({ studyUID, onClose }: { studyUID: string; onClose: () => void }) {
  const [engine, setEngine] = useViewerEngine()

  // DWV uses ?studyUID= (singular); OHIF uses ?StudyInstanceUIDs= (plural)
  const dwvUrl  = `${DWV_BASE}/viewer?studyUID=${studyUID}`
  const ohifUrl = `${OHIF_BASE}/viewer?StudyInstanceUIDs=${studyUID}`
  const url = engine === 'ohif' ? ohifUrl : dwvUrl

  // Reset to DWV when a new study is opened (avoids OHIF loading stale data)
  // — intentionally omitted: user preference should persist across studies.

  return (
    <div className="viewer-panel">
      <div className="viewer-panel-header">
        <span className="viewer-panel-title">AEGIS DICOM Viewer</span>
        <div className="viewer-panel-actions">
          {/* Viewer engine toggle */}
          <div className="viewer-engine-toggle" title="Switch viewer">
            <button
              type="button"
              className={`viewer-engine-btn${engine === 'dwv' ? ' viewer-engine-btn--active' : ''}`}
              onClick={() => setEngine('dwv')}
              aria-pressed={engine === 'dwv' ? 'true' : 'false'}
            >
              DWV
            </button>
            <button
              type="button"
              className={`viewer-engine-btn${engine === 'ohif' ? ' viewer-engine-btn--active' : ''}`}
              onClick={() => setEngine('ohif')}
              aria-pressed={engine === 'ohif' ? 'true' : 'false'}
            >
              OHIF
            </button>
          </div>
          <a
            href={url}
            target="_blank"
            rel="noreferrer"
            className="viewer-open-tab"
          >
            Open in new tab ↗
          </a>
          <button type="button" className="btn-icon" onClick={onClose} aria-label="Close viewer">
            ×
          </button>
        </div>
      </div>
      <iframe
        key={url}
        src={url}
        className="viewer-iframe"
        title={engine === 'ohif' ? 'OHIF DICOM Viewer' : 'DWV DICOM Viewer'}
        allow="fullscreen"
      />
    </div>
  )
}
