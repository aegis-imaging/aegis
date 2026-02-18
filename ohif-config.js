window.config = {
  routerBasename: '/',
  dataSources: [
    {
      namespace: '@ohif/extension-default.dataSourcesModule.dicomweb',
      sourceName: 'dicomweb',
      configuration: {
        name: 'AEGIS',
        wadoUriRoot: 'http://localhost:8080/dicomweb/wado',
        qidoRoot: 'http://localhost:8080/dicomweb',
        wadoRoot: 'http://localhost:8080/dicomweb',
        qidoSupportsIncludeField: false,
        supportsReject: false,
        imageRendering: 'wadors',
        thumbnailRendering: 'wadors',
      },
    },
    // Second data source pointing at the raw (pre-defacing) store.
    // Used by the admin defacing review panel for before/after comparison.
    // Select it via: /viewer?StudyInstanceUIDs={uid}&dataSource=dicomweb-raw
    {
      namespace: '@ohif/extension-default.dataSourcesModule.dicomweb',
      sourceName: 'dicomweb-raw',
      configuration: {
        name: 'AEGIS (pre-defacing)',
        wadoUriRoot: 'http://localhost:8080/dicomweb-raw/wado',
        qidoRoot: 'http://localhost:8080/dicomweb-raw',
        wadoRoot: 'http://localhost:8080/dicomweb-raw',
        qidoSupportsIncludeField: false,
        supportsReject: false,
        imageRendering: 'wadors',
        thumbnailRendering: 'wadors',
      },
    },
  ],
  defaultDataSourceName: 'dicomweb',
};
