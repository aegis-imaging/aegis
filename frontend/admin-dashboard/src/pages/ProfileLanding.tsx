import { Link } from 'react-router-dom'

// ProfileLanding is the index page at /profile — a small landing card list
// that points to each profile sub-section. Keeps the URL space tidy
// (/profile/activity, /profile/notifications, future /profile/settings, ...)
// while giving the user a discoverable entry point.
export function ProfileLanding() {
  return (
    <>
      <header className="aegis-page-header">
        <div className="aegis-page-header-row">
          <div>
            <h1>Your profile</h1>
            <p className="aegis-page-description">
              Self-service settings and your own activity. Anything you change here only affects your account.
            </p>
          </div>
        </div>
      </header>

      <ul className="aegis-card-grid">
        <li className="aegis-card">
          <Link to="/profile/notifications" className="aegis-card-link">
            <div className="aegis-card-title">Notification preferences</div>
            <p className="aegis-card-desc">
              Digest email frequency and which events should ping you.
            </p>
          </Link>
        </li>
        <li className="aegis-card">
          <Link to="/profile/activity" className="aegis-card-link">
            <div className="aegis-card-title">Your activity</div>
            <p className="aegis-card-desc">
              The audit entries attributed to you — uploads, approvals, shares, etc.
            </p>
          </Link>
        </li>
      </ul>
    </>
  )
}
