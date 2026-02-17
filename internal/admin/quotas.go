package admin

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"radgateway/internal/logger"
)

// QuotaCreateRequest represents a request to create a quota
type QuotaCreateRequest struct {
	Name             string `json:"name"`
	WorkspaceID      string `json:"workspaceId"`
	Description      string `json:"description,omitempty"`
	QuotaType        string `json:"quotaType"` // requests, tokens, cost
	Period           string `json:"period"`    // minute, hour, day, week, month
	LimitValue       int64  `json:"limitValue"`
	Scope            string `json:"scope"` // global, workspace, apikey, user
	WarningThreshold int    `json:"warningThreshold"` // Percentage (0-100)
}

// QuotaUpdateRequest represents a request to update a quota
type QuotaUpdateRequest struct {
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	QuotaType        string `json:"quotaType,omitempty"`
	Period           string `json:"period,omitempty"`
	LimitValue       int64  `json:"limitValue,omitempty"`
	Scope            string `json:"scope,omitempty"`
	WarningThreshold int    `json:"warningThreshold,omitempty"`
	Status           string `json:"status,omitempty"`
}

// QuotaResponse represents a quota in API responses
type QuotaResponse struct {
	ID               string    `json:"id"`
	WorkspaceID      string    `json:"workspaceId"`
	Name             string    `json:"name"`
	Description      *string   `json:"description,omitempty"`
	QuotaType        string    `json:"quotaType"`
	Period           string    `json:"period"`
	LimitValue       int64     `json:"limitValue"`
	Scope            string    `json:"scope"`
	WarningThreshold int       `json:"warningThreshold"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// QuotaListResponse represents the list response
type QuotaListResponse struct {
	Data     []QuotaResponse `json:"data"`
	Total    int             `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
	HasMore  bool            `json:"hasMore"`
}

// QuotaAssignmentCreateRequest represents a request to create a quota assignment
type QuotaAssignmentCreateRequest struct {
	QuotaID      string `json:"quotaId"`
	ResourceType string `json:"resourceType"` // workspace, apikey, user
	ResourceID   string `json:"resourceId"`
}

// QuotaAssignmentResponse represents a quota assignment in API responses
type QuotaAssignmentResponse struct {
	QuotaID       string     `json:"quotaId"`
	ResourceType  string     `json:"resourceType"`
	ResourceID    string     `json:"resourceId"`
	CurrentUsage  int64      `json:"currentUsage"`
	PeriodStart   time.Time  `json:"periodStart"`
	PeriodEnd     time.Time  `json:"periodEnd"`
	LimitValue    int64      `json:"limitValue"`
	UsagePercent  float64    `json:"usagePercent"`
	WarningSent   bool       `json:"warningSent"`
	ExceededAt    *time.Time `json:"exceededAt,omitempty"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// QuotaAssignmentListResponse represents quota assignment list response
type QuotaAssignmentListResponse struct {
	Data     []QuotaAssignmentResponse `json:"data"`
	Total    int                         `json:"total"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"pageSize"`
	HasMore  bool                        `json:"hasMore"`
}

// QuotaUsageResponse represents current quota usage
type QuotaUsageResponse struct {
	QuotaID       string     `json:"quotaId"`
	QuotaName     string     `json:"quotaName"`
	ResourceType  string     `json:"resourceType"`
	ResourceID    string     `json:"resourceId"`
	LimitValue    int64      `json:"limitValue"`
	CurrentUsage  int64      `json:"currentUsage"`
	Remaining     int64      `json:"remaining"`
	UsagePercent  float64    `json:"usagePercent"`
	PeriodStart   time.Time  `json:"periodStart"`
	PeriodEnd     time.Time  `json:"periodEnd"`
	Status        string     `json:"status"` // ok, warning, exceeded
	WarningSent   bool       `json:"warningSent"`
	ExceededAt    *time.Time `json:"exceededAt,omitempty"`
}

// QuotaCheckRequest represents a request to check quota
type QuotaCheckRequest struct {
	QuotaID      string `json:"quotaId"`
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	Requested    int64  `json:"requested"` // Amount being requested
}

// QuotaCheckResponse represents quota check response
type QuotaCheckResponse struct {
	Allowed       bool   `json:"allowed"`
	QuotaID       string `json:"quotaId"`
	LimitValue    int64  `json:"limitValue"`
	CurrentUsage  int64  `json:"currentUsage"`
	Remaining     int64  `json:"remaining"`
	UsagePercent  float64 `json:"usagePercent"`
	Reason        string `json:"reason,omitempty"`
}

// BulkQuotaRequest represents a bulk operation request
type BulkQuotaRequest struct {
	IDs    []string `json:"ids"`
	Action string   `json:"action"`
}

// QuotaAlert represents a quota alert
type QuotaAlert struct {
	ID           string    `json:"id"`
	QuotaID      string    `json:"quotaId"`
	QuotaName    string    `json:"quotaName"`
	ResourceType string    `json:"resourceType"`
	ResourceID   string    `json:"resourceId"`
	AlertType    string    `json:"alertType"` // warning, exceeded
	Message      string    `json:"message"`
	CurrentUsage int64     `json:"currentUsage"`
	LimitValue   int64     `json:"limitValue"`
	TriggeredAt  time.Time `json:"triggeredAt"`
	Acknowledged bool      `json:"acknowledged"`
}

// QuotaHandler handles quota management endpoints
type QuotaHandler struct {
	log *slog.Logger
}

// NewQuotaHandler creates a new quota handler
func NewQuotaHandler() *QuotaHandler {
	return &QuotaHandler{
		log: logger.WithComponent("admin.quotas"),
	}
}

// RegisterRoutes registers the quota management routes
func (h *QuotaHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v0/admin/quotas", h.handleQuotas)
	mux.HandleFunc("/v0/admin/quotas/", h.handleQuotaDetail)
	mux.HandleFunc("/v0/admin/quotas/bulk", h.handleBulkOperation)
	mux.HandleFunc("/v0/admin/quotas/check", h.handleQuotaCheck)
	mux.HandleFunc("/v0/admin/quotas/assignments", h.handleAssignments)
	mux.HandleFunc("/v0/admin/quotas/assignments/", h.handleAssignmentDetail)
	mux.HandleFunc("/v0/admin/quotas/usage", h.handleQuotaUsage)
	mux.HandleFunc("/v0/admin/quotas/alerts", h.handleQuotaAlerts)
}

func (h *QuotaHandler) handleQuotas(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listQuotas(w, r)
	case http.MethodPost:
		h.createQuota(w, r)
	default:
		h.log.Warn("method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *QuotaHandler) handleQuotaDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v0/admin/quotas/")
	if id == "" || id == "bulk" || id == "check" || id == "assignments" || id == "usage" || id == "alerts" {
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getQuota(w, r, id)
	case http.MethodPut:
		h.updateQuota(w, r, id)
	case http.MethodDelete:
		h.deleteQuota(w, r, id)
	case http.MethodPatch:
		h.patchQuota(w, r, id)
	default:
		h.log.Warn("method not allowed", "path", r.URL.Path, "method", r.Method)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *QuotaHandler) handleBulkOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	h.handleBulkQuota(w, r)
}

func (h *QuotaHandler) handleQuotaCheck(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.checkQuota(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *QuotaHandler) handleAssignments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listAssignments(w, r)
	case http.MethodPost:
		h.createAssignment(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *QuotaHandler) handleAssignmentDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v0/admin/quotas/assignments/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "assignment id required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getAssignment(w, r, id)
	case http.MethodDelete:
		h.deleteAssignment(w, r, id)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (h *QuotaHandler) handleQuotaUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	h.getQuotaUsage(w, r)
}

func (h *QuotaHandler) handleQuotaAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	h.listQuotaAlerts(w, r)
}

// listQuotas returns a paginated list of quotas
func (h *QuotaHandler) listQuotas(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspaceId")
	quotaType := r.URL.Query().Get("quotaType")
	scope := r.URL.Query().Get("scope")
	status := r.URL.Query().Get("status")
	page := readIntParam(r, "page", 1)
	pageSize := readIntParam(r, "pageSize", 50)

	h.log.Debug("listing quotas",
		"workspace", workspaceID,
		"type", quotaType,
		"scope", scope,
		"status", status,
	)

	// Mock quotas
	quotas := []QuotaResponse{
		{
			ID:               "quota_001",
			WorkspaceID:      "ws_001",
			Name:             "Production Requests Limit",
			Description:      strPtr("Maximum requests per hour for production"),
			QuotaType:        "requests",
			Period:           "hour",
			LimitValue:       10000,
			Scope:            "workspace",
			WarningThreshold: 80,
			Status:           "active",
			CreatedAt:        time.Now().Add(-30 * 24 * time.Hour),
			UpdatedAt:        time.Now(),
		},
		{
			ID:               "quota_002",
			WorkspaceID:      "ws_001",
			Name:             "Production Tokens Limit",
			Description:      strPtr("Maximum tokens per day for production"),
			QuotaType:        "tokens",
			Period:           "day",
			LimitValue:       100000000,
			Scope:            "workspace",
			WarningThreshold: 75,
			Status:           "active",
			CreatedAt:        time.Now().Add(-30 * 24 * time.Hour),
			UpdatedAt:        time.Now(),
		},
		{
			ID:               "quota_003",
			WorkspaceID:      "ws_002",
			Name:             "Staging Requests Limit",
			QuotaType:        "requests",
			Period:           "hour",
			LimitValue:       1000,
			Scope:            "workspace",
			WarningThreshold: 90,
			Status:           "active",
			CreatedAt:        time.Now().Add(-20 * 24 * time.Hour),
			UpdatedAt:        time.Now(),
		},
	}

	// Apply filters
	var filtered []QuotaResponse
	for _, q := range quotas {
		if workspaceID != "" && q.WorkspaceID != workspaceID {
			continue
		}
		if quotaType != "" && q.QuotaType != quotaType {
			continue
		}
		if scope != "" && q.Scope != scope {
			continue
		}
		if status != "" && q.Status != status {
			continue
		}
		filtered = append(filtered, q)
	}

	total := len(filtered)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	response := QuotaListResponse{
		Data:     filtered[start:end],
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  end < total,
	}

	writeJSON(w, http.StatusOK, response)
}

// getQuota returns a single quota
func (h *QuotaHandler) getQuota(w http.ResponseWriter, r *http.Request, id string) {
	h.log.Debug("getting quota", "id", id)

	quota := QuotaResponse{
		ID:               id,
		WorkspaceID:      "ws_001",
		Name:             "Production Requests Limit",
		Description:      strPtr("Maximum requests per hour for production"),
		QuotaType:        "requests",
		Period:           "hour",
		LimitValue:       10000,
		Scope:            "workspace",
		WarningThreshold: 80,
		Status:           "active",
		CreatedAt:        time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt:        time.Now(),
	}

	writeJSON(w, http.StatusOK, quota)
}

// createQuota creates a new quota
func (h *QuotaHandler) createQuota(w http.ResponseWriter, r *http.Request) {
	var req QuotaCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if req.QuotaType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "quotaType is required"})
		return
	}
	if req.Period == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "period is required"})
		return
	}
	if req.LimitValue <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limitValue must be positive"})
		return
	}

	h.log.Info("creating quota",
		"name", req.Name,
		"type", req.QuotaType,
		"limit", req.LimitValue,
	)

	quota := QuotaResponse{
		ID:               generateID("quota"),
		WorkspaceID:      req.WorkspaceID,
		Name:             req.Name,
		Description:      &req.Description,
		QuotaType:        req.QuotaType,
		Period:           req.Period,
		LimitValue:       req.LimitValue,
		Scope:            req.Scope,
		WarningThreshold: req.WarningThreshold,
		Status:           "active",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	writeJSON(w, http.StatusCreated, quota)
}

// updateQuota fully updates a quota
func (h *QuotaHandler) updateQuota(w http.ResponseWriter, r *http.Request, id string) {
	var req QuotaUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	h.log.Info("updating quota", "id", id)

	quota := QuotaResponse{
		ID:               id,
		WorkspaceID:      "ws_001",
		Name:             req.Name,
		Description:      &req.Description,
		QuotaType:        req.QuotaType,
		Period:           req.Period,
		LimitValue:       req.LimitValue,
		Scope:            req.Scope,
		WarningThreshold: req.WarningThreshold,
		Status:           req.Status,
		UpdatedAt:        time.Now(),
	}

	writeJSON(w, http.StatusOK, quota)
}

// patchQuota partially updates a quota
func (h *QuotaHandler) patchQuota(w http.ResponseWriter, r *http.Request, id string) {
	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	h.log.Info("patching quota", "id", id, "fields", len(updates))

	quota := QuotaResponse{
		ID:               id,
		WorkspaceID:      "ws_001",
		Name:             "Production Requests Limit",
		QuotaType:        "requests",
		Period:           "hour",
		LimitValue:       10000,
		Scope:            "workspace",
		WarningThreshold: 80,
		Status:           "active",
		UpdatedAt:        time.Now(),
	}

	writeJSON(w, http.StatusOK, quota)
}

// deleteQuota deletes a quota
func (h *QuotaHandler) deleteQuota(w http.ResponseWriter, r *http.Request, id string) {
	h.log.Info("deleting quota", "id", id)
	writeJSON(w, http.StatusNoContent, nil)
}

// handleBulkQuota handles bulk quota operations
func (h *QuotaHandler) handleBulkQuota(w http.ResponseWriter, r *http.Request) {
	var req BulkQuotaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.IDs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ids array is required"})
		return
	}

	if req.Action == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action is required"})
		return
	}

	h.log.Info("bulk quota operation",
		"action", req.Action,
		"count", len(req.IDs),
	)

	validActions := map[string]bool{
		"activate":   true,
		"deactivate": true,
		"delete":     true,
	}
	if !validActions[req.Action] {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid action"})
		return
	}

	result := map[string]interface{}{
		"processed": len(req.IDs),
		"action":    req.Action,
		"success":   true,
	}

	writeJSON(w, http.StatusOK, result)
}

// checkQuota checks if a request would exceed quota
func (h *QuotaHandler) checkQuota(w http.ResponseWriter, r *http.Request) {
	var req QuotaCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	h.log.Debug("checking quota",
		"quotaId", req.QuotaID,
		"resourceType", req.ResourceType,
		"requested", req.Requested,
	)

	// Mock check - simulate a quota with 10000 limit and 8000 current usage
	limitValue := int64(10000)
	currentUsage := int64(8000)
	remaining := limitValue - currentUsage
	usagePercent := (float64(currentUsage) / float64(limitValue)) * 100

	allowed := req.Requested <= remaining
	reason := ""
	if !allowed {
		reason = "quota exceeded"
	}

	response := QuotaCheckResponse{
		Allowed:      allowed,
		QuotaID:      req.QuotaID,
		LimitValue:   limitValue,
		CurrentUsage: currentUsage,
		Remaining:    remaining,
		UsagePercent: usagePercent,
		Reason:       reason,
	}

	writeJSON(w, http.StatusOK, response)
}

// listAssignments returns quota assignments
func (h *QuotaHandler) listAssignments(w http.ResponseWriter, r *http.Request) {
	quotaID := r.URL.Query().Get("quotaId")
	resourceType := r.URL.Query().Get("resourceType")
	resourceID := r.URL.Query().Get("resourceId")
	page := readIntParam(r, "page", 1)
	pageSize := readIntParam(r, "pageSize", 50)

	h.log.Debug("listing quota assignments", "quota", quotaID)

	// Mock assignments
	assignments := []QuotaAssignmentResponse{
		{
			QuotaID:      "quota_001",
			ResourceType: "workspace",
			ResourceID:   "ws_001",
			CurrentUsage: 7234,
			PeriodStart:  time.Now().Truncate(time.Hour),
			PeriodEnd:    time.Now().Truncate(time.Hour).Add(time.Hour),
			LimitValue:   10000,
			UsagePercent: 72.34,
			WarningSent:  false,
			UpdatedAt:    time.Now(),
		},
		{
			QuotaID:      "quota_002",
			ResourceType: "workspace",
			ResourceID:   "ws_001",
			CurrentUsage: 82345000,
			PeriodStart:  time.Now().Truncate(24 * time.Hour),
			PeriodEnd:    time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour),
			LimitValue:   100000000,
			UsagePercent: 82.35,
			WarningSent:  true,
			UpdatedAt:    time.Now(),
		},
	}

	// Apply filters
	var filtered []QuotaAssignmentResponse
	for _, a := range assignments {
		if quotaID != "" && a.QuotaID != quotaID {
			continue
		}
		if resourceType != "" && a.ResourceType != resourceType {
			continue
		}
		if resourceID != "" && a.ResourceID != resourceID {
			continue
		}
		filtered = append(filtered, a)
	}

	total := len(filtered)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	response := QuotaAssignmentListResponse{
		Data:     filtered[start:end],
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		HasMore:  end < total,
	}

	writeJSON(w, http.StatusOK, response)
}

// getAssignment returns a single quota assignment
func (h *QuotaHandler) getAssignment(w http.ResponseWriter, r *http.Request, id string) {
	h.log.Debug("getting quota assignment", "id", id)

	// Parse quotaID:resourceType:resourceID from id
	assignment := QuotaAssignmentResponse{
		QuotaID:      "quota_001",
		ResourceType: "workspace",
		ResourceID:   "ws_001",
		CurrentUsage: 7234,
		PeriodStart:  time.Now().Truncate(time.Hour),
		PeriodEnd:    time.Now().Truncate(time.Hour).Add(time.Hour),
		LimitValue:   10000,
		UsagePercent: 72.34,
		WarningSent:  false,
		UpdatedAt:    time.Now(),
	}

	writeJSON(w, http.StatusOK, assignment)
}

// createAssignment creates a new quota assignment
func (h *QuotaHandler) createAssignment(w http.ResponseWriter, r *http.Request) {
	var req QuotaAssignmentCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("invalid request body", "error", err.Error())
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.QuotaID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "quotaId is required"})
		return
	}
	if req.ResourceType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "resourceType is required"})
		return
	}
	if req.ResourceID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "resourceId is required"})
		return
	}

	h.log.Info("creating quota assignment",
		"quotaId", req.QuotaID,
		"resourceType", req.ResourceType,
		"resourceId", req.ResourceID,
	)

	// Get quota limit (mock)
	limitValue := int64(10000)

	assignment := QuotaAssignmentResponse{
		QuotaID:      req.QuotaID,
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceID,
		CurrentUsage: 0,
		PeriodStart:  time.Now().Truncate(time.Hour),
		PeriodEnd:    time.Now().Truncate(time.Hour).Add(time.Hour),
		LimitValue:   limitValue,
		UsagePercent: 0,
		WarningSent:  false,
		UpdatedAt:    time.Now(),
	}

	writeJSON(w, http.StatusCreated, assignment)
}

// deleteAssignment deletes a quota assignment
func (h *QuotaHandler) deleteAssignment(w http.ResponseWriter, r *http.Request, id string) {
	h.log.Info("deleting quota assignment", "id", id)
	writeJSON(w, http.StatusNoContent, nil)
}

// getQuotaUsage returns current quota usage
func (h *QuotaHandler) getQuotaUsage(w http.ResponseWriter, r *http.Request) {
	quotaID := r.URL.Query().Get("quotaId")
	resourceType := r.URL.Query().Get("resourceType")
	resourceID := r.URL.Query().Get("resourceId")

	h.log.Debug("getting quota usage",
		"quotaId", quotaID,
		"resourceType", resourceType,
		"resourceId", resourceID,
	)

	// Mock usage data
	usages := []QuotaUsageResponse{
		{
			QuotaID:       "quota_001",
			QuotaName:     "Production Requests Limit",
			ResourceType:  "workspace",
			ResourceID:    "ws_001",
			LimitValue:    10000,
			CurrentUsage:  7234,
			Remaining:     2766,
			UsagePercent:  72.34,
			PeriodStart:   time.Now().Truncate(time.Hour),
			PeriodEnd:     time.Now().Truncate(time.Hour).Add(time.Hour),
			Status:        "ok",
			WarningSent:   false,
		},
		{
			QuotaID:       "quota_002",
			QuotaName:     "Production Tokens Limit",
			ResourceType:  "workspace",
			ResourceID:    "ws_001",
			LimitValue:    100000000,
			CurrentUsage:  82345000,
			Remaining:     17655000,
			UsagePercent:  82.35,
			PeriodStart:   time.Now().Truncate(24 * time.Hour),
			PeriodEnd:     time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour),
			Status:        "warning",
			WarningSent:   true,
		},
	}

	// Filter by query params
	var filtered []QuotaUsageResponse
	for _, u := range usages {
		if quotaID != "" && u.QuotaID != quotaID {
			continue
		}
		if resourceType != "" && u.ResourceType != resourceType {
			continue
		}
		if resourceID != "" && u.ResourceID != resourceID {
			continue
		}
		filtered = append(filtered, u)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": filtered,
	})
}

// listQuotaAlerts returns quota alerts
func (h *QuotaHandler) listQuotaAlerts(w http.ResponseWriter, r *http.Request) {
	page := readIntParam(r, "page", 1)
	pageSize := readIntParam(r, "pageSize", 50)

	h.log.Debug("listing quota alerts")

	// Mock alerts
	alerts := []QuotaAlert{
		{
			ID:           "qalert_001",
			QuotaID:      "quota_002",
			QuotaName:    "Production Tokens Limit",
			ResourceType: "workspace",
			ResourceID:   "ws_001",
			AlertType:    "warning",
			Message:      "Token quota usage exceeded 75% threshold",
			CurrentUsage: 82345000,
			LimitValue:   100000000,
			TriggeredAt:  time.Now().Add(-2 * time.Hour),
			Acknowledged: false,
		},
	}

	total := len(alerts)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	response := map[string]interface{}{
		"data":     alerts[start:end],
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
		"hasMore":  end < total,
	}

	writeJSON(w, http.StatusOK, response)
}
