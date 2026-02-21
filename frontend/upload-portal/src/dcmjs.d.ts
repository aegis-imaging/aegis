declare module 'dcmjs' {
  interface DicomDataset {
    meta: Record<string, unknown>
    dict: Record<string, unknown>
  }

  interface DicomMessageStatic {
    readFile(arrayBuffer: ArrayBuffer): DicomDataset
  }

  interface DicomMetaDictionaryStatic {
    naturalizeDataset(dict: Record<string, unknown>): Record<string, unknown>
    denaturalizeDataset(dataset: Record<string, unknown>): Record<string, unknown>
    namifyDataset(dict: Record<string, unknown>): Record<string, unknown>
  }

  interface DicomDictConstructor {
    new (meta: Record<string, unknown>): {
      dict: Record<string, unknown>
      write(): Uint8Array
    }
  }

  interface DataModule {
    DicomMessage: DicomMessageStatic
    DicomMetaDictionary: DicomMetaDictionaryStatic
    DicomDict: DicomDictConstructor
    datasetToBuffer(dataset: Record<string, unknown>): Uint8Array
    datasetToBlob(dataset: Record<string, unknown>): Blob
  }

  const dcmjs: {
    data: DataModule
  }

  export default dcmjs
}
