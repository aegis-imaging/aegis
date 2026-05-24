import { open } from '@tauri-apps/plugin-dialog'

interface Props {
  onPicked: (path: string) => void
  pickedPath: string | null
  disabled: boolean
}

/**
 * Wraps Tauri's `dialog.open()` to pick a folder. The actual recursive walk
 * happens in Rust (enumerate_folder); we just need the absolute path here.
 */
export function FolderPicker({ onPicked, pickedPath, disabled }: Props) {
  const handlePick = async () => {
    const selected = await open({
      directory: true,
      multiple: false,
      title: 'Select a folder containing DICOM files',
    })
    if (typeof selected === 'string' && selected.length > 0) {
      onPicked(selected)
    }
  }

  return (
    <div>
      <button onClick={handlePick} disabled={disabled} style={{ padding: '8px 16px' }}>
        {pickedPath ? 'Pick a different folder' : 'Pick a folder'}
      </button>
      {pickedPath && (
        <span style={{ marginLeft: 12, color: '#374151', fontSize: 13, fontFamily: 'monospace' }}>
          {pickedPath}
        </span>
      )}
    </div>
  )
}
