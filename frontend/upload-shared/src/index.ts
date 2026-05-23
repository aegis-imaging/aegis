// Public API of @aegis/upload-shared.
//
// Reusable React components for the AEGIS upload flow. Both the web
// upload-portal and the desktop app consume these so the UI stays
// consistent across surfaces and any bug fix lands in one place.

export { FileDropZone } from './components/FileDropZone'
export type { FileDropZoneProps } from './components/FileDropZone'

export { StudySummary } from './components/StudySummary'
export type { StudySummaryProps, DisplayTimezoneMode } from './components/StudySummary'

export { TagDiffTable } from './components/TagDiffTable'
export type { TagDiffTableProps } from './components/TagDiffTable'

export { HeavyModeToggle } from './components/HeavyModeToggle'
export type { HeavyModeToggleProps, HeavyModeSettings } from './components/HeavyModeToggle'

export { PixelScrubProgress } from './components/PixelScrubProgress'
export type { PixelScrubProgressProps, PixelScrubProgressState } from './components/PixelScrubProgress'

export { BulkStudyTable } from './components/BulkStudyTable'
export type { BulkStudyTableProps } from './components/BulkStudyTable'

// High-level composable: the whole upload flow as one component.
export { UploadFlow } from './UploadFlow'
export type { UploadFlowProps, UploadFlowStage } from './UploadFlow'
