import { App, AppOptions, ViewConfig } from 'dwv'

const params = new URLSearchParams(window.location.search)
const studyUID = params.get('studyUID')
const store = params.get('store') || 'clean'   // 'raw' = pre-defacing store, 'clean' = defaced store
const apiBase = store === 'raw' ? '/dicomweb-raw' : '/dicomweb'

const statusEl = document.getElementById('status')

function setStatus(msg, cls = 'loading') {
  if (!statusEl) return
  statusEl.textContent = msg
  statusEl.className = cls
}

function showError(msg) {
  console.error('[DWV viewer]', msg)
  const errEl = document.getElementById('error-message')
  if (errEl) {
    errEl.style.display = 'block'
    errEl.textContent = msg
  }
  const container = document.getElementById('dwv-container')
  if (container) container.style.display = 'none'
  setStatus('Error', 'error')
}

// ── Toolbar tool-switching ────────────────────────────────────────────────────

let app = null
let defaultToolSet = false  // guard: only initialise the default tool once

function syncToolbar(activeTool) {
  document.querySelectorAll('.tool-btn').forEach((btn) => {
    btn.classList.toggle('active', btn.dataset.tool === activeTool)
  })
}

document.querySelectorAll('.tool-btn').forEach((btn) => {
  btn.addEventListener('click', () => {
    if (app) {
      try {
        app.setTool(btn.dataset.tool)
      } catch (err) {
        console.error('[DWV] setTool failed:', btn.dataset.tool, err)
      }
      // Always update visual state regardless of whether setTool succeeded.
      syncToolbar(btn.dataset.tool)
    }
  })
})

// ── Main ──────────────────────────────────────────────────────────────────────

if (!studyUID) {
  showError('No studyUID provided. Append ?studyUID=<DICOM StudyInstanceUID> to the URL.')
} else {
  // Initialise DWV App
  const viewConfig = new ViewConfig('dwv-container')
  const options = new AppOptions({ '*': [viewConfig] })
  options.tools = {
    Scroll: {},
    ZoomAndPan: {},
    WindowLevel: {},
  }

  app = new App()
  app.init(options)

  // 'load' fires once per loaded data item, AFTER DWV has set up the layer
  // group's active layer — the correct place to call setTool(). Using 'loadend'
  // instead causes getActiveLayer() to return undefined so bindLayerGroup() is
  // silently skipped and no canvas events are ever bound (DWV source:
  // `void 0 !== n && this.#Fl.bindLayerGroup(t, n)`).
  app.addEventListener('load', () => {
    if (!defaultToolSet) {
      defaultToolSet = true
      try {
        app.setTool('Scroll')
      } catch (err) {
        console.error('[DWV] initial setTool failed:', err)
      }
      syncToolbar('Scroll')
    }
  })

  app.addEventListener('loadprogress', (e) => {
    if (e.loaded < e.total) {
      setStatus(`Loading ${e.loaded} / ${e.total}…`)
    }
  })

  app.addEventListener('loadend', () => {
    setStatus('Ready', 'ready')
  })

  app.addEventListener('loaderror', (e) => {
    setStatus('Load error', 'error')
    console.error('[DWV loaderror]', e.error)
  })

  // 1. Fetch series list via QIDO-RS
  setStatus('Fetching series…')
  fetch(`${apiBase}/studies/${studyUID}/series`)
    .then((r) => {
      if (!r.ok) throw new Error(`Series request failed: ${r.status} ${r.statusText}`)
      return r.json()
    })
    .then((series) => {
      if (!series.length) throw new Error('No series found for this study.')
      const seriesUID = series[0]['0020000E']?.Value?.[0]
      if (!seriesUID) throw new Error('Series response is missing SeriesInstanceUID (0020000E).')

      setStatus('Fetching instances…')
      return fetch(`${apiBase}/studies/${studyUID}/series/${seriesUID}/instances`)
    })
    .then((r) => {
      if (!r.ok) throw new Error(`Instances request failed: ${r.status} ${r.statusText}`)
      return r.json()
    })
    .then((instances) => {
      if (!instances.length) throw new Error('No instances found in this series.')

      const seriesUID = instances[0]['0020000E']?.Value?.[0]
      const urls = instances.map((inst) => {
        const sopUID = inst['00080018']?.Value?.[0]
        return `${apiBase}/studies/${studyUID}/series/${seriesUID}/instances/${sopUID}`
      })

      setStatus(`Loading ${urls.length} image${urls.length !== 1 ? 's' : ''}…`)
      app.loadURLs(urls)
    })
    .catch((err) => showError(err.message))
}
