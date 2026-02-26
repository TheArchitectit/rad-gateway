// Package cedar provides Cedar policy evaluation for authorization
package cedar

import (
	"context"
	"fmt"
	"os"

	"github.com/cedar-policy/cedar-go"
	"github.com/cedar-policy/cedar-go/types"
)

// PolicyDecisionPoint evaluates authorization requests against Cedar policies
type PolicyDecisionPoint struct {
	policySet *cedar.PolicySet
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

// NewPDP creates a new policy decision point from policy files
func NewPDP(policyPath string) (*PolicyDecisionPoint, error) {
	// Read policy file
	policyBytes, err := os.ReadFile(policyPath)
	if err != nil {
		return nil, fmt.Errorf("reading policy file: %w", err)
	}

	// Parse policies using NewPolicySetFromBytes
	policySet, err := cedar.NewPolicySetFromBytes(policyPath, policyBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing policies: %w", err)
	}

	return &PolicyDecisionPoint{
		policySet: policySet,
	}, nil
}

// Authorize evaluates an authorization request against policies
func (p *PolicyDecisionPoint) Authorize(
	ctx context.Context,
	req AuthorizationRequest,
) (*AuthorizationDecision, error) {
	// Convert to Cedar entities using NewEntityUID helper
	principal := types.NewEntityUID(types.EntityType("A2A::Agent"), types.String(req.Principal))
	action := types.NewEntityUID(types.EntityType("A2A::Action"), types.String(req.Action))
	resource := types.NewEntityUID(types.EntityType("A2A::Task"), types.String(req.Resource))

	// Build entities - keys are EntityUID
	entities := types.EntityMap{
		principal: types.Entity{
			UID:        principal,
			Attributes: types.Record{},
		},
		action: types.Entity{
			UID:        action,
			Attributes: types.Record{},
		},
		resource: types.Entity{
			UID:        resource,
			Attributes: types.Record{},
		},
	}

	// Build request - EntityUID not pointer
	cedarReq := types.Request{
		Principal: principal,
		Action:    action,
		Resource:  resource,
	}

	// Evaluate the policy using Authorize function
	decision, diagnostic := cedar.Authorize(p.policySet, entities, cedarReq)

	result := "Deny"
	if decision == cedar.Allow {
		result = "Allow"
	}

	// Extract reasons if available
	var reasons []string
	if len(diagnostic.Reasons) > 0 {
		for _, r := range diagnostic.Reasons {
			reasons = append(reasons, string(r.PolicyID))
		}
	}

	return &AuthorizationDecision{
		Decision: result,
		Reasons:  reasons,
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