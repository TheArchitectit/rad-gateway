package a2a

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"radgateway/internal/logger"
)

type AgentCard struct {
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	URL            string       `json:"url"`
	Version        string       `json:"version"`
	Capabilities   Capabilities `json:"capabilities"`
	Skills         []Skill      `json:"skills"`
	Authentication AuthInfo     `json:"authentication"`
}

type Capabilities struct {
	Streaming              bool `json:"streaming"`
	PushNotifications      bool `json:"pushNotifications"`
	StateTransitionHistory bool `json:"stateTransitionHistory"`
}

type Skill struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Tags        []string               `json:"tags,omitempty"`
	Examples    []string               `json:"examples,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

type AuthInfo struct {
	Schemes []string `json:"schemes"`
}

type AgentCardHandler struct {
	baseURL string
	version string
	log     *slog.Logger
}

func NewAgentCardHandler(baseURL, version string) *AgentCardHandler {
	return &AgentCardHandler{
		baseURL: baseURL,
		version: version,
		log:     logger.WithComponent("a2a_agent_card"),
	}
}

func (h *AgentCardHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/.well-known/agent.json", h.handleAgentCard)
}

func (h *AgentCardHandler) handleAgentCard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	card := h.generateAgentCard()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(card)
}

func (h *AgentCardHandler) generateAgentCard() AgentCard {
	return AgentCard{
		Name:        "RAD Gateway",
		Description: "AI API Gateway with A2A protocol support for multi-provider LLM routing",
		URL:         h.baseURL + "/a2a",
		Version:     h.version,
		Capabilities: Capabilities{
			Streaming:              true,
			PushNotifications:      false,
			StateTransitionHistory: true,
		},
		Skills: []Skill{
			{
				ID:          "chat",
				Name:        "Chat Completions",
				Description: "OpenAI-compatible chat completion API",
				Tags:        []string{"chat", "llm", "openai"},
				Examples:    []string{"Generate a summary", "Translate to French"},
			},
			{
				ID:          "embeddings",
				Name:        "Text Embeddings",
				Description: "Generate vector embeddings for text",
				Tags:        []string{"embeddings", "vector"},
				Examples:    []string{"Embed document", "Create search index"},
			},
			{
				ID:          "images",
				Name:        "Image Generation",
				Description: "Generate images using DALL-E and similar models",
				Tags:        []string{"images", "dalle", "generation"},
				Examples:    []string{"Generate a logo", "Create illustration"},
			},
		},
		Authentication: AuthInfo{
			Schemes: []string{"Bearer", "APIKey"},
		},
	}
}
