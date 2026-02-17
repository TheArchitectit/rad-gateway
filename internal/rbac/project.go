// Package rbac provides Role-Based Access Control functionality.
package rbac

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// ProjectContext represents the project isolation context for a request.
// This ensures users can only access resources within their assigned projects.
type ProjectContext struct {
	// ProjectID is the unique identifier for the project
	ProjectID string
	// ProjectSlug is the URL-friendly identifier
	ProjectSlug string
	// WorkspaceID is the parent workspace containing the project
	WorkspaceID string
	// AllowedProjects is the list of projects the user has access to
	AllowedProjects []string
	// IsWildcardAccess allows access to all projects (Admin only)
	IsWildcardAccess bool
}

// ProjectContextKey is the context key for project information.
type ProjectContextKey struct{}

// ProjectHeaderKey is the HTTP header for specifying project scope.
const ProjectHeaderKey = "X-Project-Id"

// WorkspaceHeaderKey is the HTTP header for specifying workspace scope.
const WorkspaceHeaderKey = "X-Workspace-Id"

// Context keys for storing project information.
type projectCtxKey string

const (
	projCtxKeyProjectID      projectCtxKey = "rbac_project_id"
	projCtxKeyProjectSlug    projectCtxKey = "rbac_project_slug"
	projCtxKeyWorkspaceID    projectCtxKey = "rbac_workspace_id"
	projCtxKeyAllowedProjs   projectCtxKey = "rbac_allowed_projects"
	projCtxKeyWildcardAccess projectCtxKey = "rbac_wildcard_access"
)

// ProjectStore defines the interface for project-related queries.
// Implementations should handle database or cache lookups.
type ProjectStore interface {
	// GetProjectByID retrieves a project by its ID
	GetProjectByID(projectID string) (*Project, error)
	// GetProjectBySlug retrieves a project by its slug
	GetProjectBySlug(slug string) (*Project, error)
	// GetUserProjects retrieves all projects a user has access to
	GetUserProjects(userID string) ([]Project, error)
	// UserHasProjectAccess checks if a user has access to a specific project
	UserHasProjectAccess(userID, projectID string) (bool, error)
}

// Project represents a project entity for isolation purposes.
type Project struct {
	ID          string
	Slug        string
	Name        string
	WorkspaceID string
	Status      string
}

// NewProjectContext creates a new project context.
func NewProjectContext(projectID, projectSlug, workspaceID string, allowedProjects []string, wildcard bool) *ProjectContext {
	return &ProjectContext{
		ProjectID:        projectID,
		ProjectSlug:      projectSlug,
		WorkspaceID:      workspaceID,
		AllowedProjects:  allowedProjects,
		IsWildcardAccess: wildcard,
	}
}

// IsAllowed checks if access to a specific project is allowed.
func (pc *ProjectContext) IsAllowed(projectID string) bool {
	if pc == nil {
		return false
	}
	// Wildcard access allows any project
	if pc.IsWildcardAccess {
		return true
	}
	// Check if project is in allowed list
	for _, allowed := range pc.AllowedProjects {
		if allowed == projectID {
			return true
		}
	}
	return false
}

// IsCurrentProject checks if the given project matches the current context.
func (pc *ProjectContext) IsCurrentProject(projectID string) bool {
	if pc == nil {
		return projectID == ""
	}
	return pc.ProjectID == projectID
}

// Validate returns an error if the project context is invalid.
func (pc *ProjectContext) Validate() error {
	if pc == nil {
		return fmt.Errorf("project context is nil")
	}
	if pc.ProjectID == "" && !pc.IsWildcardAccess {
		return fmt.Errorf("project ID is required")
	}
	if pc.WorkspaceID == "" {
		return fmt.Errorf("workspace ID is required")
	}
	return nil
}

// ExtractProjectFromRequest extracts project information from an HTTP request.
// It checks headers, URL parameters, and JWT claims in order of priority.
func ExtractProjectFromRequest(r *http.Request) (*ProjectContext, error) {
	// First, try header
	projectID := strings.TrimSpace(r.Header.Get(ProjectHeaderKey))
	workspaceID := strings.TrimSpace(r.Header.Get(WorkspaceHeaderKey))

	// Then try URL query parameter
	if projectID == "" {
		projectID = strings.TrimSpace(r.URL.Query().Get("project_id"))
	}

	// Try to get from context (set by JWT middleware)
	ctx := r.Context()
	if v := ctx.Value(projCtxKeyProjectID); v != nil {
		if id, ok := v.(string); ok && id != "" {
			projectID = id
		}
	}
	if v := ctx.Value(projCtxKeyWorkspaceID); v != nil {
		if id, ok := v.(string); ok && id != "" {
			workspaceID = id
		}
	}

	// Validate we have minimum required information
	if projectID == "" && workspaceID == "" {
		return nil, fmt.Errorf("project or workspace identification required")
	}

	// Build context
	ctx_proj := &ProjectContext{
		ProjectID:   projectID,
		WorkspaceID: workspaceID,
	}

	// Check for wildcard access
	if v := ctx.Value(projCtxKeyWildcardAccess); v != nil {
		if wildcard, ok := v.(bool); ok {
			ctx_proj.IsWildcardAccess = wildcard
		}
	}

	// Get allowed projects from context
	if v := ctx.Value(projCtxKeyAllowedProjs); v != nil {
		if allowed, ok := v.([]string); ok {
			ctx_proj.AllowedProjects = allowed
		}
	}

	return ctx_proj, nil
}

// WithProjectContext adds project context to a Go context.
func WithProjectContext(ctx context.Context, pc *ProjectContext) context.Context {
	if pc == nil {
		return ctx
	}
	ctx = context.WithValue(ctx, projCtxKeyProjectID, pc.ProjectID)
	ctx = context.WithValue(ctx, projCtxKeyProjectSlug, pc.ProjectSlug)
	ctx = context.WithValue(ctx, projCtxKeyWorkspaceID, pc.WorkspaceID)
	ctx = context.WithValue(ctx, projCtxKeyAllowedProjs, pc.AllowedProjects)
	ctx = context.WithValue(ctx, projCtxKeyWildcardAccess, pc.IsWildcardAccess)
	return ctx
}

// GetProjectContext retrieves the project context from a Go context.
func GetProjectContext(ctx context.Context) *ProjectContext {
	if ctx == nil {
		return nil
	}

	projectID, _ := ctx.Value(projCtxKeyProjectID).(string)
	projectSlug, _ := ctx.Value(projCtxKeyProjectSlug).(string)
	workspaceID, _ := ctx.Value(projCtxKeyWorkspaceID).(string)
	allowedProjects, _ := ctx.Value(projCtxKeyAllowedProjs).([]string)
	wildcard, _ := ctx.Value(projCtxKeyWildcardAccess).(bool)

	return &ProjectContext{
		ProjectID:        projectID,
		ProjectSlug:      projectSlug,
		WorkspaceID:      workspaceID,
		AllowedProjects:  allowedProjects,
		IsWildcardAccess: wildcard,
	}
}

// ProjectMiddleware enforces project isolation at the HTTP middleware level.
// It extracts project information and ensures the request is properly scoped.
type ProjectMiddleware struct {
	store ProjectStore
}

// NewProjectMiddleware creates a new project middleware instance.
func NewProjectMiddleware(store ProjectStore) *ProjectMiddleware {
	return &ProjectMiddleware{store: store}
}

// RequireProject ensures a project is specified in the request.
func (pm *ProjectMiddleware) RequireProject(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pc, err := ExtractProjectFromRequest(r)
		if err != nil {
			http.Error(w, `{"error":{"message":"project or workspace required","code":400}}`, http.StatusBadRequest)
			return
		}

		if err := pc.Validate(); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":{"message":"%s","code":400}}`, err.Error()), http.StatusBadRequest)
			return
		}

		ctx := WithProjectContext(r.Context(), pc)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateProjectAccess ensures the user has access to the requested project.
func (pm *ProjectMiddleware) ValidateProjectAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pc := GetProjectContext(r.Context())
		if pc == nil {
			http.Error(w, `{"error":{"message":"project context not found","code":400}}`, http.StatusBadRequest)
			return
		}

		// Wildcard access bypasses project-specific checks
		if pc.IsWildcardAccess {
			next.ServeHTTP(w, r)
			return
		}

		// Validate project access
		if pc.ProjectID != "" && !pc.IsAllowed(pc.ProjectID) {
			http.Error(w, `{"error":{"message":"access denied to project","code":403}}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// InMemoryProjectStore is a simple in-memory implementation for testing.
type InMemoryProjectStore struct {
	projects        map[string]*Project
	userProjects    map[string][]string // userID -> []projectID
}

// NewInMemoryProjectStore creates a new in-memory project store.
func NewInMemoryProjectStore() *InMemoryProjectStore {
	return &InMemoryProjectStore{
		projects:     make(map[string]*Project),
		userProjects: make(map[string][]string),
	}
}

// GetProjectByID retrieves a project by ID.
func (s *InMemoryProjectStore) GetProjectByID(projectID string) (*Project, error) {
	if p, ok := s.projects[projectID]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("project not found: %s", projectID)
}

// GetProjectBySlug retrieves a project by slug.
func (s *InMemoryProjectStore) GetProjectBySlug(slug string) (*Project, error) {
	for _, p := range s.projects {
		if p.Slug == slug {
			return p, nil
		}
	}
	return nil, fmt.Errorf("project not found: %s", slug)
}

// GetUserProjects retrieves all projects a user has access to.
func (s *InMemoryProjectStore) GetUserProjects(userID string) ([]Project, error) {
	projectIDs, ok := s.userProjects[userID]
	if !ok {
		return []Project{}, nil
	}

	var projects []Project
	for _, id := range projectIDs {
		if p, ok := s.projects[id]; ok {
			projects = append(projects, *p)
		}
	}
	return projects, nil
}

// UserHasProjectAccess checks if a user has access to a specific project.
func (s *InMemoryProjectStore) UserHasProjectAccess(userID, projectID string) (bool, error) {
	projectIDs, ok := s.userProjects[userID]
	if !ok {
		return false, nil
	}
	for _, id := range projectIDs {
		if id == projectID {
			return true, nil
		}
	}
	return false, nil
}

// AddProject adds a project to the store.
func (s *InMemoryProjectStore) AddProject(p *Project) {
	s.projects[p.ID] = p
}

// GrantAccess grants a user access to a project.
func (s *InMemoryProjectStore) GrantAccess(userID, projectID string) {
	s.userProjects[userID] = append(s.userProjects[userID], projectID)
}
