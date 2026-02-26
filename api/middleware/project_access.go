package middleware

import (
	"context"
	"database/sql"

	"github.com/aegis-imaging/aegis/api/model"
)

const projectAccessKey contextKey = "project_access"

// WithProjectAccess stores a UserProjectAccess in the context.
// Handlers call this after resolving the user's membership for the current project.
func WithProjectAccess(ctx context.Context, a *model.UserProjectAccess) context.Context {
	return context.WithValue(ctx, projectAccessKey, a)
}

// ProjectAccessFromContext retrieves the UserProjectAccess stored by WithProjectAccess.
// Returns nil when no project-level access has been injected (public routes, platform admin paths).
func ProjectAccessFromContext(ctx context.Context) *model.UserProjectAccess {
	a, _ := ctx.Value(projectAccessKey).(*model.UserProjectAccess)
	return a
}

// IsPlatformAdmin returns true when the user has a platform-wide admin or viewer role
// and therefore bypasses project-level membership checks.
func IsPlatformAdmin(user *AuthUser) bool {
	return user != nil && (user.Role == "admin" || user.Role == "viewer")
}

// CanAccessProject returns whether the authenticated user may access a given project.
//
//   - Platform admin/viewer → always allowed; returns nil access (no scoping needed)
//   - Researcher → must have a project_members entry; returns the access record
//   - If project.restricted=false and user is admin/viewer → allowed
//   - If project.restricted=true and user is researcher with no membership → denied
//
// Returns (nil, nil) when the user is a platform admin/viewer (access granted, no scoping).
// Returns (access, nil) when researcher has membership.
// Returns (nil, ErrNoRows) or a non-nil error when access is denied or DB fails.
func CanAccessProject(
	ctx context.Context,
	db *sql.DB,
	user *AuthUser,
	projectID string,
	projectRestricted bool,
) (*model.UserProjectAccess, bool, error) {
	// Platform admin and viewer always bypass project-level checks.
	if IsPlatformAdmin(user) {
		return nil, true, nil
	}

	// Researcher role: access depends entirely on project_members.
	access, err := model.GetUserAccessForProject(ctx, db, user.ID, projectID)
	if err != nil {
		return nil, false, err
	}
	if access == nil {
		// No membership entry found.
		// If the project is not restricted, platform viewers can still see it —
		// but researcher role users cannot (they have no membership).
		return nil, false, nil
	}
	return access, true, nil
}

// ResolveProjectAccess is a convenience function called by handlers that operate on a
// specific project. It:
//  1. Looks up the project's restricted flag
//  2. Checks whether the user may access it
//  3. Returns (projectRestricted, access, allowed, error)
//
// Use allowed=false to return 404 (not 403, to avoid leaking project existence).
func ResolveProjectAccess(
	ctx context.Context,
	db *sql.DB,
	user *AuthUser,
	projectID string,
) (restricted bool, access *model.UserProjectAccess, allowed bool, err error) {
	// Fetch project's restricted flag.
	if err = db.QueryRowContext(ctx,
		`SELECT restricted FROM projects WHERE id = $1`, projectID,
	).Scan(&restricted); err != nil {
		// If project doesn't exist, return not allowed (404 will be returned by handler).
		return false, nil, false, err
	}

	access, allowed, err = CanAccessProject(ctx, db, user, projectID, restricted)
	return
}
