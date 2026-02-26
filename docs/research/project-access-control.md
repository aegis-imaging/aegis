# AEGIS Project-Level Access Control & Site-Based Permissions

**Status:** Implemented (migrations 066–068, PR #319)
**Date:** 2026-02-26

---

## Background

Prior to this work, AEGIS had a flat two-tier access model:

- **`admin`** — platform super-admin; full read/write access to everything
- **`viewer`** — platform read-only; no write operations

All authenticated users could see all projects and all studies. There was no concept of:
- A project being restricted to specific users
- A user being scoped to only one institution's studies within a project
- Per-project roles (PI, data manager, site coordinator, etc.)

This design was adequate for internal operator use but insufficient for externally-facing research deployments where different research teams work on independent projects and site-level researchers should only access data they submitted.

---

## Clinical Trial Model

The access control system is modelled on **multi-site clinical trial data management**. Each AEGIS project represents a study with:

```
Project: "ADNI-5 Alzheimer's MRI Study"
│
├── Coordinating Center (Principal Investigator's institution)
│   ├── PI (owner)              → sees ALL sites' data, full control
│   ├── Data Manager (coordinator) → sees ALL sites' data, write access
│   └── QC Reviewer (reviewer)  → sees ALL sites' data, read + approve/reject
│
├── Site 1: Mayo Clinic           → Site Coordinator sees ONLY Mayo studies
├── Site 2: Johns Hopkins         → Site Coordinator sees ONLY Hopkins studies
├── Site 3: UCSF                  → Site Coordinator sees ONLY UCSF studies
└── Site 4: Mass General          → Site Coordinator sees ONLY MGH studies
```

### Key properties

1. **Multiple independent projects** — a user can be a PI on Project A and a site coordinator on Project B; roles are per-project, not global
2. **Coordinating center vs. participating sites** — the coordinating center controls everything; sites only see their own submissions
3. **Walled-off projects** — when `restricted=true`, only project members and platform admins can see the project exists
4. **Institution scoping** — site roles (site_coordinator, site_viewer) are tied to a specific institution's `institution_id`; their study queries automatically filter on `study.institution_id`

---

## Data Model

### New table: `project_members` (migration 066)

```sql
CREATE TABLE project_members (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    admin_user_id   UUID        NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
    role            TEXT        NOT NULL CHECK (role IN (
                                    'owner', 'coordinator', 'reviewer',
                                    'site_coordinator', 'site_viewer')),
    institution_id  UUID        REFERENCES institutions(id) ON DELETE SET NULL,
    notes           TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, admin_user_id)
);
```

- `institution_id = NULL` → coordinating center member; sees all studies in the project
- `institution_id = <uuid>` → site member; sees only studies where `study.institution_id` matches

### New column: `projects.restricted` (migration 067)

```sql
ALTER TABLE projects ADD COLUMN restricted BOOLEAN NOT NULL DEFAULT false;
```

- `false` (default) — backwards-compatible; all authenticated users can see the project
- `true` — only project members and platform admins (admin/viewer role) can see the project

### New role: `admin_users.role = 'researcher'` (migration 068)

```sql
ALTER TABLE admin_users DROP CONSTRAINT IF EXISTS admin_users_role_check;
ALTER TABLE admin_users ADD CONSTRAINT admin_users_role_check
    CHECK (role IN ('admin', 'viewer', 'researcher'));
```

The `researcher` role means "project-scoped user". A researcher's access is determined entirely by their `project_members` entries. Researchers cannot see projects they are not a member of (when `restricted=true`).

---

## Role Definitions

### Platform roles (`admin_users.role`) — unchanged for existing users

| Role | Description |
|------|-------------|
| `admin` | Platform super-admin (AEGIS operators). Sees all projects/studies. Bypasses all project-level checks. |
| `viewer` | Platform read-only. Sees all projects/studies. Bypasses project-level checks. |
| `researcher` | Project-scoped user. Access determined entirely by `project_members` entries. |

### Project roles (`project_members.role`) — new clinical-trial layer

**Coordinating Center roles** (`institution_id = NULL` — see all studies):

| Role | Typical User | Write Capabilities |
|------|-------------|-------------------|
| `owner` | Principal Investigator | Full: manage members, project settings, approve/reject, configure pipeline |
| `coordinator` | Data Manager / Study Coordinator | Write: approve/reject/flag, manage routing + protocol templates |
| `reviewer` | QC Reviewer / Statistician | Limited: approve/reject/flag, add notes; cannot edit project settings |

**Site roles** (`institution_id = <uuid>` — own site's studies only):

| Role | Typical User | Write Capabilities |
|------|-------------|-------------------|
| `site_coordinator` | Site Research Coordinator | Upload studies, add notes and flags for own studies |
| `site_viewer` | Site Monitor / Sponsor Representative | Read-only; cannot upload or modify |

---

## Access Matrix

| Action | platform `admin` | platform `viewer` | `owner` | `coordinator` | `reviewer` | `site_coordinator` | `site_viewer` |
|--------|---|---|---|---|---|---|---|
| See restricted project | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| See all-site studies | ✓ | ✓ | ✓ | ✓ | ✓ | own site only | own site only |
| Approve/reject studies | ✓ | — | ✓ | ✓ | ✓ | — | — |
| Edit project settings | ✓ | — | ✓ | — | — | — | — |
| Manage project members | ✓ | — | ✓ | — | — | — | — |
| Upload/ingest studies | ✓ | ✓ | ✓ | ✓ | — | ✓ (own site) | — |

---

## Implementation Summary

### Go API

| File | Change |
|------|--------|
| `api/migrate/migrations/066_create_project_members.sql` | New — `project_members` table |
| `api/migrate/migrations/067_add_project_restricted.sql` | New — `restricted` column on projects |
| `api/migrate/migrations/068_add_researcher_role.sql` | New — `researcher` role constraint |
| `api/model/project_member.go` | New — CRUD for ProjectMember + UserProjectAccess |
| `api/middleware/project_access.go` | New — context helpers, CanAccessProject, ResolveProjectAccess |
| `api/middleware/auth.go` | Modified — added OptionalAuth middleware |
| `api/handler/project_member.go` | New — List/Add/Update/Remove project members |
| `api/handler/project.go` | Modified — ListProjects filters by user role; SetProjectRestricted handler |
| `api/handler/study.go` | Modified — ListStudies injects institution_id filter for site roles |
| `api/model/project.go` | Modified — added restricted/member_count fields; ListProjectsForResearcher, ListProjectsPublic, SetProjectRestricted |
| `api/main.go` | Modified — GET /api/projects uses optionalAuth; new member + restricted routes |

### New API endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/projects/{id}/members` | `auth` | List project members |
| `POST` | `/api/projects/{id}/members` | `auth` (+ handler role check) | Add a member |
| `PUT` | `/api/projects/{id}/members/{memberID}` | `auth` (+ handler role check) | Update a member |
| `DELETE` | `/api/projects/{id}/members/{memberID}` | `auth` (+ handler role check) | Remove a member |
| `PUT` | `/api/projects/{id}/restricted` | `adminOnly` | Toggle restricted flag |

### Modified endpoints

| Endpoint | Change |
|----------|--------|
| `GET /api/projects` | Moved from public to `optionalAuth`. Unauthenticated → non-restricted projects only. `researcher` → membership list. `admin`/`viewer` → all projects. |
| `GET /api/studies` | For `researcher` users with `site_coordinator`/`site_viewer` role: automatically injects `institution_id` filter. |

### Frontend (admin dashboard)

- `ProjectMembersPanel` component — modal for listing/adding/editing/removing project members
- Restricted toggle button per project row (orange styling when restricted, with confirmation prompt)
- "Access" column (Open/Restricted badge) and "Members" count column in projects table
- `Project` type extended with `restricted` and `member_count` fields
- `AdminUser` type extended with `researcher` role option

### Frontend (upload portal)

- Calls `GET /api/auth/me` on mount (non-blocking)
- Shows a teal context banner for researcher users: "Logged in as [name] — viewing project: [project name]" (when 1 project) or project count
- Project list is already filtered server-side by the API

### MCP server

New read tool: `list_project_members`

New write tools:
- `add_project_member` — add a user to a project with a role and optional institution scoping
- `update_project_member` — change a member's role or institution scoping
- `remove_project_member` — remove a user from a project
- `toggle_project_restricted` — set or clear the restricted flag

Updated tools:
- `list_projects` — description updated to mention `restricted` and `member_count` fields
- `create_admin_user` / `update_admin_user` — `researcher` added as valid role value

---

## Backwards Compatibility

- All existing `admin` and `viewer` users continue to work unchanged
- Existing projects get `restricted=false` by default — no behaviour change
- `GET /api/projects` still returns data for unauthenticated callers (non-restricted projects only)
- Upload portal works without login for public (non-restricted) projects

---

## Operational Workflow

### Onboarding a new research team

```bash
# 1. Create users
curl -X POST /api/admin-users -d '{"email":"pi@univ.edu","name":"Dr. PI","role":"researcher"}'
curl -X POST /api/admin-users -d '{"email":"coord@site1.com","name":"Site Coord","role":"researcher"}'

# 2. Create (or identify) the project
curl -X POST /api/projects -d '{"name":"ADNI-5 Pilot","slug":"adni5-pilot"}'

# 3. Restrict the project to members only
curl -X PUT /api/projects/<project-id>/restricted -d '{"restricted":true}'

# 4. Add PI as project owner (sees all sites)
curl -X POST /api/projects/<project-id>/members \
  -d '{"admin_user_id":"<pi-uuid>","role":"owner"}'

# 5. Add site coordinator scoped to their institution
curl -X POST /api/projects/<project-id>/members \
  -d '{"admin_user_id":"<coord-uuid>","role":"site_coordinator","institution_id":"<institution-uuid>"}'

# 6. Verify: set DEV_USER_EMAIL=pi@univ.edu, GET /api/projects → returns only this project
# 7. Verify: set DEV_USER_EMAIL=coord@site1.com, GET /api/studies?project_id=<id>
#            → returns only studies where institution_id = <institution-uuid>
```
