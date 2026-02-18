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
  ],
  defaultDataSourceName: 'dicomweb',
};
