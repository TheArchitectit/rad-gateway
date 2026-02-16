package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"radgateway/internal/provider/gemini"
)

// AdapterFactory creates provider adapters with appropriate transformers.
type AdapterFactory struct {
	configs map[string]ProviderConfig
}

// NewAdapterFactory creates a new factory with the given provider configurations.
func NewAdapterFactory(configs map[string]ProviderConfig) *AdapterFactory {
	return &AdapterFactory{configs: configs}
}

// CreateAdapter builds a fully configured adapter for the specified provider.
func (f *AdapterFactory) CreateAdapter(providerName string) (AdapterWithContext, error) {
	config, ok := f.configs[providerName]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	switch providerName {
	case "openai":
		return f.createOpenAIAdapter(config), nil
	case "anthropic":
		return f.createAnthropicAdapter(config), nil
	case "gemini":
		return f.createGeminiAdapter(config), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

// RegisterProvider adds or updates a provider configuration.
func (f *AdapterFactory) RegisterProvider(config ProviderConfig) {
	f.configs[config.Name] = config
}

// GetRegisteredProviders returns the list of configured provider names.
func (f *AdapterFactory) GetRegisteredProviders() []string {
	names := make([]string, 0, len(f.configs))
	for name := range f.configs {
		names = append(names, name)
	}
	return names
}

// createOpenAIAdapter creates an adapter configured for OpenAI API.
func (f *AdapterFactory) createOpenAIAdapter(config ProviderConfig) AdapterWithContext {
	reqTransform := &OpenAIRequestTransformer{config: config}
	respTransform := &OpenAIResponseTransformer{config: config}

	adapter := NewExecutableAdapter(config, reqTransform, respTransform)

	if config.StreamingEnabled {
		adapter.SetStreamTransformer(&OpenAIStreamTransformer{})
	}

	return adapter
}

// createAnthropicAdapter creates an adapter configured for Anthropic API.
func (f *AdapterFactory) createAnthropicAdapter(config ProviderConfig) AdapterWithContext {
	reqTransform := &AnthropicRequestTransformer{config: config}
	respTransform := &AnthropicResponseTransformer{config: config}

	adapter := NewExecutableAdapter(config, reqTransform, respTransform)

	if config.StreamingEnabled {
		adapter.SetStreamTransformer(&AnthropicStreamTransformer{})
	}

	return adapter
}

// createGeminiAdapter creates an adapter configured for Google Gemini API.
func (f *AdapterFactory) createGeminiAdapter(config ProviderConfig) AdapterWithContext {
	// Create the gemini package adapter with configuration
	geminiAdapter := gemini.NewAdapter(config.APIKey,
		gemini.WithBaseURL(config.BaseURL),
		gemini.WithTimeout(config.Timeout.RequestTimeout),
	)

	// Create wrapper transformers that implement the provider package interfaces
	// while delegating to the gemini package implementations
	reqTransform := &GeminiRequestTransformer{config: config}
	respTransform := &GeminiResponseTransformer{config: config, adapter: geminiAdapter}

	adapter := NewExecutableAdapter(config, reqTransform, respTransform)

	if config.StreamingEnabled {
		adapter.SetStreamTransformer(&GeminiStreamTransformer{adapter: geminiAdapter})
	}

	return adapter
}

// =============================================================================
// OpenAI Transformers
// =============================================================================

// OpenAIRequestTransformer handles OpenAI-specific request transformations.
type OpenAIRequestTransformer struct {
	config ProviderConfig
}

// TransformHeaders adds OpenAI authentication and version headers.
func (t *OpenAIRequestTransformer) TransformHeaders(req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+t.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	for key, value := range t.config.Headers {
		req.Header.Set(key, value)
	}

	return nil
}

// TransformBody handles OpenAI-specific body modifications.
func (t *OpenAIRequestTransformer) TransformBody(body io.Reader, contentType string) (io.Reader, string, error) {
	if body == nil {
		return body, contentType, nil
	}

	// OpenAI accepts the standard format, so we mainly pass through
	// but could add provider-specific fields here
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, "", err
	}

	// Parse and potentially modify the request
	var requestMap map[string]any
	if err := json.Unmarshal(data, &requestMap); err != nil {
		// Not JSON, pass through as-is
		return bytes.NewReader(data), contentType, nil
	}

	// Set default model if not specified
	if model, ok := requestMap["model"].(string); !ok || model == "" {
		requestMap["model"] = t.config.DefaultModel
	}

	modified, err := json.Marshal(requestMap)
	if err != nil {
		return nil, "", err
	}

	return bytes.NewReader(modified), "application/json", nil
}

// TransformURL modifies the request URL for OpenAI endpoints.
func (t *OpenAIRequestTransformer) TransformURL(req *http.Request) error {
	// Rewrite URL to point to OpenAI API
	req.URL.Scheme = "https"
	req.URL.Host = strings.TrimPrefix(t.config.BaseURL, "https://")
	// Path is preserved from original request
	return nil
}

// OpenAIResponseTransformer handles OpenAI-specific response transformations.
type OpenAIResponseTransformer struct {
	config ProviderConfig
}

// TransformHeaders normalizes OpenAI response headers.
func (t *OpenAIResponseTransformer) TransformHeaders(resp *http.Response) error {
	// Add gateway identification header
	resp.Header.Set("X-Gateway-Provider", "openai")
	return nil
}

// TransformBody handles OpenAI-specific response body normalization.
func (t *OpenAIResponseTransformer) TransformBody(body io.Reader, contentType string) (io.Reader, string, error) {
	// OpenAI responses are already in a standard format, pass through
	return body, contentType, nil
}

// TransformStatusCode normalizes OpenAI status codes.
func (t *OpenAIResponseTransformer) TransformStatusCode(code int) int {
	// OpenAI status codes are standard, no transformation needed
	return code
}

// OpenAIStreamTransformer handles OpenAI SSE streaming transformations.
type OpenAIStreamTransformer struct{}

// TransformStreamChunk processes OpenAI SSE chunks.
func (t *OpenAIStreamTransformer) TransformStreamChunk(chunk []byte) ([]byte, error) {
	// OpenAI SSE format is already standard, pass through
	return chunk, nil
}

// IsDoneMarker checks for OpenAI stream completion.
func (t *OpenAIStreamTransformer) IsDoneMarker(chunk []byte) bool {
	return bytes.Contains(chunk, []byte("[DONE]"))
}

// =============================================================================
// Anthropic Transformers
// =============================================================================

// AnthropicRequestTransformer handles Anthropic-specific request transformations.
type AnthropicRequestTransformer struct {
	config ProviderConfig
}

// TransformHeaders adds Anthropic authentication headers.
func (t *AnthropicRequestTransformer) TransformHeaders(req *http.Request) error {
	req.Header.Set("x-api-key", t.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	for key, value := range t.config.Headers {
		req.Header.Set(key, value)
	}

	return nil
}

// TransformBody converts internal format to Anthropic's expected format.
func (t *AnthropicRequestTransformer) TransformBody(body io.Reader, contentType string) (io.Reader, string, error) {
	if body == nil {
		return body, contentType, nil
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, "", err
	}

	// Parse internal request format
	var internalReq map[string]any
	if err := json.Unmarshal(data, &internalReq); err != nil {
		return bytes.NewReader(data), contentType, nil
	}

	// Transform to Anthropic format
	anthropicReq := make(map[string]any)

	// Map model
	if model, ok := internalReq["model"].(string); ok {
		anthropicReq["model"] = model
	} else {
		anthropicReq["model"] = t.config.DefaultModel
	}

	// Transform messages format if present
	if messages, ok := internalReq["messages"].([]any); ok {
		anthropicReq["messages"] = messages
	}

	// Map max_tokens
	if maxTokens, ok := internalReq["max_tokens"].(float64); ok {
		anthropicReq["max_tokens"] = int(maxTokens)
	} else {
		anthropicReq["max_tokens"] = 4096 // Default
	}

	// Map temperature
	if temp, ok := internalReq["temperature"].(float64); ok {
		anthropicReq["temperature"] = temp
	}

	// Map stream
	if stream, ok := internalReq["stream"].(bool); ok {
		anthropicReq["stream"] = stream
	}

	modified, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, "", err
	}

	return bytes.NewReader(modified), "application/json", nil
}

// TransformURL modifies the request URL for Anthropic endpoints.
func (t *AnthropicRequestTransformer) TransformURL(req *http.Request) error {
	req.URL.Scheme = "https"
	req.URL.Host = strings.TrimPrefix(t.config.BaseURL, "https://")
	return nil
}

// AnthropicResponseTransformer handles Anthropic-specific response transformations.
type AnthropicResponseTransformer struct {
	config ProviderConfig
}

// TransformHeaders normalizes Anthropic response headers.
func (t *AnthropicResponseTransformer) TransformHeaders(resp *http.Response) error {
	resp.Header.Set("X-Gateway-Provider", "anthropic")
	return nil
}

// TransformBody converts Anthropic response to internal standard format.
func (t *AnthropicResponseTransformer) TransformBody(body io.Reader, contentType string) (io.Reader, string, error) {
	// For now, pass through Anthropic responses
	// Future: normalize to a common response format
	return body, contentType, nil
}

// TransformStatusCode normalizes Anthropic status codes.
func (t *AnthropicResponseTransformer) TransformStatusCode(code int) int {
	return code
}

// AnthropicStreamTransformer handles Anthropic SSE streaming transformations.
type AnthropicStreamTransformer struct{}

// TransformStreamChunk processes Anthropic SSE chunks.
func (t *AnthropicStreamTransformer) TransformStreamChunk(chunk []byte) ([]byte, error) {
	return chunk, nil
}

// IsDoneMarker checks for Anthropic stream completion.
func (t *AnthropicStreamTransformer) IsDoneMarker(chunk []byte) bool {
	return bytes.Contains(chunk, []byte("event: message_stop"))
}

// =============================================================================
// Gemini Transformers
// =============================================================================

// GeminiRequestTransformer handles Google Gemini-specific request transformations.
// It wraps the gemini package's RequestTransformer to implement the provider.RequestTransformer interface.
type GeminiRequestTransformer struct {
	config    ProviderConfig
	transform *gemini.RequestTransformer
}

// TransformHeaders adds Gemini authentication headers.
func (t *GeminiRequestTransformer) TransformHeaders(req *http.Request) error {
	// Gemini uses API key in query params, not headers
	req.Header.Set("Content-Type", "application/json")

	for key, value := range t.config.Headers {
		req.Header.Set(key, value)
	}

	return nil
}

// TransformBody converts internal format to Gemini's expected format.
func (t *GeminiRequestTransformer) TransformBody(body io.Reader, contentType string) (io.Reader, string, error) {
	if body == nil {
		return body, contentType, nil
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, "", err
	}

	var internalReq map[string]any
	if err := json.Unmarshal(data, &internalReq); err != nil {
		return bytes.NewReader(data), contentType, nil
	}

	// Transform to Gemini format
	geminiReq := make(map[string]any)

	// Gemini uses contents array with parts
	contents := make([]map[string]any, 0)

	if messages, ok := internalReq["messages"].([]any); ok {
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]any); ok {
				role, _ := msgMap["role"].(string)
				content, _ := msgMap["content"].(string)

				// Map roles: user -> user, assistant -> model
				geminiRole := role
				if role == "assistant" {
					geminiRole = "model"
				}

				contents = append(contents, map[string]any{
					"role": geminiRole,
					"parts": []map[string]any{
						{"text": content},
					},
				})
			}
		}
	}

	geminiReq["contents"] = contents

	// Add generation config
	genConfig := make(map[string]any)
	if temp, ok := internalReq["temperature"].(float64); ok {
		genConfig["temperature"] = temp
	}
	if maxTokens, ok := internalReq["max_tokens"].(float64); ok {
		genConfig["maxOutputTokens"] = int(maxTokens)
	}
	if len(genConfig) > 0 {
		geminiReq["generationConfig"] = genConfig
	}

	modified, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, "", err
	}

	return bytes.NewReader(modified), "application/json", nil
}

// TransformURL modifies the request URL for Gemini endpoints.
func (t *GeminiRequestTransformer) TransformURL(req *http.Request) error {
	req.URL.Scheme = "https"
	req.URL.Host = strings.TrimPrefix(t.config.BaseURL, "https://")

	// Add API key as query parameter
	query := req.URL.Query()
	query.Set("key", t.config.APIKey)

	// Check if streaming
	data, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(bytes.NewReader(data))

	var bodyMap map[string]any
	if err := json.Unmarshal(data, &bodyMap); err == nil {
		if stream, ok := bodyMap["stream"].(bool); ok && stream {
			query.Set("alt", "sse")
		}
	}

	req.URL.RawQuery = query.Encode()
	return nil
}

// GeminiResponseTransformer handles Gemini-specific response transformations.
// It wraps the gemini package's ResponseTransformer to implement the provider.ResponseTransformer interface.
type GeminiResponseTransformer struct {
	config  ProviderConfig
	adapter *gemini.Adapter
}

// TransformHeaders normalizes Gemini response headers.
func (t *GeminiResponseTransformer) TransformHeaders(resp *http.Response) error {
	resp.Header.Set("X-Gateway-Provider", "gemini")
	return nil
}

// TransformBody converts Gemini response to internal standard format.
func (t *GeminiResponseTransformer) TransformBody(body io.Reader, contentType string) (io.Reader, string, error) {
	if body == nil {
		return body, contentType, nil
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, "", err
	}

	// Try to parse and transform Gemini response
	var geminiResp gemini.GeminiResponse
	if err := json.Unmarshal(data, &geminiResp); err == nil && len(geminiResp.Candidates) > 0 {
		// Use the gemini package's transformer
		transformer := gemini.NewResponseTransformer()
		result, err := transformer.Transform(geminiResp, t.config.DefaultModel)
		if err == nil {
			// Convert to JSON and return
			output, err := json.Marshal(result)
			if err == nil {
				return bytes.NewReader(output), "application/json", nil
			}
		}
	}

	// Fall back to passing through unchanged if transformation fails
	return bytes.NewReader(data), contentType, nil
}

// TransformStatusCode normalizes Gemini status codes.
func (t *GeminiResponseTransformer) TransformStatusCode(code int) int {
	return code
}

// GeminiStreamTransformer handles Gemini SSE streaming transformations.
// It wraps the gemini package's StreamTransformer to implement the provider.StreamTransformer interface.
type GeminiStreamTransformer struct {
	adapter   *gemini.Adapter
	transform *gemini.StreamTransformer
	model     string
}

// TransformStreamChunk processes Gemini SSE chunks.
func (t *GeminiStreamTransformer) TransformStreamChunk(chunk []byte) ([]byte, error) {
	// Initialize transformer on first use
	if t.transform == nil {
		t.transform = gemini.NewStreamTransformer()
		t.transform.Init(t.model)
	}

	// Extract data prefix if present
	data := string(chunk)
	if strings.HasPrefix(data, "data: ") {
		data = strings.TrimPrefix(data, "data: ")
	}

	// Skip empty lines and [DONE] markers
	if data == "" || data == "[DONE]" {
		return chunk, nil
	}

	// Transform the chunk using gemini package
	result, isFinal, err := t.transform.TransformChunk(data)
	if err != nil {
		// If transformation fails, pass through unchanged
		return chunk, nil
	}

	// The gemini TransformChunk returns already formatted SSE data
	if isFinal {
		// Add DONE marker after the result
		result = append(result, []byte("data: [DONE]\n\n")...)
	}

	return result, nil
}

// IsDoneMarker checks for Gemini stream completion.
func (t *GeminiStreamTransformer) IsDoneMarker(chunk []byte) bool {
	return gemini.IsDoneMarker(chunk)
}
