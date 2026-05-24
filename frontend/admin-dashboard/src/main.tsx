import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { DashboardLayout } from './layout/DashboardLayout'
import { AboutLayout } from './layout/AboutLayout'
import { ProjectPage } from './pages/ProjectPage'
import { SubjectPage } from './pages/SubjectPage'
import { StudyPage } from './pages/StudyPage'
import { AdminApp } from './pages/AdminApp'
import { AboutPage } from './pages/AboutPage'
import { RootRedirect } from './pages/RootRedirect'

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

        {/* Authenticated researcher UI — XNAT-style Project → Subject → Study. */}
        <Route element={<DashboardLayout />}>
          <Route index element={<RootRedirect />} />
          <Route path="projects/:projectId" element={<ProjectPage />} />
          <Route
            path="projects/:projectId/subjects/:subjectId"
            element={<SubjectPage />}
          />
          <Route
            path="projects/:projectId/subjects/:subjectId/studies/:studyId"
            element={<StudyPage />}
          />
        </Route>

        {/* Existing 18-tab admin experience under /admin/*. */}
        <Route path="/admin/*" element={<AdminApp />} />

        {/* Anything unmatched falls back to the auth-aware root. */}
        <Route path="*" element={<DashboardLayout />}>
          <Route index element={<RootRedirect />} />
        </Route>
      </Routes>
    </BrowserRouter>
  </StrictMode>,
)
