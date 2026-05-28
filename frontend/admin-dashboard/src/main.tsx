import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { DashboardLayout } from './layout/DashboardLayout'
import { AboutLayout } from './layout/AboutLayout'
import { ProjectPage } from './pages/ProjectPage'
import { SubjectPage } from './pages/SubjectPage'
import { StudyPage } from './pages/StudyPage'
import { AdminApp } from './pages/AdminApp'
import { AboutPage } from './pages/AboutPage'
import { RootRedirect } from './pages/RootRedirect'
import { ProfileLanding } from './pages/ProfileLanding'
import { ProfileNotifications } from './pages/ProfileNotifications'
import { ProfileActivity } from './pages/ProfileActivity'
import { ProjectSettingsPage } from './pages/ProjectSettingsPage'
import { AdminTabRedirect } from './pages/AdminTabRedirect'
import { TCIAPanel } from './components/TCIAPanel'

// Canonical admin-tab slugs. Each renders the same AdminApp dispatcher which
// reads location.pathname via parseAdminTab to pick the right tab. Keeping
// the list here (not imported from App.tsx) so main.tsx doesn't pull the
// 12k-line App module into the root bundle's critical path — but the names
// must stay in sync with ADMIN_TABS in App.tsx.
//
// tcia_import is intentionally NOT in this list — it renders TCIAPanel
// directly (researcher-facing, not an admin tab dispatcher entry).
// Including it here would route /tcia_import through AdminApp, whose
// parseAdminTab can't find tcia_import in ADMIN_TABS and silently falls
// through to 'studies'. Net effect: TCIA Import breadcrumb above the
// Studies panel.
const ADMIN_TAB_SLUGS = [
  'studies', 'audit', 'shares', 'routing', 'dimse_ops', 'institutions',
  'satellites', 'profiles', 'protocol_templates', 'notifications',
  'federation', 'users', 'api_keys', 'invite_codes',
  'downloads', 'system', 'agent',
] as const

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <Routes>
        {/* Public /about/* — no auth probe. The IAP load balancer routes
            these paths through the no-IAP backend service so anonymous
            visitors can reach them without a login redirect. */}
        <Route path="/about" element={<AboutLayout />}>
          <Route index element={<AboutPage />} />
        </Route>

        {/* Authenticated app — every page mounts under DashboardLayout so
            the topbar / breadcrumbs / sidebar stay stable across nav,
            and clicking sidebar items triggers a clean Outlet swap.

            Critical: each admin tab gets its OWN <Route> entry. Earlier
            versions had a single `<Route path="admin/*" element={<AdminApp />} />`
            catch-all, which meant clicking /admin/audit → /admin/routing
            matched the SAME route, so React reused the AdminApp instance
            and the conditional renders inside App kept stale state — the
            recurring "dead admin links" symptom. With one Route per tab,
            React sees each click as a route swap, mounts a fresh element,
            and the conditional renders re-evaluate cleanly. */}
        <Route element={<DashboardLayout />}>
          <Route index element={<RootRedirect />} />

          {/* Researcher project tree. /projects (no param) is the admin
              projects list; /projects/:id is the researcher detail page.
              React Router matches the more specific :id route first, so
              the two coexist without collision. */}
          <Route path="projects" element={<AdminApp />} />
          <Route path="projects/:projectId" element={<ProjectPage />} />
          <Route path="projects/:projectId/settings" element={<ProjectSettingsPage />} />
          <Route path="projects/:projectId/subjects/:subjectId" element={<SubjectPage />} />
          <Route path="projects/:projectId/subjects/:subjectId/studies/:studyId" element={<StudyPage />} />

          {/* Self-service profile pages. */}
          <Route path="profile" element={<ProfileLanding />} />
          <Route path="profile/notifications" element={<ProfileNotifications />} />
          <Route path="profile/activity" element={<ProfileActivity />} />

          {/* Every admin tab at root. Single source of truth: ADMIN_TAB_SLUGS. */}
          {ADMIN_TAB_SLUGS.map(slug => (
            <Route key={slug} path={slug} element={<AdminApp />} />
          ))}

          {/* Sub-route for institution detail. /institutions/:id resolves to
              the same AdminApp; the admin panel reads the id from the URL. */}
          <Route path="institutions/:id" element={<AdminApp />} />

          {/* TCIA collection import — researcher-facing, renders the panel
              directly (not via AdminApp). /tcia is the legacy short URL. */}
          <Route path="tcia_import" element={<TCIAPanel />} />
          <Route path="tcia" element={<Navigate to="/tcia_import" replace />} />

          {/* Back-compat: every legacy /admin/* URL redirects to the root
              variant. Audit records, email links, external bookmarks, and
              in-flight tabs all keep resolving without breakage. */}
          <Route path="admin" element={<Navigate to="/studies" replace />} />
          <Route path="admin/:tab" element={<AdminTabRedirect />} />
          <Route path="admin/:tab/:detail" element={<AdminTabRedirect />} />
        </Route>

        {/* Anything unmatched falls back to the auth-aware root. */}
        <Route path="*" element={<DashboardLayout />}>
          <Route index element={<RootRedirect />} />
        </Route>
      </Routes>
    </BrowserRouter>
  </StrictMode>,
)
