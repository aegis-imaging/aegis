# Browser Upload Allowlist — design proposal

Status: design draft, not implemented. Tracks the second half of the
chunk-4 followup item ("browser upload allowlist sub-section" on the
per-institution detail panel) that was deferred from session 26261 as
"needs schema design".

## Context

The Institution detail panel (`/admin/institutions/:id`) currently owns
two deployment-method sub-sections:

- **Satellites** — on-prem routers enrolled via mTLS cert (existing).
- **Installer invitations** — desktop client install links scoped to
  the institution (added by the PR this design accompanies).

The third planned sub-section is the **browser upload allowlist**:
per-institution control of which browser-side upload flows are
permitted. Today every authenticated upload-portal user can submit
studies via any upload flow the server has built in. Some institutions
will need to disable specific paths (e.g. "no anonymized DIMSE-pull
imports from the browser; only researcher direct upload via per-study
share token").

## Goals

1. Per-institution allowlist of upload methods.
2. Server-enforced at the upload-init / upload-complete handlers, not
   just hidden in the UI.
3. Empty allowlist = back-compat default (all methods allowed) so
   existing tenants see no change.
4. Auditable: every allowlist change emits a `institution.upload_allowlist.*`
   audit entry.

## Non-goals

- Per-user (rather than per-institution) allowlisting — out of scope.
- Per-project upload throttling — separate feature.
- Anonymization-profile enforcement — already exists, distinct concern.

## Upload methods enumeration

Today the upload-portal speaks these distinct flows. The allowlist
toggles individual entries on/off per institution.

| Method ID | What | Server handler |
|---|---|---|
| `browser.web-upload` | Standard drag-and-drop upload via the public upload portal | `UploadInit`, `UploadFile`, `UploadComplete` |
| `browser.share-redeem` | Returning to upload via a share-token URL | `RedeemUploadShareToken` |
| `browser.dimse-pull` | "Pull from PACS" web wizard that triggers a DIMSE C-MOVE | `DimseImportInit` |
| `browser.tcia-import` | TCIA collection import from the admin dashboard | `ImportTCIASeries` |
| `desktop.installer-pair` | First-launch pairing of the desktop installer | `PairDesktopInstaller` |

This list is intentionally small to start. New methods get added to the
enum + a default-on entry in the allowlist as they ship.

## Schema

New table — single source of truth for what each institution is
permitted to do at upload time.

```sql
-- migration NNN_create_institution_upload_allowlist.sql
CREATE TABLE institution_upload_allowlist (
    institution_id  UUID    NOT NULL REFERENCES institutions(id) ON DELETE CASCADE,
    method_id       TEXT    NOT NULL,    -- e.g. 'browser.web-upload'
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    note            TEXT    NOT NULL DEFAULT '',  -- admin-facing rationale
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by      TEXT    NOT NULL,
    PRIMARY KEY (institution_id, method_id)
);
```

Default behavior: the allowlist is *additive in the deny direction* —
absence of a row for `(inst, method_id)` means "allowed". A row with
`enabled = FALSE` explicitly denies. This makes back-compat free (every
existing institution sees zero deny rows = all methods allowed) and
makes the deny set easy to audit ("show me everything an institution
has turned off").

## Resolution + enforcement

```go
// In api/handler/upload.go (and the other listed handlers):

if inst := s.requireInstitutionForUpload(w, r); inst != nil {
    allowed, err := model.IsUploadMethodAllowed(ctx, db, inst.ID, "browser.web-upload")
    if err != nil { ... 500 ... }
    if !allowed {
        s.writeError(w, http.StatusForbidden,
            "this upload method is disabled for your institution")
        return
    }
}
```

`requireInstitutionForUpload` reuses the institution attribution logic
that already runs in `UploadInit` to identify the calling site
(explicit selector, IP-range match, AE-title match — see CLAUDE.md
§"Institution Management"). If no institution can be attributed the
upload defaults to allowed (legacy/anonymous uploads stay open).

## API

```
GET    /api/institutions/{id}/upload-allowlist           — list rows + computed effective state
PUT    /api/institutions/{id}/upload-allowlist/{methodID} — body: {"enabled": false, "note": "..."}
DELETE /api/institutions/{id}/upload-allowlist/{methodID} — drop the row (reverts to default-on)
```

`GET` returns the *full* effective allowlist (one entry per known
method) so the UI can render every toggle with its current state and a
"this is the default" pill on rows that fall through to default. That
keeps the UI simple — render the enum once, light up the rows that
have an explicit deny.

## Admin dashboard UI

New sub-section in `InstitutionDetailPanel`, sibling to Satellites and
Installer Invitations:

```
Browser upload allowlist  (5 of 5 methods enabled)
  ☑  Standard web upload                 [edit]
  ☑  Share-token redeem                  [edit]
  ☑  Pull from PACS (DIMSE-C-MOVE)       [edit]
  ☑  TCIA collection import              [edit]
  ☑  Desktop installer pairing           [edit]

[edit] opens an inline form: enabled toggle + optional note.
```

Keep the section read-only for non-admin viewers; only `adminOnly`
endpoints accept writes.

## Test plan

- Default-on: institution with zero allowlist rows can call every
  upload handler.
- Explicit deny: PUT method_id = `browser.dimse-pull` with
  `enabled=false` → the dimse-pull handler responds 403 for callers
  attributed to that institution.
- Cascade: deleting the institution removes its allowlist rows
  (ON DELETE CASCADE).
- Audit: every PUT/DELETE emits a `institution.upload_allowlist.*`
  audit entry with the method_id + before/after.
- Cross-institution isolation: a deny for institution A does not
  affect institution B.

## Rollout

1. **Schema only** — first PR creates the table and the model/handler
   surface, but every handler still allows every method (no `IsUploadMethodAllowed`
   call sites). UI ships behind a `VITE_UPLOAD_ALLOWLIST_ENABLED` flag.
2. **Enforcement** — second PR wires the check into each handler,
   one at a time, with a feature flag per method so a misconfigured
   row can be reverted without a redeploy.
3. **UI default-on** — once enforcement has run quietly for a release,
   remove the build flag.

This keeps the blast radius small and lets us catch attribution edge
cases (e.g. a legitimate upload that isn't getting attributed to the
institution at all) before they become user-visible 403s.

## Open questions

- Should `browser.web-upload` be split by anonymization profile? E.g.
  "this institution can submit only studies that use anon profile X".
  Probably not — profile selection is already enforced project-side.
- Should `desktop.installer-pair` be in this allowlist at all, or
  treated as a separate "desktop client provisioning" concern? Putting
  it here keeps the operator UI in one place but conceptually it's a
  different decision than browser uploads. Recommendation: include it,
  but label clearly.
- Allowlist vs denylist UX: I've described it as a deny-set
  (default-on), but the table column is named `enabled` which reads as
  "this method is enabled = true/false". An explicit `enabled = TRUE`
  row is redundant with the absence-of-row default. We could either
  (a) only allow `enabled = FALSE` rows in the table, or (b) allow
  both and use the row to also carry the admin note even when enabled
  is true. Going with (b) for ergonomics — the note is useful even
  when the method is on ("expressly authorized by site IRB 2026-04-…").
