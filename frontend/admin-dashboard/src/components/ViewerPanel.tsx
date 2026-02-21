const OHIF_BASE = 'http://localhost:3002'

export function ViewerPanel({ studyUID, onClose }: { studyUID: string; onClose: () => void }) {
  const viewerUrl = `${OHIF_BASE}/viewer?StudyInstanceUIDs=${studyUID}`

  return (
    <div className="viewer-panel">
      <div className="viewer-panel-header">
        <span className="viewer-panel-title">OHIF Viewer</span>
        <div className="viewer-panel-actions">
          <a
            href={viewerUrl}
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
        src={viewerUrl}
        className="viewer-iframe"
        title="DICOM Viewer"
        allow="fullscreen"
      />
    </div>
  )
}
