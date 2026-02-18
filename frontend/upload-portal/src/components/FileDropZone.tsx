import { useCallback, useRef, useState } from 'react'

interface FileDropZoneProps {
  onFilesSelected: (files: File[]) => void
  disabled?: boolean
}

export function FileDropZone({ onFilesSelected, disabled }: FileDropZoneProps) {
  const [isDragging, setIsDragging] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (!disabled) setIsDragging(true)
  }, [disabled])

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(false)
  }, [])

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(false)
    if (disabled) return

    const items = e.dataTransfer.items
    if (items) {
      collectFilesFromDataTransfer(items).then(onFilesSelected)
    }
  }, [disabled, onFilesSelected])

  const handleInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const fileList = e.target.files
    if (fileList) {
      onFilesSelected(Array.from(fileList))
    }
  }, [onFilesSelected])

  return (
    <div
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      onClick={() => !disabled && inputRef.current?.click()}
      style={{
        border: `2px dashed ${isDragging ? '#2563eb' : '#6b7280'}`,
        borderRadius: '12px',
        padding: '48px 24px',
        textAlign: 'center',
        cursor: disabled ? 'not-allowed' : 'pointer',
        backgroundColor: isDragging ? '#eff6ff' : disabled ? '#f3f4f6' : '#fafafa',
        transition: 'all 0.2s ease',
        opacity: disabled ? 0.6 : 1,
      }}
    >
      <div style={{ fontSize: '48px', marginBottom: '16px' }}>
        {isDragging ? '\u{1F4E5}' : '\u{1F4C1}'}
      </div>
      <p style={{ fontSize: '18px', fontWeight: 600, margin: '0 0 8px' }}>
        {isDragging ? 'Drop DICOM files here' : 'Drag & drop DICOM files or folders'}
      </p>
      <p style={{ fontSize: '14px', color: '#6b7280', margin: 0 }}>
        or click to browse. Supports .dcm files and DICOM directories.
      </p>

      {/* Hidden file input with webkitdirectory for folder selection */}
      <input
        ref={inputRef}
        type="file"
        multiple
        // @ts-expect-error webkitdirectory is not in standard HTML types
        webkitdirectory=""
        onChange={handleInputChange}
        style={{ display: 'none' }}
      />
    </div>
  )
}

/** Recursively collect files from drag-and-drop DataTransferItemList */
async function collectFilesFromDataTransfer(items: DataTransferItemList): Promise<File[]> {
  const files: File[] = []

  const entries: FileSystemEntry[] = []
  for (let i = 0; i < items.length; i++) {
    const entry = items[i].webkitGetAsEntry()
    if (entry) entries.push(entry)
  }

  async function traverseEntry(entry: FileSystemEntry): Promise<void> {
    if (entry.isFile) {
      const file = await new Promise<File>((resolve, reject) => {
        (entry as FileSystemFileEntry).file(resolve, reject)
      })
      files.push(file)
    } else if (entry.isDirectory) {
      const reader = (entry as FileSystemDirectoryEntry).createReader()
      const subEntries = await new Promise<FileSystemEntry[]>((resolve, reject) => {
        reader.readEntries(resolve, reject)
      })
      for (const subEntry of subEntries) {
        await traverseEntry(subEntry)
      }
    }
  }

  for (const entry of entries) {
    await traverseEntry(entry)
  }

  return files
}
