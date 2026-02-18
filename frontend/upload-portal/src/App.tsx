import { useState, useCallback } from 'react'
import './App.css'
import { FileDropZone } from './components/FileDropZone'
import { StudySummary } from './components/StudySummary'
import { TagDiffTable } from './components/TagDiffTable'
import { parseDicomFile, buildStudySummary, isDicomFile } from '@aegis/client'
import { deidentify } from '@aegis/client'
import { uploadStudy } from '@aegis/client'
import type { ParsedDicomFile, StudySummary as StudySummaryType, DicomTag, UploadResult } from '@aegis/client'

type Stage = 'select' | 'parsing' | 'preview' | 'uploading' | 'ready'

export function App() {
  const [stage, setStage] = useState<Stage>('select')
  const [files, setFiles] = useState<ParsedDicomFile[]>([])
  const [summary, setSummary] = useState<StudySummaryType | null>(null)
  const [tagChanges, setTagChanges] = useState<DicomTag[]>([])
  const [privateTagsRemoved, setPrivateTagsRemoved] = useState(0)
  const [parseProgress, setParseProgress] = useState({ current: 0, total: 0 })
  const [uploadProgress, setUploadProgress] = useState({ current: 0, total: 0 })
  const [uploadResult, setUploadResult] = useState<UploadResult | null>(null)
  const [uploaderEmail, setUploaderEmail] = useState('')
  const [currentFile, setCurrentFile] = useState('')
  const [error, setError] = useState<string | null>(null)

  const handleFilesSelected = useCallback(async (selectedFiles: File[]) => {
    setError(null)
    setStage('parsing')

    try {
      // Filter to DICOM files only
      const dicomChecks = await Promise.all(
        selectedFiles.map(async f => ({ file: f, isDicom: await isDicomFile(f) }))
      )
      const dicomFiles = dicomChecks.filter(c => c.isDicom).map(c => c.file)

      if (dicomFiles.length === 0) {
        setError('No valid DICOM files found. Files must be DICOM Part 10 format.')
        setStage('select')
        return
      }

      setParseProgress({ current: 0, total: dicomFiles.length })

      // Parse files sequentially to avoid memory pressure
      const parsed: ParsedDicomFile[] = []
      for (let i = 0; i < dicomFiles.length; i++) {
        const arrayBuffer = await dicomFiles[i].arrayBuffer()
        try {
          const { parsed: p } = parseDicomFile(arrayBuffer, dicomFiles[i].name)
          parsed.push(p)
        } catch (parseErr) {
          console.warn(`Skipping ${dicomFiles[i].name}: ${parseErr}`)
        }
        setParseProgress({ current: i + 1, total: dicomFiles.length })
      }

      if (parsed.length === 0) {
        setError('Could not parse any of the selected files.')
        setStage('select')
        return
      }

      setFiles(parsed)
      setSummary(buildStudySummary(parsed))

      // Run de-identification preview on the first file to show tag diff
      const firstBuffer = parsed[0].arrayBuffer
      const { dataset } = parseDicomFile(firstBuffer, parsed[0].filename)
      const deidResult = await deidentify(dataset, { salt: 'aegis-preview' })
      setTagChanges(deidResult.tagChanges)
      setPrivateTagsRemoved(deidResult.privateTagsRemoved)

      setStage('preview')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to parse DICOM files')
      setStage('select')
    }
  }, [])

  const handleReset = useCallback(() => {
    setStage('select')
    setFiles([])
    setSummary(null)
    setTagChanges([])
    setPrivateTagsRemoved(0)
    setUploadProgress({ current: 0, total: 0 })
    setUploadResult(null)
    setUploaderEmail('')
    setError(null)
  }, [])

  const handleUpload = useCallback(async () => {
    if (!summary || files.length === 0) {
      setError('No files available for upload.')
      return
    }

    setError(null)
    setStage('uploading')
    setUploadProgress({ current: 0, total: files.length })

    try {
      // Fetch the project's active anonymization profile (if any) so that any
      // project-specific tag retention overrides are applied during de-id.
      let retainedTags: string[] | undefined
      try {
        const profileRes = await fetch('/api/projects/default/active-anon-profile')
        if (profileRes.ok) {
          const profile = await profileRes.json() as { retained_tags: string[] }
          if (Array.isArray(profile.retained_tags) && profile.retained_tags.length > 0) {
            retainedTags = profile.retained_tags
          }
        }
        // 204 = no default profile configured — proceed with full Basic Profile strip
      } catch {
        // Non-fatal: if profile fetch fails, fall back to full strip
      }

      const result = await uploadStudy(files, 'default', summary, {
        onProgress: (uploaded, total) => setUploadProgress({ current: uploaded, total }),
        onFileStart: (filename) => setCurrentFile(filename),
        uploaderEmail: uploaderEmail.trim() || undefined,
        deid: retainedTags ? { retainedTags } : undefined,
      })
      setUploadResult(result)
      setStage('ready')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed')
      setStage('preview')
    }
  }, [files, summary])

  return (
    <div style={{
      maxWidth: '1000px',
      margin: '0 auto',
      padding: '32px 24px',
      fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
    }}>
      {/* Header */}
      <header style={{ marginBottom: '32px' }}>
        <h1 style={{ margin: '0 0 4px', fontSize: '28px', fontWeight: 700 }}>
          AEGIS Upload Portal
        </h1>
        <p style={{ margin: 0, color: '#6b7280', fontSize: '15px' }}>
          Anonymization & Exchange Gateway for Imaging Studies
        </p>
      </header>

      {/* Error display */}
      {error && (
        <div style={{
          padding: '12px 16px',
          backgroundColor: '#fef2f2',
          border: '1px solid #fecaca',
          borderRadius: '8px',
          color: '#dc2626',
          marginBottom: '24px',
          fontSize: '14px',
        }}>
          {error}
        </div>
      )}

      {/* Step 1: File selection */}
      {stage === 'select' && (
        <FileDropZone onFilesSelected={handleFilesSelected} />
      )}

      {/* Parsing progress */}
      {stage === 'parsing' && (
        <div style={{ textAlign: 'center', padding: '48px 24px' }}>
          <p style={{ fontSize: '16px', marginBottom: '16px' }}>
            Parsing DICOM files... {parseProgress.current} / {parseProgress.total}
          </p>
          <div style={{
            height: '8px',
            backgroundColor: '#e5e7eb',
            borderRadius: '4px',
            overflow: 'hidden',
            maxWidth: '400px',
            margin: '0 auto',
          }}>
            <div style={{
              height: '100%',
              width: `${parseProgress.total ? (parseProgress.current / parseProgress.total) * 100 : 0}%`,
              backgroundColor: '#2563eb',
              borderRadius: '4px',
              transition: 'width 0.2s ease',
            }} />
          </div>
        </div>
      )}

      {/* Step 2: Preview */}
      {stage === 'preview' && summary && (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
          <StudySummary summary={summary} />

          <TagDiffTable tags={tagChanges} privateTagsRemoved={privateTagsRemoved} />

          {/* Optional email for upload confirmation */}
          <div style={{
            padding: '16px',
            backgroundColor: '#f9fafb',
            border: '1px solid #e5e7eb',
            borderRadius: '8px',
          }}>
            <label style={{ display: 'block', fontSize: '14px', fontWeight: 500, marginBottom: '6px' }}>
              Your email <span style={{ color: '#6b7280', fontWeight: 400 }}>(optional)</span>
            </label>
            <input
              type="email"
              value={uploaderEmail}
              onChange={e => setUploaderEmail(e.target.value)}
              placeholder="you@institution.edu"
              style={{
                width: '100%',
                padding: '8px 12px',
                border: '1px solid #d1d5db',
                borderRadius: '6px',
                fontSize: '14px',
                boxSizing: 'border-box',
              }}
            />
            <p style={{ margin: '6px 0 0', fontSize: '12px', color: '#6b7280' }}>
              Receive a notification when your study has been reviewed.
            </p>
          </div>

          {/* Action buttons */}
          <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
            <button
              onClick={handleReset}
              style={{
                padding: '10px 24px',
                borderRadius: '8px',
                border: '1px solid #d1d5db',
                backgroundColor: '#fff',
                cursor: 'pointer',
                fontSize: '14px',
                fontWeight: 500,
              }}
            >
              Start over
            </button>
            <button
              onClick={handleUpload}
              style={{
                padding: '10px 24px',
                borderRadius: '8px',
                border: 'none',
                backgroundColor: '#2563eb',
                color: '#fff',
                cursor: 'pointer',
                fontSize: '14px',
                fontWeight: 600,
              }}
            >
              Confirm anonymization & upload ({files.length} files)
            </button>
          </div>
        </div>
      )}

      {stage === 'uploading' && (
        <div style={{ textAlign: 'center', padding: '48px 24px' }}>
          <p style={{ fontSize: '16px', marginBottom: '16px' }}>
            Anonymizing & uploading... {uploadProgress.current} / {uploadProgress.total}
          </p>
          <div style={{
            height: '8px',
            backgroundColor: '#e5e7eb',
            borderRadius: '4px',
            overflow: 'hidden',
            maxWidth: '400px',
            margin: '0 auto',
          }}>
            <div style={{
              height: '100%',
              width: `${uploadProgress.total ? (uploadProgress.current / uploadProgress.total) * 100 : 0}%`,
              backgroundColor: '#2563eb',
              borderRadius: '4px',
              transition: 'width 0.2s ease',
            }} />
          </div>
          {currentFile && (
            <p className="upload-current-file">
              {currentFile.length > 48 ? '…' + currentFile.slice(-46) : currentFile}
            </p>
          )}
        </div>
      )}

      {/* Step 3: Complete */}
      {stage === 'ready' && (
        <div style={{
          textAlign: 'center',
          padding: '48px 24px',
          border: '1px solid #bbf7d0',
          borderRadius: '12px',
          backgroundColor: '#f0fdf4',
        }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>{'\u2705'}</div>
          <h2 style={{ margin: '0 0 8px', fontSize: '20px' }}>
            Upload complete
          </h2>
          <p style={{ color: '#6b7280', marginBottom: '24px' }}>
            {files.length} files anonymized and uploaded successfully.
          </p>
          {uploadResult && (
            <p style={{ color: '#4b5563', marginBottom: '24px', fontSize: '14px' }}>
              Session: {uploadResult.sessionId}
              {uploadResult.study?.studyInstanceUid ? ` · Study UID: ${uploadResult.study.studyInstanceUid}` : ''}
            </p>
          )}
          <button
            onClick={handleReset}
            style={{
              padding: '10px 24px',
              borderRadius: '8px',
              border: '1px solid #d1d5db',
              backgroundColor: '#fff',
              cursor: 'pointer',
              fontSize: '14px',
            }}
          >
            Upload more files
          </button>
        </div>
      )}
    </div>
  )
}
