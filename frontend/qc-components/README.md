# @aegis/qc-components

Shared React component package implementing the image-analyst QC workflow.
Used by both the cloud `admin-dashboard` and the `desktop` app so the
analyst sees the same UI in either place.

## Components

| Component | Purpose |
|---|---|
| `QCTriageQueue` | Sortable / filterable list of studies awaiting review. Color-coded severity, "assigned to me" filter, auto-refresh. |
| `QCReviewPane` | Single-study review surface: viewer slot + findings panel + start/complete-review actions. Hosts provide the DICOM viewer (DWV / OHIF / etc.) via `viewerSlot`. |
| `QCFindingsPanel` | Per-study findings list with inline "raise finding" form and resolve / reopen / delete actions. |
| `AnalystThroughputCard` | Per-analyst stats: studies completed, open assignments, findings raised/resolved, avg review minutes, studies/hour. |

## Backend it talks to

All four components hit the QC endpoints added by the API in the same PR:

```
GET    /api/qc/triage                    list triage queue (filters)
GET    /api/qc/throughput                analyst stats
POST   /api/studies/{id}/qc/assign       assign analyst (or self)
DELETE /api/studies/{id}/qc/assign       unassign
POST   /api/studies/{id}/qc/start        stamp qc_review_started_at
POST   /api/studies/{id}/qc/complete     stamp qc_review_ended_at
GET    /api/studies/{id}/qc-findings     list findings (?open=true for open)
POST   /api/studies/{id}/qc-findings     create finding
POST   /api/qc-findings/{id}/resolve     resolve with note
POST   /api/qc-findings/{id}/reopen      reopen
DELETE /api/qc-findings/{id}             delete
```

Auth is whatever the host passes via `apiOptions.authorization` — typically
the admin dashboard relies on session cookies (no header) and the desktop
app uses a `Bearer <api_key>` header.

## Usage

```tsx
import { QCTriageQueue, QCReviewPane, AnalystThroughputCard } from '@aegis/qc-components'

function QCTab() {
  const [selectedStudy, setSelectedStudy] = useState<QCTriageItem | null>(null)
  return (
    <div>
      <AnalystThroughputCard />
      <QCTriageQueue onSelectStudy={setSelectedStudy} />
      {selectedStudy && (
        <QCReviewPane
          study={selectedStudy}
          viewerSlot={<DWVViewer studyUid={selectedStudy.study_instance_uid} />}
          onCompleted={() => setSelectedStudy(null)}
        />
      )}
    </div>
  )
}
```

For the desktop app, the only difference is `apiOptions`:

```tsx
const apiOptions = {
  baseUrl: 'https://api.aegisimaging.ai',
  authorization: `Bearer ${apiKey}`,
}
// ...components take apiOptions={apiOptions}
```
