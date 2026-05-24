# AEGIS Admin Dashboard

React 19 + Vite SPA for AEGIS administrators and image analysts. Provides
study management, pipeline operations, audit/compliance views, and the
researcher-facing project → subject → study drill-down.

## Development

```bash
npm install
npm run dev          # serves on :3001
npm run typecheck    # tsc -b without emit
npm run build        # type-check + Vite production build
```

The dev server proxies API requests to `http://localhost:8080` (see
`vite.config.ts`). Bring up the Go API with `cd ../../api && go run .` and
the local docker-compose stack with `docker compose up -d` from the repo
root if the API needs Postgres/Mailpit.

## Navigation

After login, users land on `/` which shows the projects they have access to,
recent activity, and a top-bar search. Drilling in follows:

```
/                                                home
/projects/:projectId                             project detail (subjects, sessions, access)
/projects/:projectId/subjects/:subjectId         subject detail (demographics, studies)
/projects/:projectId/subjects/:subjectId/studies/:studyId   study detail
/admin/*                                         operator tabs (routing, profiles, users, ...)
```

## Inspired By

The Project → Subject → Study navigation hierarchy is inspired by
[XNAT](https://www.xnat.org), the open-source neuroimaging data-management
platform maintained by the Neuroinformatics Research Group at Washington
University School of Medicine. XNAT is licensed under the 2-Clause BSD
license (Copyright (c) 2019, Washington University School of Medicine).

**No XNAT source code is included in AEGIS.** XNAT served only as a
conceptual UX reference — the patterns familiar to working neuroimaging
researchers (subjects table inside a project, sessions by date inside a
subject, exact-match top-bar search). All code in this dashboard is
original to AEGIS. See `/CREDITS.md` at the repository root for the full
attribution.
