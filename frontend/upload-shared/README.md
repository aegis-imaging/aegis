# @aegis/upload-shared

Shared React UI for the AEGIS upload flow. Both the web `upload-portal` and
the Tauri `desktop` app import their pieces from here so the upload
experience is identical across surfaces — single source of truth.

## Components

### Leaf components

| Component | Purpose |
|---|---|
| `FileDropZone` | Drag-and-drop area for DICOM files and folders. Recursive `webkitGetAsEntry` traversal so a parent folder of N studies just works. |
| `StudySummary` | Per-study metadata header (patient, modality, body part, date, series + image counts) with timezone-aware date rendering. |
| `TagDiffTable` | Before/after diff of the PS3.15 Annex E Basic Profile transformations applied to a study's tags. |
| `HeavyModeToggle` | Opt-in panel for in-browser pixel PHI scrub + face de-id. Reads `studyLooksLikeHead` from the host to enable / disable the defacing option. |
| `PixelScrubProgress` | Per-file progress + findings summary while heavy-mode is running. |
| `BulkStudyTable` | Live table of detected studies during bulk upload; rows update as the per-study state transitions. |

### Composable

| Component | Purpose |
|---|---|
| `UploadFlow` | **The whole upload pipeline as one drop-in component.** Stage machine (select → parsing → preview → uploading → ready), file picker, tag-diff preview, heavy-mode toggle, bulk upload, progress, "start over." Hosts pass auth/project/attribution as props; the flow handles the rest. |

### Why `UploadFlow` exists

Three current and future hosts need exactly the same upload pipeline:

1. **upload-portal** (web) — uses the leaf components and runs its own stage machine in `App.tsx` (~700 lines)
2. **desktop** (Tauri) — uses `UploadFlow` directly (this PR)
3. *(future)* embedded uploader inside the admin-dashboard for "upload here" UX from inside a project

Pulling the stage machine into a single component eliminates the divergence
risk and means a new host can stand up the full flow with one line of JSX.

```tsx
import { UploadFlow } from '@aegis/upload-shared'

<UploadFlow
  projectSlug="default"
  initialUploaderEmail="me@example.com"
  pickFolderOverride={tauriFolderPicker /* optional, browser falls back to <input> */}
  onCompleted={results => refreshStudyList()}
/>
```

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

- Migrate upload-portal's `App.tsx` to use `<UploadFlow>` (currently still
  runs its own stage machine inline; works fine, but is duplicated logic).
  Deferred to its own PR so the diff stays small and the production web
  flow keeps its tested behavior until we're ready to swap.
- A pluggable "viewer" slot in `BulkStudyTable` so hosts can show a tiny
  preview thumbnail per study.
- Optional `<UploadFlow.PreviewSlot>` / `<UploadFlow.HeavyModeSlot>` etc.
  composition pattern for hosts that want to swap in custom panels.

## Tests

Smoke-only — these are leaf components with no business logic. The
behavior-level tests (parsing, de-id, scrub) live in `@aegis/client`.
