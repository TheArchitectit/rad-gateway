package controlroom

import (
	"testing"
)

func TestTagString(t *testing.T) {
	tests := []struct {
		name     string
		tag      Tag
		expected string
	}{
		{
			name:     "simple tag",
			tag:      Tag{Category: "env", Value: "production"},
			expected: "env:production",
		},
		{
			name:     "tag with hyphen",
			tag:      Tag{Category: "cost-center", Value: "engineering"},
			expected: "cost-center:engineering",
		},
		{
			name:     "empty values",
			tag:      Tag{Category: "", Value: ""},
			expected: ":",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tag.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTagEquals(t *testing.T) {
	tests := []struct {
		name     string
		tag1     Tag
		tag2     Tag
		expected bool
	}{
		{
			name:     "same tags",
			tag1:     Tag{Category: "env", Value: "production"},
			tag2:     Tag{Category: "env", Value: "production"},
			expected: true,
		},
		{
			name:     "different value",
			tag1:     Tag{Category: "env", Value: "production"},
			tag2:     Tag{Category: "env", Value: "staging"},
			expected: false,
		},
		{
			name:     "different category",
			tag1:     Tag{Category: "env", Value: "production"},
			tag2:     Tag{Category: "team", Value: "production"},
			expected: false,
		},
		{
			name:     "case sensitive",
			tag1:     Tag{Category: "env", Value: "Production"},
			tag2:     Tag{Category: "env", Value: "production"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.tag1.Equals(tt.tag2)
			if result != tt.expected {
				t.Errorf("Equals() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTagMatchesWildcard(t *testing.T) {
	tests := []struct {
		name    string
		tag     Tag
		pattern string
		want    bool
	}{
		// Exact match
		{name: "exact match", tag: Tag{Category: "env", Value: "production"}, pattern: "production", want: true},
		{name: "no exact match", tag: Tag{Category: "env", Value: "production"}, pattern: "staging", want: false},

		// Prefix wildcard
		{name: "prefix match", tag: Tag{Category: "project", Value: "customer-abc"}, pattern: "customer-*", want: true},
		{name: "prefix no match", tag: Tag{Category: "project", Value: "internal-abc"}, pattern: "customer-*", want: false},

		// Suffix wildcard
		{name: "suffix match", tag: Tag{Category: "env", Value: "us-east"}, pattern: "*-east", want: true},
		{name: "suffix no match", tag: Tag{Category: "env", Value: "us-west"}, pattern: "*-east", want: false},

		// Contains wildcard
		{name: "contains match", tag: Tag{Category: "env", Value: "prod-us-east"}, pattern: "*us*", want: true},
		{name: "contains no match", tag: Tag{Category: "env", Value: "prod-eu-west"}, pattern: "*us*", want: false},

		// Full wildcard
		{name: "full wildcard", tag: Tag{Category: "env", Value: "anything"}, pattern: "*", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tag.MatchesWildcard(tt.pattern)
			if got != tt.want {
				t.Errorf("MatchesWildcard(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestParseTag(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Tag
		wantErr bool
	}{
		{
			name:    "valid tag",
			input:   "env:production",
			want:    Tag{Category: "env", Value: "production"},
			wantErr: false,
		},
		{
			name:    "valid tag with whitespace",
			input:   "  env  :  production  ",
			want:    Tag{Category: "env", Value: "production"},
			wantErr: false,
		},
		{
			name:    "missing colon",
			input:   "envproduction",
			want:    Tag{},
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			want:    Tag{},
			wantErr: true,
		},
		{
			name:    "multiple colons",
			input:   "env:prod:test",
			want:    Tag{Category: "env", Value: "prod:test"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTag(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidCategory(t *testing.T) {
	tests := []struct {
		name     string
		category string
		want     bool
	}{
		{name: "lowercase", category: "env", want: true},
		{name: "uppercase", category: "ENV", want: true},
		{name: "mixed case", category: "EnV", want: true},
		{name: "with hyphen", category: "cost-center", want: true},
		{name: "with underscore", category: "cost_center", want: true},
		{name: "with number", category: "env2", want: true},
		{name: "empty", category: "", want: false},
		{name: "with space", category: "cost center", want: false},
		{name: "with special char", category: "cost@center", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidCategory(tt.category)
			if got != tt.want {
				t.Errorf("IsValidCategory(%q) = %v, want %v", tt.category, got, tt.want)
			}
		})
	}
}

func TestTaggableResource(t *testing.T) {
	tr := &TaggableResource{}

	// Test AddTag
	tag1 := Tag{Category: "env", Value: "production"}
	tag2 := Tag{Category: "team", Value: "platform"}

	tr.AddTag(tag1)
	if !tr.HasTag(tag1) {
		t.Error("HasTag() should return true after AddTag()")
	}

	tr.AddTag(tag2)
	if len(tr.GetTags()) != 2 {
		t.Errorf("GetTags() should return 2 tags, got %d", len(tr.GetTags()))
	}

	// Test duplicate AddTag
	tr.AddTag(tag1)
	if len(tr.GetTags()) != 2 {
		t.Errorf("GetTags() should still return 2 tags after duplicate AddTag, got %d", len(tr.GetTags()))
	}

	// Test RemoveTag
	tr.RemoveTag(tag1)
	if tr.HasTag(tag1) {
		t.Error("HasTag() should return false after RemoveTag()")
	}
	if len(tr.GetTags()) != 1 {
		t.Errorf("GetTags() should return 1 tag after RemoveTag, got %d", len(tr.GetTags()))
	}

	// Test GetTags returns copy
	tags := tr.GetTags()
	tags = append(tags, Tag{Category: "test", Value: "value"})
	if len(tr.GetTags()) != 1 {
		t.Error("GetTags() should return a copy, not the original slice")
	}

	// Test SetTags
	newTags := []Tag{{Category: "a", Value: "1"}, {Category: "b", Value: "2"}}
	tr.SetTags(newTags)
	if len(tr.GetTags()) != 2 {
		t.Errorf("SetTags() should set 2 tags, got %d", len(tr.GetTags()))
	}

	// Modify original and verify copy
	newTags[0].Value = "changed"
	if tr.GetTags()[0].Value == "changed" {
		t.Error("SetTags() should make a copy of the tags")
	}
}

func TestTaggedResource(t *testing.T) {
	resource := TaggedResource{
		ID:   "provider-123",
		Type: "provider",
		Tags: []Tag{
			{Category: "env", Value: "production"},
			{Category: "team", Value: "platform"},
			{Category: "env", Value: "us-east"}, // Multiple tags with same category
		},
	}

	// Test HasCategory
	if !resource.HasCategory("env") {
		t.Error("HasCategory('env') should return true")
	}
	if !resource.HasCategory("team") {
		t.Error("HasCategory('team') should return true")
	}
	if resource.HasCategory("project") {
		t.Error("HasCategory('project') should return false")
	}

	// Test GetTagValue
	if got := resource.GetTagValue("team"); got != "platform" {
		t.Errorf("GetTagValue('team') = %q, want 'platform'", got)
	}
	if got := resource.GetTagValue("nonexistent"); got != "" {
		t.Errorf("GetTagValue('nonexistent') = %q, want empty string", got)
	}
	// Returns first match when multiple tags have same category
	if got := resource.GetTagValue("env"); got == "" {
		t.Error("GetTagValue('env') should not return empty")
	}
}

func TestToTagSlice(t *testing.T) {
	input := []string{"env:production", "team:platform"}
	tags, err := ToTagSlice(input)
	if err != nil {
		t.Fatalf("ToTagSlice() error = %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("ToTagSlice() returned %d tags, want 2", len(tags))
	}

	// Test invalid input
	_, err = ToTagSlice([]string{"invalid"})
	if err == nil {
		t.Error("ToTagSlice() should return error for invalid input")
	}
}

func TestFromTagSlice(t *testing.T) {
	tags := []Tag{
		{Category: "env", Value: "production"},
		{Category: "team", Value: "platform"},
	}
	result := FromTagSlice(tags)
	if len(result) != 2 {
		t.Errorf("FromTagSlice() returned %d strings, want 2", len(result))
	}
	if result[0] != "env:production" || result[1] != "team:platform" {
		t.Errorf("FromTagSlice() = %v, want [env:production team:platform]", result)
	}
}

func BenchmarkTagString(b *testing.B) {
	tag := Tag{Category: "cost-center", Value: "engineering"}
	for i := 0; i < b.N; i++ {
		_ = tag.String()
	}
}

func BenchmarkTagEquals(b *testing.B) {
	tag1 := Tag{Category: "env", Value: "production"}
	tag2 := Tag{Category: "env", Value: "production"}
	for i := 0; i < b.N; i++ {
		_ = tag1.Equals(tag2)
	}
}

func BenchmarkMatchesWildcard(b *testing.B) {
	tag := Tag{Category: "project", Value: "customer-abc-123"}
	for i := 0; i < b.N; i++ {
		_ = tag.MatchesWildcard("customer-*")
	}
}
