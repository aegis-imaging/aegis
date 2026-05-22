// Smoke test: verify every public export is defined. These are leaf React
// components with no business logic — anything beyond "the module loads and
// the function exists" belongs in the host or @aegis/client.
//
// We don't pull in react-dom or jsdom; we just check the export surface.

import { describe, it, expect } from 'vitest'
import * as shared from '../index'

describe('@aegis/upload-shared public exports', () => {
  it('exports FileDropZone', () => expect(typeof shared.FileDropZone).toBe('function'))
  it('exports StudySummary', () => expect(typeof shared.StudySummary).toBe('function'))
  it('exports TagDiffTable', () => expect(typeof shared.TagDiffTable).toBe('function'))
  it('exports HeavyModeToggle', () => expect(typeof shared.HeavyModeToggle).toBe('function'))
  it('exports PixelScrubProgress', () => expect(typeof shared.PixelScrubProgress).toBe('function'))
  it('exports BulkStudyTable', () => expect(typeof shared.BulkStudyTable).toBe('function'))
})
