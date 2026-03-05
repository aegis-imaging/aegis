const DWV_BASE = import.meta.env.VITE_DWV_BASE_URL || 'http://localhost:3005'

export function ViewerPanel({ studyUID, onClose }: { studyUID: string; onClose: () => void }) {
  const url = `${DWV_BASE}/viewer?studyUID=${studyUID}`

  return (
    <div className="viewer-panel">
      <div className="viewer-panel-header">
        <span className="viewer-panel-title">AEGIS DICOM Viewer</span>
        <div className="viewer-panel-actions">
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
        src={url}
        className="viewer-iframe"
        title="DICOM Viewer"
        allow="fullscreen"
      />
    </div>
  )
}
