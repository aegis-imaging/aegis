package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aegis-imaging/aegis/api/model"
)

// SeedProject returns the "default" project created by migration 001.
func SeedProject(t *testing.T, db *sql.DB) *model.Project {
	t.Helper()
	p, err := model.GetProjectBySlug(context.Background(), db, "default")
	if err != nil {
		t.Fatalf("seed project: %v", err)
	}
	return p
}

var studyCounter int

// CreateTestStudy creates a study with sensible defaults and returns it.
func CreateTestStudy(t *testing.T, db *sql.DB, projectID string) *model.Study {
	t.Helper()
	studyCounter++
	s := &model.Study{
		ProjectID:        projectID,
		StudyInstanceUID: fmt.Sprintf("1.2.3.4.%d.%d", time.Now().UnixNano(), studyCounter),
		Modality:         "MRI",
		BodyPart:         "HEAD",
		StudyDescription: "Test Study",
		SeriesCount:      1,
		InstanceCount:    10,
		Status:           "received",
		DicomStore:       "raw",
		Source:           "external",
	}
	if err := model.CreateStudy(context.Background(), db, s); err != nil {
		t.Fatalf("create test study: %v", err)
	}
	return s
}

// CreateTestAdminUser creates an admin user and returns it.
func CreateTestAdminUser(t *testing.T, db *sql.DB, email, role string) *model.AdminUser {
	t.Helper()
	u := &model.AdminUser{
		Email:   email,
		Name:    "Test User",
		Role:    role,
		Enabled: true,
	}
	if err := model.CreateAdminUser(context.Background(), db, u); err != nil {
		t.Fatalf("create test admin user: %v", err)
	}
	return u
}

// CreateTestDestination creates an enabled DICOMweb destination and returns it.
func CreateTestDestination(t *testing.T, db *sql.DB, name string) *model.Destination {
	t.Helper()
	d := &model.Destination{
		Name:        name,
		Slug:        name,
		Type:        "dicomweb",
		DicomwebURL: "https://example.com/dicomweb",
		Enabled:     true,
	}
	if err := model.CreateDestination(context.Background(), db, d); err != nil {
		t.Fatalf("create test destination: %v", err)
	}
	return d
}

// CreateTestInstitution creates an enabled institution and returns it.
func CreateTestInstitution(t *testing.T, db *sql.DB, name string) *model.Institution {
	t.Helper()
	inst := &model.Institution{
		Name:    name,
		Slug:    name,
		Type:    "sender",
		Enabled: true,
	}
	if err := model.CreateInstitution(context.Background(), db, inst); err != nil {
		t.Fatalf("create test institution: %v", err)
	}
	return inst
}

// CreateTestRoutingRule creates an enabled routing rule and returns it.
func CreateTestRoutingRule(t *testing.T, db *sql.DB, name, action string) *model.RoutingRule {
	t.Helper()
	r := &model.RoutingRule{
		Name:     name,
		Priority: 100,
		Enabled:  true,
		Action:   action,
	}
	if err := model.CreateRoutingRule(context.Background(), db, r); err != nil {
		t.Fatalf("create test routing rule: %v", err)
	}
	return r
}

// CreateTestProject creates a new project with the given name and a derived slug.
func CreateTestProject(t *testing.T, db *sql.DB, name string) *model.Project {
	t.Helper()
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-")) + fmt.Sprintf("-%d", time.Now().UnixNano())
	p, err := model.CreateProject(context.Background(), db, name, slug, "")
	if err != nil {
		t.Fatalf("create test project %q: %v", name, err)
	}
	return p
}

// CreateTestRoutingRuleForProject creates an enabled routing rule scoped to a specific project.
func CreateTestRoutingRuleForProject(t *testing.T, db *sql.DB, name, action, projectID string) *model.RoutingRule {
	t.Helper()
	r := &model.RoutingRule{
		Name:      name,
		Priority:  100,
		Enabled:   true,
		Action:    action,
		ProjectID: &projectID,
	}
	if err := model.CreateRoutingRule(context.Background(), db, r); err != nil {
		t.Fatalf("create test routing rule for project: %v", err)
	}
	return r
}
