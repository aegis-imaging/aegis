// OHIF runtime configuration for AEGIS.
// Uses window.location.origin so this static file works in every environment
// (docker-compose, Cloud Run, local dev) without env-var plumbing.
// OHIF's nginx proxies /dicomweb/ and /dicomweb-raw/ to the real API backend,
// so all DICOMweb requests are same-origin — no CORS required.
window.config = {
  routerBasename: '/',
  dataSources: [
    {
      namespace: '@ohif/extension-default.dataSourcesModule.dicomweb',
      sourceName: 'dicomweb',
      configuration: {
        name: 'AEGIS',
        wadoUriRoot: window.location.origin + '/dicomweb/wado',
        qidoRoot:    window.location.origin + '/dicomweb',
        wadoRoot:    window.location.origin + '/dicomweb',
        qidoSupportsIncludeField: false,
        supportsReject: false,
        imageRendering: 'wadors',
        thumbnailRendering: 'wadors',
      },
    },
    {
      // Raw (pre-defacing) store — select via ?dataSource=dicomweb-raw
      // Used by the admin defacing review panel for before/after comparison.
      namespace: '@ohif/extension-default.dataSourcesModule.dicomweb',
      sourceName: 'dicomweb-raw',
      configuration: {
        name: 'AEGIS (pre-defacing)',
        wadoUriRoot: window.location.origin + '/dicomweb-raw/wado',
        qidoRoot:    window.location.origin + '/dicomweb-raw',
        wadoRoot:    window.location.origin + '/dicomweb-raw',
        qidoSupportsIncludeField: false,
        supportsReject: false,
        imageRendering: 'wadors',
        thumbnailRendering: 'wadors',
      },
    },
  ],
  defaultDataSourceName: 'dicomweb',
};
