import { useCallback, useEffect, useState } from 'react'
import { invoke } from '@tauri-apps/api/core'
import { FirstRunPairing } from './components/FirstRunPairing'
import { ServerSettings } from './components/ServerSettings'
import { ProjectSelector } from './components/ProjectSelector'
import { FolderPicker } from './components/FolderPicker'
import { AnonymizationPreview } from './components/AnonymizationPreview'
import { UploadProgress } from './components/UploadProgress'
import { AegisApi, type Project } from './lib/aegis-api'
import {
  deidentifyToBytes,
  parseBytes,
  summarizeStudies,
  type ParsedStudy,
  type PreviewFile,
} from './lib/anon'

interface StoredCredential {
  server_url: string
  api_key: string
}

interface FileEntry {
  path: string
  size_bytes: number
}

type Stage =
  | 'loading'
  | 'pairing'
  | 'settings'
  | 'ready'
  | 'parsing'
  | 'preview'
  | 'uploading'
  | 'done'

interface UploadState {
  uploaded: number
  total: number
  currentFile: string
  finished: boolean
  error: string | null
}

export function App() {
  const [stage, setStage] = useState<Stage>('loading')
  const [credential, setCredential] = useState<StoredCredential | null>(null)
  const [api, setApi] = useState<AegisApi | null>(null)
  const [projects, setProjects] = useState<Project[]>([])
  const [selectedProject, setSelectedProject] = useState<Project | null>(null)
  const [pickedFolder, setPickedFolder] = useState<string | null>(null)
  const [files, setFiles] = useState<PreviewFile[]>([])
  const [studies, setStudies] = useState<ParsedStudy[]>([])
  const [uploaderEmail, setUploaderEmail] = useState<string>('')
  const [upload, setUpload] = useState<UploadState>({
    uploaded: 0,
    total: 0,
    currentFile: '',
    finished: false,
    error: null,
  })
  const [error, setError] = useState<string | null>(null)

  // First-run: check stored credential, then check pairing token, otherwise
  // fall back to manual server-URL+API-key entry.
  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const stored = await invoke<StoredCredential | null>('get_credential')
        if (cancelled) return
        if (stored && stored.api_key) {
          setCredential(stored)
          setStage('ready')
          return
        }
        const token = await invoke<string | null>('get_pairing_token')
        if (cancelled) return
        if (token) {
          setStage('pairing')
        } else {
          setStage('settings')
        }
      } catch (e) {
        if (cancelled) return
        setError(String(e))
        setStage('settings')
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  // Once credentialed, instantiate the API client and load projects.
  useEffect(() => {
    if (!credential) return
    const client = new AegisApi(credential.server_url, credential.api_key)
    setApi(client)
    ;(async () => {
      try {
        const list = await client.listProjects()
        setProjects(list)
      } catch (e) {
        setError(`Loading projects failed: ${e instanceof Error ? e.message : String(e)}`)
      }
    })()
  }, [credential])

  const handlePaired = useCallback((cred: StoredCredential) => {
    setCredential(cred)
    setStage('ready')
  }, [])

  const handleManualSettings = useCallback(
    async (serverUrl: string, apiKey: string) => {
      try {
        await invoke('store_credential', { serverUrl, apiKey })
        setCredential({ server_url: serverUrl, api_key: apiKey })
        setStage('ready')
      } catch (e) {
        setError(String(e))
      }
    },
    [],
  )

  const handleResetCredential = useCallback(async () => {
    await invoke('clear_credential')
    setCredential(null)
    setApi(null)
    setProjects([])
    setSelectedProject(null)
    setPickedFolder(null)
    setFiles([])
    setStudies([])
    setStage('settings')
  }, [])

  const handleFolderPicked = useCallback(async (path: string) => {
    setPickedFolder(path)
    setFiles([])
    setStudies([])
    setError(null)
    setStage('parsing')
    try {
      const entries = await invoke<FileEntry[]>('enumerate_folder', { path })
      const mapped: PreviewFile[] = entries.map(e => ({
        path: e.path,
        filename: e.path.split(/[\\/]/).pop() ?? e.path,
        size_bytes: e.size_bytes,
      }))
      setFiles(mapped)
      if (mapped.length === 0) {
        setError('No DICOM files found in the selected folder.')
        setStage('ready')
        return
      }

      // Parse all files for the preview. We hold every parsed dataset in memory
      // here for the per-study summary; for very large datasets the user can
      // pick a sub-folder. uploadStudies() below frees and re-parses one file
      // at a time so peak memory during upload stays bounded.
      const parsed = []
      for (const f of mapped) {
        const bytes = await invoke<number[]>('read_dicom_bytes', { path: f.path })
        const u8 = new Uint8Array(bytes)
        const pf = parseBytes(u8, f.filename)
        if (pf) parsed.push(pf)
      }
      setStudies(summarizeStudies(parsed))
      setStage('preview')
    } catch (e) {
      setError(`Folder scan failed: ${e instanceof Error ? e.message : String(e)}`)
      setStage('ready')
    }
  }, [])

  const handleUpload = useCallback(async () => {
    if (!api || !selectedProject || studies.length === 0) return
    setStage('uploading')
    setUpload({
      uploaded: 0,
      total: studies.reduce((acc, s) => acc + s.files.length, 0),
      currentFile: '',
      finished: false,
      error: null,
    })

    try {
      let cumulativeUploaded = 0
      for (const study of studies) {
        // 1. init session
        const init = await api.uploadInit({
          projectSlug: selectedProject.slug,
          fileCount: study.files.length,
          uploaderEmail,
          studyMetadata: {
            studyInstanceUid: study.summary.studyInstanceUid,
            modality: study.summary.modality,
            bodyPart: study.summary.bodyPart,
            studyDescription: study.summary.studyDescription,
            studyDate: study.summary.studyDate,
            seriesCount: study.summary.seriesCount,
            imageCount: study.summary.imageCount,
          },
        })
        if (init.upload_urls.length !== study.files.length) {
          throw new Error('upload init returned mismatched URL count')
        }

        // 2. for each file: anonymize + PUT, then release the buffer.
        for (let i = 0; i < study.files.length; i++) {
          const parsed = study.files[i]
          setUpload(prev => ({ ...prev, currentFile: parsed.filename }))
          const deidBytes = await deidentifyToBytes(parsed, selectedProject.slug)
          await api.uploadFile(init.upload_urls[i], deidBytes)
          cumulativeUploaded += 1
          setUpload(prev => ({ ...prev, uploaded: cumulativeUploaded }))
          // Hint to V8 — drop our reference before the loop reads the next file.
          // (parsed is still held by `studies`; only deidBytes is local.)
        }

        // 3. complete
        await api.uploadComplete(init.session_id)
      }
      setUpload(prev => ({ ...prev, finished: true }))
      setStage('done')
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      setUpload(prev => ({ ...prev, error: msg }))
    }
  }, [api, selectedProject, studies, uploaderEmail])

  const handleStartOver = useCallback(() => {
    setPickedFolder(null)
    setFiles([])
    setStudies([])
    setUpload({ uploaded: 0, total: 0, currentFile: '', finished: false, error: null })
    setStage('ready')
  }, [])

  if (stage === 'loading') {
    return <main className="container"><p>Loading…</p></main>
  }
  if (stage === 'pairing') {
    return (
      <main className="container">
        <FirstRunPairing
          onPaired={handlePaired}
          onCancel={() => setStage('settings')}
        />
      </main>
    )
  }
  if (stage === 'settings') {
    return (
      <main className="container">
        <ServerSettings onSubmit={handleManualSettings} initialError={error} />
      </main>
    )
  }

  return (
    <main className="container">
      <header style={{ display: 'flex', justifyContent: 'space-between', padding: '12px 16px', borderBottom: '1px solid #e5e7eb' }}>
        <h1 style={{ fontSize: 18, margin: 0 }}>AEGIS Uploader</h1>
        <div style={{ fontSize: 12, color: '#6b7280' }}>
          {credential?.server_url}{' '}
          <button onClick={handleResetCredential} style={{ marginLeft: 8 }}>Re-pair</button>
        </div>
      </header>

      <section style={{ padding: 16 }}>
        {error && (
          <div role="alert" style={{ background: '#ffedd5', color: '#9a3412', padding: 12, marginBottom: 12, borderRadius: 6 }}>
            {error}
          </div>
        )}

        <ProjectSelector
          projects={projects}
          selected={selectedProject}
          onSelect={setSelectedProject}
        />

        <div style={{ marginTop: 16 }}>
          <label style={{ display: 'block', fontSize: 12, color: '#374151', marginBottom: 4 }}>
            Uploader email (optional — receives the processing confirmation)
          </label>
          <input
            type="email"
            value={uploaderEmail}
            onChange={e => setUploaderEmail(e.target.value)}
            placeholder="you@hospital.org"
            style={{ width: 320, padding: 6 }}
          />
        </div>

        {selectedProject && (
          <div style={{ marginTop: 16 }}>
            <FolderPicker
              onPicked={handleFolderPicked}
              pickedPath={pickedFolder}
              disabled={stage === 'parsing' || stage === 'uploading'}
            />
          </div>
        )}

        {stage === 'parsing' && (
          <p style={{ marginTop: 16, color: '#6b7280' }}>
            Scanning folder and parsing DICOM headers ({files.length} files)…
          </p>
        )}

        {stage === 'preview' && studies.length > 0 && (
          <AnonymizationPreview
            studies={studies}
            onConfirm={handleUpload}
            onCancel={handleStartOver}
          />
        )}

        {(stage === 'uploading' || stage === 'done') && (
          <UploadProgress
            state={upload}
            onDone={handleStartOver}
          />
        )}
      </section>
    </main>
  )
}
