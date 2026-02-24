import { useState } from 'react'

const OHIF_BASE   = import.meta.env.VITE_OHIF_BASE_URL   || 'http://localhost:3002'
const WEASIS_BASE = import.meta.env.VITE_WEASIS_BASE_URL || 'http://localhost:3005'

type ViewerType = 'ohif' | 'weasis'

export function ViewerPanel({ studyUID, onClose }: { studyUID: string; onClose: () => void }) {
  const [viewer, setViewer] = useState<ViewerType>('ohif')

  const ohifUrl   = `${OHIF_BASE}/viewer?StudyInstanceUIDs=${studyUID}`
  const weasisUrl = `${WEASIS_BASE}/viewer?studyUID=${studyUID}`
  const activeUrl = viewer === 'ohif' ? ohifUrl : weasisUrl

  return (
    <div className="viewer-panel">
      <div className="viewer-panel-header">
        <div className="viewer-toggle">
          <button
            type="button"
            className={`viewer-toggle-btn${viewer === 'ohif' ? ' active' : ''}`}
            onClick={() => setViewer('ohif')}
          >
            OHIF
          </button>
          <button
            type="button"
            className={`viewer-toggle-btn${viewer === 'weasis' ? ' active' : ''}`}
            onClick={() => setViewer('weasis')}
          >
            DWV
          </button>
        </div>
        <div className="viewer-panel-actions">
          <a
            href={activeUrl}
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
        key={activeUrl}
        src={activeUrl}
        className="viewer-iframe"
        title="DICOM Viewer"
        allow="fullscreen"
      />
    </div>
  )
}
