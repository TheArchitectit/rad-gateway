package rbac

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProjectContextIsAllowed(t *testing.T) {
	tests := []struct {
		name      string
		ctx       *ProjectContext
		projectID string
		want      bool
	}{
		{
			name: "allowed project",
			ctx: &ProjectContext{
				ProjectID:       "proj-1",
				AllowedProjects: []string{"proj-1", "proj-2"},
				WorkspaceID:     "ws-1",
			},
			projectID: "proj-1",
			want:      true,
		},
		{
			name: "not allowed project",
			ctx: &ProjectContext{
				ProjectID:       "proj-1",
				AllowedProjects: []string{"proj-2"},
				WorkspaceID:     "ws-1",
			},
			projectID: "proj-3",
			want:      false,
		},
		{
			name: "wildcard allows all",
			ctx: &ProjectContext{
				IsWildcardAccess: true,
				WorkspaceID:      "ws-1",
			},
			projectID: "any-project",
			want:      true,
		},
		{
			name:      "nil context",
			ctx:       nil,
			projectID: "proj-1",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ctx.IsAllowed(tt.projectID)
			if got != tt.want {
				t.Errorf("IsAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjectContextIsCurrentProject(t *testing.T) {
	tests := []struct {
		name      string
		ctx       *ProjectContext
		projectID string
		want      bool
	}{
		{
			name: "current project matches",
			ctx: &ProjectContext{
				ProjectID:   "proj-1",
				WorkspaceID: "ws-1",
			},
			projectID: "proj-1",
			want:      true,
		},
		{
			name: "current project does not match",
			ctx: &ProjectContext{
				ProjectID:   "proj-1",
				WorkspaceID: "ws-1",
			},
			projectID: "proj-2",
			want:      false,
		},
		{
			name:      "nil context",
			ctx:       nil,
			projectID: "proj-1",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ctx.IsCurrentProject(tt.projectID)
			if got != tt.want {
				t.Errorf("IsCurrentProject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjectContextValidate(t *testing.T) {
	tests := []struct {
		name    string
		ctx     *ProjectContext
		wantErr bool
	}{
		{
			name: "valid context",
			ctx: &ProjectContext{
				ProjectID:   "proj-1",
				WorkspaceID: "ws-1",
			},
			wantErr: false,
		},
		{
			name: "wildcard without project ID is valid",
			ctx: &ProjectContext{
				IsWildcardAccess: true,
				WorkspaceID:      "ws-1",
			},
			wantErr: false,
		},
		{
			name:    "nil context",
			ctx:     nil,
			wantErr: true,
		},
		{
			name: "missing project ID without wildcard",
			ctx: &ProjectContext{
				ProjectID:   "",
				WorkspaceID: "ws-1",
			},
			wantErr: true,
		},
		{
			name: "missing workspace ID",
			ctx: &ProjectContext{
				ProjectID:   "proj-1",
				WorkspaceID: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ctx.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWithProjectContext(t *testing.T) {
	ctx := context.Background()
	pc := &ProjectContext{
		ProjectID:        "proj-1",
		ProjectSlug:      "my-project",
		WorkspaceID:      "ws-1",
		AllowedProjects:  []string{"proj-1", "proj-2"},
		IsWildcardAccess: false,
	}

	newCtx := WithProjectContext(ctx, pc)
	retrieved := GetProjectContext(newCtx)

	if retrieved == nil {
		t.Fatal("GetProjectContext returned nil")
	}

	if retrieved.ProjectID != pc.ProjectID {
		t.Errorf("ProjectID = %q, want %q", retrieved.ProjectID, pc.ProjectID)
	}
	if retrieved.ProjectSlug != pc.ProjectSlug {
		t.Errorf("ProjectSlug = %q, want %q", retrieved.ProjectSlug, pc.ProjectSlug)
	}
	if retrieved.WorkspaceID != pc.WorkspaceID {
		t.Errorf("WorkspaceID = %q, want %q", retrieved.WorkspaceID, pc.WorkspaceID)
	}
	if retrieved.IsWildcardAccess != pc.IsWildcardAccess {
		t.Errorf("IsWildcardAccess = %v, want %v", retrieved.IsWildcardAccess, pc.IsWildcardAccess)
	}
}

func TestWithProjectContextNil(t *testing.T) {
	ctx := context.Background()
	newCtx := WithProjectContext(ctx, nil)

	if newCtx != ctx {
		t.Error("WithProjectContext(nil) should return original context")
	}
}

func TestGetProjectContextNil(t *testing.T) {
	result := GetProjectContext(nil)
	if result != nil {
		t.Error("GetProjectContext(nil) should return nil")
	}
}

func TestExtractProjectFromRequest(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*http.Request)
		wantErr   bool
		projectID string
		workspace string
	}{
		{
			name: "header project ID",
			setup: func(r *http.Request) {
				r.Header.Set(ProjectHeaderKey, "proj-1")
				r.Header.Set(WorkspaceHeaderKey, "ws-1")
			},
			wantErr:   false,
			projectID: "proj-1",
			workspace: "ws-1",
		},
		{
			name: "query parameter project ID",
			setup: func(r *http.Request) {
				q := r.URL.Query()
				q.Set("project_id", "proj-2")
				r.URL.RawQuery = q.Encode()
				r.Header.Set(WorkspaceHeaderKey, "ws-1")
			},
			wantErr:   false,
			projectID: "proj-2",
			workspace: "ws-1",
		},
		{
			name: "missing both project and workspace",
			setup: func(r *http.Request) {
				// No headers or query params
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			tt.setup(req)

			pc, err := ExtractProjectFromRequest(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractProjectFromRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && pc != nil {
				if pc.ProjectID != tt.projectID {
					t.Errorf("ProjectID = %q, want %q", pc.ProjectID, tt.projectID)
				}
				if pc.WorkspaceID != tt.workspace {
					t.Errorf("WorkspaceID = %q, want %q", pc.WorkspaceID, tt.workspace)
				}
			}
		})
	}
}

func TestInMemoryProjectStore(t *testing.T) {
	store := NewInMemoryProjectStore()

	// Add projects
	store.AddProject(&Project{
		ID:          "proj-1",
		Slug:        "project-one",
		Name:        "Project One",
		WorkspaceID: "ws-1",
	})
	store.AddProject(&Project{
		ID:          "proj-2",
		Slug:        "project-two",
		Name:        "Project Two",
		WorkspaceID: "ws-1",
	})

	// Grant access
	store.GrantAccess("user-1", "proj-1")
	store.GrantAccess("user-1", "proj-2")

	t.Run("GetProjectByID", func(t *testing.T) {
		p, err := store.GetProjectByID("proj-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.ID != "proj-1" {
			t.Errorf("ID = %q, want %q", p.ID, "proj-1")
		}
	})

	t.Run("GetProjectByID not found", func(t *testing.T) {
		_, err := store.GetProjectByID("non-existent")
		if err == nil {
			t.Error("expected error for non-existent project")
		}
	})

	t.Run("GetProjectBySlug", func(t *testing.T) {
		p, err := store.GetProjectBySlug("project-one")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Slug != "project-one" {
			t.Errorf("Slug = %q, want %q", p.Slug, "project-one")
		}
	})

	t.Run("GetUserProjects", func(t *testing.T) {
		projects, err := store.GetUserProjects("user-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(projects) != 2 {
			t.Errorf("len(projects) = %d, want %d", len(projects), 2)
		}
	})

	t.Run("GetUserProjects no access", func(t *testing.T) {
		projects, err := store.GetUserProjects("user-2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(projects) != 0 {
			t.Errorf("len(projects) = %d, want %d", len(projects), 0)
		}
	})

	t.Run("UserHasProjectAccess true", func(t *testing.T) {
		has, err := store.UserHasProjectAccess("user-1", "proj-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !has {
			t.Error("expected user to have access to proj-1")
		}
	})

	t.Run("UserHasProjectAccess false", func(t *testing.T) {
		has, err := store.UserHasProjectAccess("user-1", "proj-3")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if has {
			t.Error("expected user to NOT have access to proj-3")
		}
	})

	t.Run("UserHasProjectAccess unknown user", func(t *testing.T) {
		has, err := store.UserHasProjectAccess("unknown", "proj-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if has {
			t.Error("expected unknown user to NOT have access")
		}
	})
}

func TestProjectMiddleware(t *testing.T) {
	store := NewInMemoryProjectStore()
	store.AddProject(&Project{ID: "proj-1", Slug: "p1", WorkspaceID: "ws-1"})
	store.GrantAccess("user-1", "proj-1")

	pm := NewProjectMiddleware(store)

	t.Run("RequireProject success", func(t *testing.T) {
		handler := pm.RequireProject(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set(ProjectHeaderKey, "proj-1")
		req.Header.Set(WorkspaceHeaderKey, "ws-1")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
		}
	})

	t.Run("RequireProject missing project", func(t *testing.T) {
		handler := pm.RequireProject(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
		}
	})

	t.Run("ValidateProjectAccess wildcard access", func(t *testing.T) {
		handler := pm.ValidateProjectAccess(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		ctx := context.Background()
		pc := &ProjectContext{
			WorkspaceID:      "ws-1",
			IsWildcardAccess: true,
		}
		ctx = WithProjectContext(ctx, pc)

		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
		}
	})
}
