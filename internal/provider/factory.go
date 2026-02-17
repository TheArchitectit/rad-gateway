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
	geminiAdapter := gemini.NewAdapter(config.APIKey,
		gemini.WithBaseURL(config.BaseURL),
		gemini.WithTimeout(config.Timeout.RequestTimeout),
	)

	reqTransform := &GeminiRequestTransformer{config: config}
	respTransform := &GeminiResponseTransformer{config: config, adapter: geminiAdapter}

	adapter := NewExecutableAdapter(config, reqTransform, respTransform)

	if config.StreamingEnabled {
		adapter.SetStreamTransformer(&GeminiStreamTransformer{adapter: geminiAdapter})
	}

	return adapter
}

// OpenAIRequestTransformer handles OpenAI-specific request transformations.
type OpenAIRequestTransformer struct {
	config ProviderConfig
}

// TransformHeaders adds OpenAI authentication headers.
func (t *OpenAIRequestTransformer) TransformHeaders(req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+t.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	for key, value := range t.config.Headers {
		req.Header.Set(key, value)
	}
	return nil
}

// TransformBody sets default model if not specified.
func (t *OpenAIRequestTransformer) TransformBody(body io.Reader, contentType string) (io.Reader, string, error) {
	if body == nil {
		return body, contentType, nil
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, "", err
	}
	var requestMap map[string]any
	if err := json.Unmarshal(data, &requestMap); err != nil {
		return bytes.NewReader(data), contentType, nil
	}
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
	req.URL.Scheme = "https"
	req.URL.Host = strings.TrimPrefix(t.config.BaseURL, "https://")
	return nil
}

// OpenAIResponseTransformer handles OpenAI-specific response transformations.
type OpenAIResponseTransformer struct {
	config ProviderConfig
}

// TransformHeaders normalizes OpenAI response headers.
func (t *OpenAIResponseTransformer) TransformHeaders(resp *http.Response) error {
	resp.Header.Set("X-Gateway-Provider", "openai")
	return nil
}

// TransformBody handles OpenAI-specific response body normalization.
func (t *OpenAIResponseTransformer) TransformBody(body io.Reader, contentType string) (io.Reader, string, error) {
	return body, contentType, nil
}

// TransformStatusCode normalizes OpenAI status codes.
func (t *OpenAIResponseTransformer) TransformStatusCode(code int) int {
	return code
}

// OpenAIStreamTransformer handles OpenAI SSE streaming transformations.
type OpenAIStreamTransformer struct{}

// TransformStreamChunk processes OpenAI SSE chunks.
func (t *OpenAIStreamTransformer) TransformStreamChunk(chunk []byte) ([]byte, error) {
	return chunk, nil
}

// IsDoneMarker checks for OpenAI stream completion.
func (t *OpenAIStreamTransformer) IsDoneMarker(chunk []byte) bool {
	return bytes.Contains(chunk, []byte("[DONE]"))
}

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
	var internalReq struct {
		Model       string `json:"model"`
		Messages    []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Stream      bool    `json:"stream,omitempty"`
		Temperature float64 `json:"temperature,omitempty"`
		MaxTokens   int     `json:"max_tokens,omitempty"`
	}
	if err := json.Unmarshal(data, &internalReq); err != nil {
		return bytes.NewReader(data), contentType, nil
	}
	anthropicReq := make(map[string]any)
	if internalReq.Model != "" {
		anthropicReq["model"] = internalReq.Model
	} else {
		anthropicReq["model"] = t.config.DefaultModel
	}
	messages := make([]map[string]any, len(internalReq.Messages))
	for i, m := range internalReq.Messages {
		messages[i] = map[string]any{
			"role":    m.Role,
			"content": m.Content,
		}
	}
	anthropicReq["messages"] = messages
	if internalReq.MaxTokens > 0 {
		anthropicReq["max_tokens"] = internalReq.MaxTokens
	} else {
		anthropicReq["max_tokens"] = 4096
	}
	if internalReq.Temperature != 0 {
		anthropicReq["temperature"] = internalReq.Temperature
	}
	anthropicReq["stream"] = internalReq.Stream
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

// GeminiRequestTransformer handles Google Gemini-specific request transformations.
type GeminiRequestTransformer struct {
	config ProviderConfig
}

// TransformHeaders adds Gemini authentication headers.
func (t *GeminiRequestTransformer) TransformHeaders(req *http.Request) error {
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
	var internalReq struct {
		Messages    []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Temperature float64 `json:"temperature,omitempty"`
		MaxTokens   int     `json:"max_tokens,omitempty"`
		Stream      bool    `json:"stream,omitempty"`
	}
	if err := json.Unmarshal(data, &internalReq); err != nil {
		return bytes.NewReader(data), contentType, nil
	}
	geminiReq := make(map[string]any)
	contents := make([]map[string]any, 0)
	for _, msg := range internalReq.Messages {
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, map[string]any{
			"role": role,
			"parts": []map[string]any{
				{"text": msg.Content},
			},
		})
	}
	geminiReq["contents"] = contents
	genConfig := make(map[string]any)
	if internalReq.Temperature != 0 {
		genConfig["temperature"] = internalReq.Temperature
	}
	if internalReq.MaxTokens != 0 {
		genConfig["maxOutputTokens"] = internalReq.MaxTokens
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
	query := req.URL.Query()
	query.Set("key", t.config.APIKey)
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
	var geminiResp gemini.GeminiResponse
	if err := json.Unmarshal(data, &geminiResp); err == nil && len(geminiResp.Candidates) > 0 {
		transformer := gemini.NewResponseTransformer()
		result, err := transformer.Transform(geminiResp, t.config.DefaultModel)
		if err == nil {
			output, err := json.Marshal(result)
			if err == nil {
				return bytes.NewReader(output), "application/json", nil
			}
		}
	}
	return bytes.NewReader(data), contentType, nil
}

// TransformStatusCode normalizes Gemini status codes.
func (t *GeminiResponseTransformer) TransformStatusCode(code int) int {
	return code
}

// GeminiStreamTransformer handles Gemini SSE streaming transformations.
type GeminiStreamTransformer struct {
	adapter   *gemini.Adapter
	transform *gemini.StreamTransformer
	model     string
}

// TransformStreamChunk processes Gemini SSE chunks.
func (t *GeminiStreamTransformer) TransformStreamChunk(chunk []byte) ([]byte, error) {
	if t.transform == nil {
		t.transform = gemini.NewStreamTransformer()
		t.transform.Init(t.model)
	}
	data := string(chunk)
	if strings.HasPrefix(data, "data: ") {
		data = strings.TrimPrefix(data, "data: ")
	}
	if data == "" || data == "[DONE]" {
		return chunk, nil
	}
	result, isFinal, err := t.transform.TransformChunk(data)
	if err != nil {
		return chunk, nil
	}
	if isFinal {
		result = append(result, []byte("data: [DONE]\n\n")...)
	}
	return result, nil
}

// IsDoneMarker checks for Gemini stream completion.
func (t *GeminiStreamTransformer) IsDoneMarker(chunk []byte) bool {
	return gemini.IsDoneMarker(chunk)
}
