// Package controlroom provides the Control Room tagging system for RAD Gateway.
// Control rooms enable customizable operational views with tag-based filtering.
package controlroom

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Tag represents a hierarchical tag in the format category:value.
// Tags are used to categorize resources like providers, models, API keys, etc.
type Tag struct {
	Category string `json:"category" db:"category"` // env, team, project, cost-center, etc.
	Value    string `json:"value" db:"value"`       // production, platform, customer-a, etc.
}

// String returns the tag in "category:value" format.
func (t Tag) String() string {
	return fmt.Sprintf("%s:%s", t.Category, t.Value)
}

// Equals checks if two tags are equal (case-sensitive for values).
func (t Tag) Equals(other Tag) bool {
	return t.Category == other.Category && t.Value == other.Value
}

// MatchesWildcard checks if the tag value matches a wildcard pattern.
// Supports * for zero or more characters and ? for a single character.
func (t Tag) MatchesWildcard(pattern string) bool {
	// Convert glob-style wildcard to a simple match
	// pattern can be: exact-value, prefix-*, *-suffix, *contains*, etc.
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") && !strings.Contains(pattern[:len(pattern)-1], "*") {
		// Prefix match: value*
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(t.Value, prefix)
	}
	if strings.HasPrefix(pattern, "*") && !strings.Contains(pattern[1:], "*") {
		// Suffix match: *value
		suffix := pattern[1:]
		return strings.HasSuffix(t.Value, suffix)
	}
	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") && len(pattern) > 2 {
		// Contains match: *value*
		substr := pattern[1 : len(pattern)-1]
		return strings.Contains(t.Value, substr)
	}
	// Exact match
	return t.Value == pattern
}

// ParseTag parses a tag from a "category:value" string.
func ParseTag(s string) (Tag, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return Tag{}, fmt.Errorf("invalid tag format: %q (expected category:value)", s)
	}
	return Tag{
		Category: strings.TrimSpace(parts[0]),
		Value:    strings.TrimSpace(parts[1]),
	}, nil
}

// MustParseTag parses a tag from a string, panicking on error (use with caution).
func MustParseTag(s string) Tag {
	tag, err := ParseTag(s)
	if err != nil {
		panic(err)
	}
	return tag
}

// IsValidCategory checks if a category name is valid.
// Categories must be alphanumeric with optional hyphens/underscores.
func IsValidCategory(category string) bool {
	if category == "" {
		return false
	}
	for _, r := range category {
		if !isValidCategoryChar(r) {
			return false
		}
	}
	return true
}

// IsValidValue checks if a tag value is valid.
// Values can contain most characters except control characters.
func IsValidValue(value string) bool {
	if value == "" {
		return false
	}
	// Disallow control characters
	for _, r := range value {
		if r < 32 {
			return false
		}
	}
	return true
}

func isValidCategoryChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '-' || r == '_'
}

// StandardCategories defines commonly used tag categories.
// These are recommendations, not restrictions - custom categories are allowed.
const (
	CategoryEnv        = "env"
	CategoryTeam       = "team"
	CategoryProject    = "project"
	CategoryCostCenter = "cost-center"
	CategoryRegion     = "region"
	CategoryProvider   = "provider"
	CategoryTier       = "tier"
	CategoryCompliance = "compliance"
	CategoryPriority   = "priority"
)

// Taggable is the interface for resources that can be tagged.
// Resources implementing this interface can be filtered by control rooms.
type Taggable interface {
	// GetTags returns all tags associated with the resource.
	GetTags() []Tag
	// SetTags replaces all tags on the resource.
	SetTags(tags []Tag)
	// HasTag checks if the resource has a specific tag.
	HasTag(tag Tag) bool
	// AddTag adds a tag to the resource if not already present.
	AddTag(tag Tag)
	// RemoveTag removes a tag from the resource.
	RemoveTag(tag Tag)
}

// TaggableResource is a base implementation of the Taggable interface.
// Embed this struct into resources that need tagging support.
type TaggableResource struct {
	tags []Tag
}

// GetTags returns all tags on the resource.
func (t *TaggableResource) GetTags() []Tag {
	return append([]Tag{}, t.tags...) // Return a copy
}

// SetTags replaces all tags on the resource.
func (t *TaggableResource) SetTags(tags []Tag) {
	t.tags = make([]Tag, len(tags))
	copy(t.tags, tags)
}

// HasTag checks if the resource has a specific tag.
func (t *TaggableResource) HasTag(tag Tag) bool {
	for _, existing := range t.tags {
		if existing.Equals(tag) {
			return true
		}
	}
	return false
}

// AddTag adds a tag if not already present.
func (t *TaggableResource) AddTag(tag Tag) {
	if !t.HasTag(tag) {
		t.tags = append(t.tags, tag)
	}
}

// RemoveTag removes a tag from the resource.
func (t *TaggableResource) RemoveTag(tag Tag) {
	for i, existing := range t.tags {
		if existing.Equals(tag) {
			// Remove by swapping with last and truncating
			t.tags[i] = t.tags[len(t.tags)-1]
			t.tags = t.tags[:len(t.tags)-1]
			return
		}
	}
}

// TaggedResource is a resource reference with its tags.
// Used for matching resources against control room filters.
type TaggedResource struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // provider, model, apikey, etc.
	Tags     []Tag  `json:"tags"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// HasCategory checks if the resource has any tag with the given category.
func (r TaggedResource) HasCategory(category string) bool {
	for _, tag := range r.Tags {
		if tag.Category == category {
			return true
		}
	}
	return false
}

// GetTagValue returns the value for a specific category, or empty string if not found.
// If multiple tags have the same category, returns the first one.
func (r TaggedResource) GetTagValue(category string) string {
	for _, tag := range r.Tags {
		if tag.Category == category {
			return tag.Value
		}
	}
	return ""
}

// ToTagSlice converts a slice of "category:value" strings to Tags.
func ToTagSlice(strings []string) ([]Tag, error) {
	tags := make([]Tag, 0, len(strings))
	for _, s := range strings {
		tag, err := ParseTag(s)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

// FromTagSlice converts Tags to a slice of "category:value" strings.
func FromTagSlice(tags []Tag) []string {
	strings := make([]string, len(tags))
	for i, tag := range tags {
		strings[i] = tag.String()
	}
	return strings
}

// TagsJSON is a helper type for JSON marshaling/unmarshaling of tags.
type TagsJSON []Tag

// MarshalJSON implements json.Marshaler.
func (t TagsJSON) MarshalJSON() ([]byte, error) {
	strings := FromTagSlice(t)
	return json.Marshal(strings)
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *TagsJSON) UnmarshalJSON(data []byte) error {
	var strings []string
	if err := json.Unmarshal(data, &strings); err != nil {
		return err
	}
	tags, err := ToTagSlice(strings)
	if err != nil {
		return err
	}
	*t = tags
	return nil
}
