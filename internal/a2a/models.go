// Package a2a provides A2A (Agent-to-Agent) protocol support for RAD Gateway.
package a2a

import (
	"encoding/json"
	"time"
)

// ============================================================================
// Database Model Card Types (for a2a_model_cards table)
// ============================================================================

// ModelCardStatus represents the status of a model card.
type ModelCardStatus string

const (
	// ModelCardStatusActive indicates the model card is active and usable.
	ModelCardStatusActive ModelCardStatus = "active"
	// ModelCardStatusDeprecated indicates the model card is deprecated.
	ModelCardStatusDeprecated ModelCardStatus = "deprecated"
	// ModelCardStatusArchived indicates the model card is archived.
	ModelCardStatusArchived ModelCardStatus = "archived"
)

// ModelCard represents an A2A Model Card stored in the database.
// This matches the a2a_model_cards table schema.
type ModelCard struct {
	// ID is the unique identifier (UUID).
	ID string `db:"id" json:"id"`
	// WorkspaceID is the owning workspace.
	WorkspaceID string `db:"workspace_id" json:"workspaceId"`
	// UserID is the optional owning user.
	UserID *string `db:"user_id" json:"userId,omitempty"`
	// Name is the display name of the model card.
	Name string `db:"name" json:"name"`
	// Slug is the URL-friendly identifier.
	Slug string `db:"slug" json:"slug"`
	// Description is an optional description.
	Description *string `db:"description" json:"description,omitempty"`
	// Card is the JSONB A2A model card document.
	Card json.RawMessage `db:"card" json:"card"`
	// Version is the schema version.
	Version int `db:"version" json:"version"`
	// Status is the current status.
	Status ModelCardStatus `db:"status" json:"status"`
	// CreatedAt is when the record was created.
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	// UpdatedAt is when the record was last updated.
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// ModelCardList represents a list of model cards with pagination.
type ModelCardList struct {
	// Items is the list of model cards.
	Items []ModelCard `json:"items"`
	// Total is the total count (for pagination).
	Total int `json:"total"`
	// Limit is the page size.
	Limit int `json:"limit"`
	// Offset is the page offset.
	Offset int `json:"offset"`
}

// IsValidStatus checks if a status string is valid.
func IsValidStatus(status string) bool {
	switch ModelCardStatus(status) {
	case ModelCardStatusActive, ModelCardStatusDeprecated, ModelCardStatusArchived:
		return true
	}
	return false
}

// ============================================================================
// A2A Protocol Types
// ============================================================================

// AgentCard represents an agent's capabilities and metadata per A2A protocol.
type AgentCard struct {
	// Name is the display name of the agent.
	Name string `json:"name"`
	// Description describes what the agent does.
	Description string `json:"description"`
	// URL is the endpoint URL for the agent.
	URL string `json:"url"`
	// Version is the agent version.
	Version string `json:"version"`
	// Capabilities describes what the agent can do.
	Capabilities Capabilities `json:"capabilities"`
	// Skills lists the capabilities offered by the agent.
	Skills []Skill `json:"skills"`
	// Authentication describes supported authentication schemes.
	Authentication AuthInfo `json:"authentication"`
	// CreatedAt is when the agent card was created.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt is when the agent card was last updated.
	UpdatedAt time.Time `json:"updatedAt"`
}

// AuthInfo describes authentication schemes.
type AuthInfo struct {
	Schemes []string `json:"schemes"`
}

// Capabilities describes what an agent can do.
type Capabilities struct {
	// Streaming indicates if the agent supports streaming responses.
	Streaming bool `json:"streaming"`
	// PushNotifications indicates if the agent supports push notifications.
	PushNotifications bool `json:"pushNotifications"`
	// StateTransitionHistory indicates if the agent tracks state transitions.
	StateTransitionHistory bool `json:"stateTransitionHistory"`
}

// Skill represents a capability offered by an agent.
type Skill struct {
	// ID is the unique identifier for the skill.
	ID string `json:"id"`
	// Name is the display name of the skill.
	Name string `json:"name"`
	// Description describes what the skill does.
	Description string `json:"description"`
	// Tags are labels for categorizing the skill.
	Tags []string `json:"tags,omitempty"`
	// Examples are sample inputs or use cases.
	Examples []string `json:"examples,omitempty"`
	// Input is the input schema for the skill.
	Input *SkillSchema `json:"input,omitempty"`
	// Output is the output schema for the skill.
	Output *SkillSchema `json:"output,omitempty"`
}

// SkillSchema defines the input/output structure for a skill.
type SkillSchema struct {
	// Type is the data type (e.g., "object", "string").
	Type string `json:"type"`
	// Properties are the schema properties for object types.
	Properties map[string]interface{} `json:"properties,omitempty"`
	// Required lists the required property names.
	Required []string `json:"required,omitempty"`
}

// A2ACard represents the A2A protocol model card structure.
// This is stored in the Card JSONB field.
type A2ACard struct {
	// SchemaVersion is the A2A schema version.
	SchemaVersion string `json:"schemaVersion,omitempty"`
	// Name is the model name.
	Name string `json:"name,omitempty"`
	// Description describes the model.
	Description string `json:"description,omitempty"`
	// Capabilities lists what the model can do.
	Capabilities []ModelCapability `json:"capabilities,omitempty"`
	// InputSchema defines the expected input format.
	InputSchema *SchemaDefinition `json:"inputSchema,omitempty"`
	// OutputSchema defines the output format.
	OutputSchema *SchemaDefinition `json:"outputSchema,omitempty"`
	// Pricing information.
	Pricing *ModelPricing `json:"pricing,omitempty"`
	// Metadata for extensions.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ModelCapability represents a single model capability.
type ModelCapability struct {
	// Type is the capability type (e.g., "vision", "code", "streaming").
	Type string `json:"type"`
	// Name is the human-readable name.
	Name string `json:"name,omitempty"`
	// Description describes the capability.
	Description string `json:"description,omitempty"`
	// Enabled indicates if the capability is available.
	Enabled bool `json:"enabled,omitempty"`
	// Config contains capability-specific configuration.
	Config map[string]interface{} `json:"config,omitempty"`
}

// SchemaDefinition defines input/output schemas.
type SchemaDefinition struct {
	// Type is the schema type (e.g., "json", "text").
	Type string `json:"type,omitempty"`
	// Schema is the JSON Schema or similar.
	Schema map[string]interface{} `json:"schema,omitempty"`
	// ContentTypes lists supported content types.
	ContentTypes []string `json:"contentTypes,omitempty"`
}

// ModelPricing contains pricing information.
type ModelPricing struct {
	// InputPricePerToken is the cost per input token.
	InputPricePerToken float64 `json:"inputPricePerToken,omitempty"`
	// OutputPricePerToken is the cost per output token.
	OutputPricePerToken float64 `json:"outputPricePerToken,omitempty"`
	// Currency is the pricing currency (default: USD).
	Currency string `json:"currency,omitempty"`
}

// CreateModelCardRequest represents a request to create a model card.
type CreateModelCardRequest struct {
	WorkspaceID string          `json:"workspaceId"`
	UserID      *string         `json:"userId,omitempty"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description *string         `json:"description,omitempty"`
	Card        json.RawMessage `json:"card"`
}

// UpdateModelCardRequest represents a request to update a model card.
type UpdateModelCardRequest struct {
	Name        *string          `json:"name,omitempty"`
	Description *string          `json:"description,omitempty"`
	Card        *json.RawMessage `json:"card,omitempty"`
	Status      *ModelCardStatus `json:"status,omitempty"`
}

// ParseA2ACard parses the Card JSONB field into an A2ACard struct.
func (m *ModelCard) ParseA2ACard() (*A2ACard, error) {
	if len(m.Card) == 0 {
		return &A2ACard{}, nil
	}
	var card A2ACard
	if err := json.Unmarshal(m.Card, &card); err != nil {
		return nil, err
	}
	return &card, nil
}

// SetA2ACard sets the Card field from an A2ACard struct.
func (m *ModelCard) SetA2ACard(card *A2ACard) error {
	data, err := json.Marshal(card)
	if err != nil {
		return err
	}
	m.Card = data
	return nil
}
