# Admin Dashboard Redesign — XNAT-style Restyle

## Context

The new XNAT-style researcher home (`/`) shipped via #497 introduced a
clean `aegis-*` design system: teal/orange colorblind-safe palette,
top-bar nav, breadcrumbs, card-based `aegis-section` containers. It feels
modern and purpose-built for academic research.

The existing admin section (`/admin/*`) still uses the legacy chrome:
collapsible left sidebar, generic inline styles, mixed dark/light
theming, per-tab ad-hoc visual conventions. The contrast between `/`
and `/admin/*` is jarring — researchers who become admins see two
different products. The goal of this redesign is to bring `/admin/*`
visually in line with the XNAT chrome while preserving every existing
feature.

## Non-goals

- **No role-model changes.** The current `admin / viewer / researcher`
  global roles + per-project `owner / coordinator / reviewer /
  site_coordinator / site_viewer` ACL roles stay as-is. A future PR
  can revisit if needed.
- **No feature removal.** Every existing admin function continues to
  exist somewhere in the post-redesign app. Some features may *move*
  out of `/admin/*` into the regular nav, but nothing is deleted.
- **No backend changes.** The Go API, terraform, and CI stay
  untouched. This is purely a frontend (React + CSS) effort.

## Visual direction

The new admin uses the same `aegis-*` design system as the researcher
home:

- **Top bar** (`.aegis-topbar`): brand on left, primary nav in the middle
  (Home / Admin), global search on right. Already exists from #497 —
  the new admin lives under this same shell, not in a separate
  sidebar.
- **Left sidebar** (re-skinned, not removed): the existing nav groups
  (Overview / Data / Config / Advanced / Admin) stay but use
  `aegis-sidenav-*` classes with the teal/orange palette, the same hover
  + active states as the topbar, and consistent typography. The
  collapse toggle stays.
- **Page chrome**: every tab gets a `aegis-page-header` with title +
  description, then content in `aegis-section` cards. No more bare
  tables on white backgrounds — every block lives in a card.
- **Tables**: replace inline-styled tables with the existing
  `aegis-table` class (used on `ProjectPage`, `SubjectPage`). Striped
  rows, sticky headers, hover states, the `aegis-pill` status indicators
  already defined.
- **Forms**: use the `aegis-form-row` / `aegis-btn-primary` /
  `aegis-btn-secondary` classes added in #511. No more inline-styled
  inputs.
- **Pipeline stage indicators**: reuse the `aegis-stage-*` glyph + color
  combo from `SubjectPage.tsx` (○ ◌ ● ◐ ✕ —) for any status display.
- **Dark mode**: keep the existing toggle but ensure `aegis-*` classes
  read from CSS variables that respond to it (already wired in
  `nav.css`).

## Tab categorization (where each feature lives after redesign)

Based on the audit (see `docs/admin-tab-audit.md` — to be created
alongside the first PR), tabs fall into three groups:

### Stays in `/admin/*` (pure ops surface, 11 tabs)

- **studies** — global pipeline triage (admin can see/act on all)
- **agent** — AI agent panel for ad-hoc study queries
- **routing** — routing rules + destinations + analytics
- **dimse_ops** — DIMSE receiver health, retries, DLQ
- **spokes** — on-prem AEGIS Router enrollment
- **projects** — global project CRUD, PHI config, retention, quotas
- **federation** — peer AEGIS instance registry
- **users** — global user CRUD + roles
- **api_keys** — API key management
- **invite_codes** — invite/registration tokens
- **downloads** — bulk study export / archive
- **system** — infrastructure health

### Moves out to researcher nav (1 tab)

- **notifications** — digest email + webhook subscriptions are
  self-service. Should live under a new `/profile/notifications` or
  similar in the XNAT nav, scoped to the signed-in user.

### Split between admin and researcher contexts (6 tabs)

- **audit** — global audit stays in `/admin/audit`; new
  `/profile/activity` shows the user's own audit entries
- **shares** — global share manager stays in `/admin/shares`;
  per-study share creation moves inline into the study detail panel
  on the XNAT nav
- **institutions** — system institutions in `/admin/institutions`;
  per-project institution config under `/projects/:id/settings`
- **profiles** — anonymization profiles per-project under
  `/projects/:id/settings`; defaults in `/admin/profiles`
- **protocol_templates** — per-project rules under
  `/projects/:id/settings`; system templates in `/admin/protocols`
- **tcia_import** — keep in `/admin/tcia` for now (clarify
  researcher-vs-admin scope in a follow-up audit)

This split is **the part most likely to need iteration**. The PR
plan below lands the visual redesign first; the categorization
moves come after the redesign is stable.

## Files affected

- `frontend/admin-dashboard/src/App.tsx` — every per-tab block needs
  re-styling. Extracting tabs into their own routed page components
  (per-tab files like `AdminStudiesPage.tsx`) is the cleanest path;
  one PR per tab keeps the diff small.
- `frontend/admin-dashboard/src/styles/nav.css` — additions for
  sidebar nav, page header, page layout containers, per-tab visual
  patterns.
- `frontend/admin-dashboard/src/styles/panels.css`,
  `dark-theme.css`, `forms-shares.css`, `viewers.css`,
  `components.css`, `layout.css` — legacy CSS files that will be
  progressively migrated into `aegis-*` patterns and eventually deleted.
- `frontend/admin-dashboard/src/pages/admin/` — new directory for
  the per-tab page components extracted from `App.tsx`.

## PR sequence

Each PR is independent and small enough to review in one sitting.
After each lands, the next can begin from a clean develop.

### PR 1 — Admin shell restyle (no functional changes)

Apply `aegis-*` chrome to `/admin/*`: topbar (already present from #497
via `<AdminApp>` wrapper, but the inner admin still has its own
topbar — unify), left sidebar restyled with `aegis-sidenav-*` classes,
page header strip with `aegis-page-header`. The 18 tab body blocks stay
visually unchanged in this PR — only the chrome around them
changes. Smallest PR, biggest visual delta.

### PR 2 — Extract first 3 tabs into routed pages

Move `studies`, `audit`, and `users` into
`frontend/admin-dashboard/src/pages/admin/`. Each tab becomes a real
React Router route (`/admin/studies`, `/admin/audit`,
`/admin/users`). The bodies stay in their current form (legacy
styling) so the diff is mechanical. Validates the extraction
pattern.

### PR 3 — Restyle the first 3 tabs

Now that they're isolated files, apply `aegis-table` + `aegis-section` +
`aegis-form-row` patterns to studies/audit/users. Drop the inline
styles. ~3 small PRs would also work (one per tab) if the diff is
too big.

### PRs 4–7 — Extract + restyle the remaining 15 tabs

Repeat the PR-2/PR-3 pattern in groups of 3-5 tabs. Order suggestion
(by visual complexity, simplest first):
- Group A: api_keys, invite_codes, federation (simple tables)
- Group B: institutions, profiles, protocol_templates (form-heavy)
- Group C: routing, projects, dimse_ops (dashboardy)
- Group D: shares, downloads, system, agent, spokes, tcia_import,
  notifications (mixed bag)

### PR 8 — Tab re-categorization

Move `notifications` out to a new `/profile/notifications` route in
the researcher nav. Split `audit` into global (`/admin/audit`) and
personal (`/profile/activity`). This is the biggest semantic change;
ship it last so users get used to the visual redesign first.

### PR 9 — Cleanup

Delete legacy CSS files (`panels.css`, `dark-theme.css`,
`forms-shares.css`) once nothing references them. Drop the
collapsible-sidebar localStorage state if the new sidebar handles
collapse differently.

## Risks & mitigations

- **Regression risk on individual tabs**: each extraction PR is a
  mechanical move; behavior should be identical. Mitigation: smoke
  test by clicking through each tab after each PR.
- **Visual inconsistency during transition**: PRs 1-7 will have a
  mix of restyled + legacy tabs for a few hours/days. Acceptable;
  the alternative (one giant PR) is worse.
- **Dark mode breakage**: legacy CSS may have dark-mode tweaks that
  the new `aegis-*` system doesn't yet have. Mitigation: each restyle
  PR explicitly tests dark mode and adds variables as needed.
- **Behavior changes**: some tabs use ad-hoc state management
  (localStorage keys, scroll positions). Mitigation: keep all
  state-management code unchanged in PRs 1-7; only PR 8+ touches
  semantics.

## Verification per PR

After each PR's deploy:

- [ ] Visit `/admin/<tab>` in browser, click through each
  sub-section.
- [ ] Toggle dark mode, verify everything is still readable.
- [ ] Run any tab-specific workflow (e.g. create a routing rule,
  approve a study, etc.) and confirm the data flow works end-to-end.
- [ ] Mobile/narrow viewport: at minimum, the sidebar should
  collapse cleanly. Tablet+ is the target; phone is not.

## Out of scope (followups, not blocking)

- Role-model refinement (deferred per user direction; current model
  is fine for now).
- Mobile-first responsive design.
- Light/dark theme variable cleanup (the current `--aegis-*` variables
  already work; legacy CSS variables under `dark-theme.css` can be
  retired in PR 9).
- Search inside the admin section (currently only the top-bar search
  spans projects/subjects/studies; admin tabs have their own
  filters).
- Replacing the agent/AI panel with a more polished chat UI.
