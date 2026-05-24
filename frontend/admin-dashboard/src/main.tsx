import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { DashboardLayout } from './layout/DashboardLayout'
import { HomePage } from './pages/HomePage'
import { ProjectPage } from './pages/ProjectPage'
import { SubjectPage } from './pages/SubjectPage'
import { StudyPage } from './pages/StudyPage'
import { AdminApp } from './pages/AdminApp'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <Routes>
        {/* XNAT-inspired Project → Subject → Study hierarchy. */}
        <Route element={<DashboardLayout />}>
          <Route index element={<HomePage />} />
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

        {/* Existing 18-tab admin experience, re-homed under /admin/*. */}
        <Route path="/admin/*" element={<AdminApp />} />

        {/* Anything unmatched falls back to the home page. */}
        <Route path="*" element={<DashboardLayout />}>
          <Route index element={<HomePage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  </StrictMode>,
)
