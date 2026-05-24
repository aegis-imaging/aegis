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
let loadedCount = 0         // images that rendered successfully; checked in loadend

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

// ── Slice navigation helpers ──────────────────────────────────────────────────

function getVC() {
  const lg = app?.getActiveLayerGroup()
  const vl = lg?.getActiveViewLayer()
  return vl?.getViewController() ?? null
}

function updateSliceLabel() {
  const vc = getVC()
  if (!vc) return
  const k = vc.getCurrentIndexScrollValue()
  const total = typeof vc.getNumberOfSlices === 'function' ? vc.getNumberOfSlices() : null
  const label = total != null ? `${k + 1} / ${total}` : `${k + 1}`
  const labelEl = document.getElementById('slice-label')
  const prevBtn = document.getElementById('slice-prev')
  const nextBtn = document.getElementById('slice-next')
  if (labelEl) labelEl.textContent = label
  if (prevBtn) prevBtn.disabled = k <= 0
  if (nextBtn) nextBtn.disabled = total != null && k >= total - 1
}

document.getElementById('slice-prev')?.addEventListener('click', () => {
  const vc = getVC()
  if (!vc) return
  vc.getPositionHelper?.()?.decrementPositionAlongScroll()
})

document.getElementById('slice-next')?.addEventListener('click', () => {
  const vc = getVC()
  if (!vc) return
  vc.getPositionHelper?.()?.incrementPositionAlongScroll()
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
    loadedCount++
    if (!defaultToolSet) {
      defaultToolSet = true
      try {
        app.setTool('Scroll')
      } catch (err) {
        console.error('[DWV] initial setTool failed:', err)
      }
      syncToolbar('Scroll')
    }
    updateSliceLabel()
  })

  app.addEventListener('loadprogress', (e) => {
    if (e.loaded < e.total) {
      setStatus(`Loading ${e.loaded} / ${e.total}…`)
    }
  })

  app.addEventListener('loadend', () => {
    if (loadedCount === 0) {
      // All instance fetches failed — loaderror fired for each but loadend always
      // runs last, which previously overwrote the error state with 'Ready'.
      setStatus('Load failed — no images rendered', 'error')
      const errEl = document.getElementById('error-message')
      if (errEl) {
        errEl.style.display = 'block'
        errEl.textContent = 'Could not load DICOM images. The files may be unavailable. Check the browser console for details.'
      }
      const container = document.getElementById('dwv-container')
      if (container) container.style.display = 'none'
    } else {
      setStatus('Ready', 'ready')
      // Dispatch resize so DWV recomputes canvas dimensions — guards against a
      // race where the container wasn't fully painted during the initial render.
      window.dispatchEvent(new Event('resize'))
    }
  })

  // loaderror fires per-file; don't set status here — loadend runs last and
  // uses loadedCount to decide the final state.
  app.addEventListener('loaderror', (e) => {
    console.error('[DWV loaderror]', e.error)
  })

  // ── Yoke scrolling — bidirectional postMessage sync ──────────────────────────
  // When embedded in the defacing review panel, relay slice position to the parent
  // so it can synchronise the opposite (before/after) iframe.

  let yokeReceiving = false  // prevent echo when we receive a dwv-goto command

  app.addEventListener('positionchange', () => {
    updateSliceLabel()
    if (yokeReceiving) return
    if (window.parent === window) return   // not embedded in an iframe
    // Read k from the ViewController (reliable) instead of e.value[0] (may be a plain
    // array from getValues() in some DWV event paths, causing .get() to be undefined).
    const layerGroup = app.getActiveLayerGroup()
    if (!layerGroup) return
    const viewLayer = layerGroup.getActiveViewLayer()
    if (!viewLayer) return
    const vc = viewLayer.getViewController()
    if (!vc) return
    const k = vc.getCurrentIndexScrollValue()
    if (typeof k !== 'number') return
    window.parent.postMessage({ type: 'dwv-position', k }, '*')
  })

  window.addEventListener('message', (e) => {
    if (!app || !e.data || e.data.type !== 'dwv-goto') return
    const targetK = e.data.k
    if (typeof targetK !== 'number') return
    const layerGroup = app.getActiveLayerGroup()
    if (!layerGroup) return
    const viewLayer = layerGroup.getActiveViewLayer()
    if (!viewLayer) return
    const vc = viewLayer.getViewController()
    if (!vc) return
    // incrementPositionAlongScroll / decrementPositionAlongScroll live on PositionHelper,
    // not on ViewController directly — must go through vc.getPositionHelper().
    const posHelper = vc.getPositionHelper()
    if (!posHelper) return
    const currentK = vc.getCurrentIndexScrollValue()
    const delta = targetK - currentK
    if (delta === 0) return
    yokeReceiving = true
    const step = delta > 0
      ? () => posHelper.incrementPositionAlongScroll()
      : () => posHelper.decrementPositionAlongScroll()
    for (let i = 0; i < Math.abs(delta); i++) step()
    // Reset after a brief delay — positionchange may fire synchronously or via
    // microtask, so 100 ms is enough to swallow the echo without noticeable lag.
    setTimeout(() => { yokeReceiving = false }, 100)
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
