# @aegis/upload-shared

Shared React UI for the AEGIS upload flow. Both the web `upload-portal` and
the Tauri `desktop` app import their pieces from here so the upload
experience is identical across surfaces — single source of truth.

## Components

| Component | Purpose |
|---|---|
| `FileDropZone` | Drag-and-drop area for DICOM files and folders. Recursive `webkitGetAsEntry` traversal so a parent folder of N studies just works. |
| `StudySummary` | Per-study metadata header (patient, modality, body part, date, series + image counts) with timezone-aware date rendering. |
| `TagDiffTable` | Before/after diff of the PS3.15 Annex E Basic Profile transformations applied to a study's tags. |
| `HeavyModeToggle` | Opt-in panel for in-browser pixel PHI scrub + face de-id. Reads `studyLooksLikeHead` from the host to enable / disable the defacing option. |
| `PixelScrubProgress` | Per-file progress + findings summary while heavy-mode is running. |
| `BulkStudyTable` | Live table of detected studies during bulk upload; rows update as the per-study state transitions. |

## Design rules

- **No host-specific state** — auth, project lists, timezone settings, invite
  gates etc. live in the host (upload-portal `App.tsx` or desktop `App.tsx`).
  These components take their inputs via props.
- **No host-specific routing or stage machine** — the components are leaf
  views, not full pages. Stage management (`select → preview → upload`)
  belongs to the host.
- **Re-exports of `@aegis/client` types where useful** — props that wrap
  values from the client lib don't redefine them here.
- **Bundled CSS-in-JS only** — no separate stylesheet so consumers don't
  need to import a `.css` file.

## Migrating a host to use it

Replace:

```tsx
import { FileDropZone } from './components/FileDropZone'
import { StudySummary } from './components/StudySummary'
// ...
```

with:

```tsx
import {
  FileDropZone,
  StudySummary,
  TagDiffTable,
  HeavyModeToggle,
  PixelScrubProgress,
  BulkStudyTable,
  type HeavyModeSettings,
} from '@aegis/upload-shared'
```

Both hosts (upload-portal and desktop) already do this in this PR. New
hosts (e.g. an embedded uploader in the admin dashboard) follow the same
pattern.

## Roadmap

- Factor out `<UploadFlow>` — the full stage machine (select → parse →
  preview → upload → done) so a new host can drop in the whole flow in
  one component. The current state lives in upload-portal's `App.tsx`.
- Move `studyLooksLikeHeadScan()` heuristic from `@aegis/client` here?
  (No — it's used by the upload pipeline directly, belongs in client.)
- A pluggable "viewer" slot in `BulkStudyTable` so hosts can show a tiny
  preview thumbnail per study.

## Tests

Smoke-only — these are leaf components with no business logic. The
behavior-level tests (parsing, de-id, scrub) live in `@aegis/client`.
