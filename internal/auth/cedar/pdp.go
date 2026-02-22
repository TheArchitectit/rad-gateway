// Package cedar provides Cedar policy evaluation for authorization
package cedar

import (
	"context"
	"fmt"
	"os"

	"github.com/cedar-policy/cedar-go"
)

// PolicyDecisionPoint evaluates authorization requests against Cedar policies
type PolicyDecisionPoint struct {
	policySet *cedar.PolicySet
	schema    *cedar.Schema
}

// NewPDP creates a new policy decision point from policy files
func NewPDP(policyPath string) (*PolicyDecisionPoint, error) {
	// Read policy file
	policyBytes, err := os.ReadFile(policyPath)
	if err != nil {
		return nil, fmt.Errorf("reading policy file: %w", err)
	}

	// Parse policies
	policySet, err := cedar.ParsePolicies(string(policyBytes))
	if err != nil {
		return nil, fmt.Errorf("parsing policies: %w", err)
	}

	return &PolicyDecisionPoint{
		policySet: policySet,
	}, nil
}

// AuthorizationRequest represents a request to authorize
type AuthorizationRequest struct {
	Principal string         `json:"principal"`
	Action    string         `json:"action"`
	Resource  string         `json:"resource"`
	Context   map[string]any `json:"context,omitempty"`
}

// AuthorizationDecision represents the authorization result
type AuthorizationDecision struct {
	Decision string   `json:"decision"` // "Allow" or "Deny"
	Reasons  []string `json:"reasons,omitempty"`
}

// Authorize evaluates an authorization request against policies
func (p *PolicyDecisionPoint) Authorize(
	ctx context.Context,
	req AuthorizationRequest,
) (*AuthorizationDecision, error) {
	// Convert to Cedar entities
	principal := cedar.EntityUID{
		Type: "A2A::Agent",
		ID:   cedar.String(req.Principal),
	}

	action := cedar.EntityUID{
		Type: "A2A::Action",
		ID:   cedar.String(req.Action),
	}

	resource := cedar.EntityUID{
		Type: "A2A::Task",
		ID:   cedar.String(req.Resource),
	}

	// Build context from request
	context := cedar.NewRecord(cedar.RecordMap{})
	for k, v := range req.Context {
		context.Set(cedar.String(k), cedar.String(fmt.Sprintf("%v", v)))
	}

	// Evaluate the policy
	result, err := p.policySet.IsAuthorized(
		principal,
		action,
		resource,
		[]cedar.Context{context},
	)
	if err != nil {
		return nil, fmt.Errorf("policy evaluation error: %w", err)
	}

	decision := "Deny"
	if result.Decision == cedar.Allow {
		decision = "Allow"
	}

	return &AuthorizationDecision{
		Decision: decision,
		Reasons:  result.Reasons,
	}, nil
}

// IsAuthorized checks if a principal can perform an action on a resource
func (p *PolicyDecisionPoint) IsAuthorized(
	principalID string,
	action string,
	resourceID string,
) (bool, error) {
	ctx := context.Background()
	req := AuthorizationRequest{
		Principal: principalID,
		Action:    action,
		Resource:  resourceID,
	}

	result, err := p.Authorize(ctx, req)
	if err != nil {
		return false, err
	}

	return result.Decision == "Allow", nil
}
