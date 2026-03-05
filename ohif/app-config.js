// AEGIS OHIF Viewer configuration
//
// Two data sources are defined:
//   aegis-clean  — reads from the de-identified (post-defacing) DICOM store.
//                  This is the default source for normal study review.
//   aegis-raw    — reads from the raw (pre-defacing) DICOM store.
//                  Select via URL param: ?dataSources=aegis-raw
//
// Both sources proxy through this container's nginx (/dicomweb/ and
// /dicomweb-raw/), which forwards to the AEGIS Go API backend.
//
// URL to open a specific study:
//   /viewer?StudyInstanceUIDs=1.2.3.4.5
//   /viewer?StudyInstanceUIDs=1.2.3.4.5&dataSources=aegis-raw
//
window.config = {
  routerBasename: '/',
  extensions: [],
  modes: [],

  // AEGIS controls study navigation via the admin dashboard.
  // Disable the OHIF built-in worklist to keep the UX clean.
  showStudyList: false,

  showLoadingIndicator: true,

  // All studies in AEGIS are de-identified — no patient info to display.
  showPatientInfo: 'disabled',

  defaultDataSourceName: 'aegis-clean',

  dataSources: [
    // ── De-identified store (default) ────────────────────────────────────────
    {
      namespace: '@ohif/extension-default.dataSourcesModule.dicomweb',
      sourceName: 'aegis-clean',
      configuration: {
        friendlyName: 'AEGIS (De-identified)',
        name: 'aegis-clean',

        // All three roots point to /dicomweb/ — nginx proxies to the AEGIS API.
        wadoUriRoot: '/dicomweb',
        qidoRoot:    '/dicomweb',
        wadoRoot:    '/dicomweb',

        imageRendering:     'wadors',
        // AEGIS does not implement the /rendered thumbnail endpoint.
        // Setting thumbnailRendering to 'wadors' causes OHIF to fetch the first
        // WADO-RS frame as a thumbnail fallback instead.
        thumbnailRendering: 'wadors',

        enableStudyLazyLoad:    true,
        supportsFuzzyMatching:  false,
        supportsWildcard:       true,
        dicomUploadEnabled:     false,

        // Tell OHIF which payload types AEGIS serves as singlepart responses.
        singlepart: 'bulkdata,video,pdf',
      },
    },

    // ── Raw (pre-defacing) store ──────────────────────────────────────────────
    // Used for defacing review: select via ?dataSources=aegis-raw in the URL.
    // QIDO-RS queries still use /dicomweb/ (clean index) so study/series
    // metadata is consistent; only pixel retrieval (WADO-RS) uses /dicomweb-raw/.
    {
      namespace: '@ohif/extension-default.dataSourcesModule.dicomweb',
      sourceName: 'aegis-raw',
      configuration: {
        friendlyName: 'AEGIS (Raw / Pre-defacing)',
        name: 'aegis-raw',

        wadoUriRoot: '/dicomweb-raw',
        qidoRoot:    '/dicomweb',      // metadata index is always from clean store
        wadoRoot:    '/dicomweb-raw',

        imageRendering:     'wadors',
        thumbnailRendering: 'wadors',

        enableStudyLazyLoad:   true,
        supportsFuzzyMatching: false,
        supportsWildcard:      true,
        dicomUploadEnabled:    false,

        singlepart: 'bulkdata,video,pdf',
      },
    },
  ],
}
