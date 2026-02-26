// Package benchmarks provides performance benchmarks for critical paths.
package benchmarks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"radgateway/internal/models"
	"radgateway/internal/provider/openai"
)

// BenchmarkOpenAIRequestTransform benchmarks request transformation.
func BenchmarkOpenAIRequestTransform(b *testing.B) {
	transformer := openai.NewRequestTransformer()

	req := models.ChatCompletionRequest{
		Model:    "gpt-4",
		Stream:   false,
		User:     "user-123",
		Messages: make([]models.Message, 10),
	}

	for i := range req.Messages {
		req.Messages[i] = models.Message{
			Role:    "user",
			Content: fmt.Sprintf("Message %d: This is a test message for benchmarking purposes.", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = transformer.Transform(req)
	}
}

// BenchmarkOpenAIResponseTransform benchmarks response transformation.
func BenchmarkOpenAIResponseTransform(b *testing.B) {
	transformer := openai.NewResponseTransformer()

	resp := openai.OpenAIResponse{
		ID:      "chatcmpl-test123",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "gpt-4",
		Choices: make([]openai.OpenAIChoice, 5),
		Usage: openai.OpenAIUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}

	for i := range resp.Choices {
		resp.Choices[i] = openai.OpenAIChoice{
			Index: i,
			Message: openai.OpenAIMessage{
				Role:    "assistant",
				Content: fmt.Sprintf("Response %d: This is a generated response.", i),
			},
			FinishReason: "stop",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := transformer.Transform(resp)
		if err != nil {
			b.Fatalf("Transform failed: %v", err)
		}
	}
}

// BenchmarkOpenAIStreamTransform benchmarks stream chunk transformation.
func BenchmarkOpenAIStreamTransform(b *testing.B) {
	transformer := openai.NewStreamTransformer()

	chunk := openai.OpenAIStreamResponse{
		ID:      "chatcmpl-stream123",
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   "gpt-4",
		Choices: []openai.OpenAIStreamChoice{
			{
				Index: 0,
				Delta: openai.OpenAIMessageDelta{
					Role:    "assistant",
					Content: "Hello",
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = transformer.TransformChunk(chunk)
	}
}

// BenchmarkOpenAIFullTransformCycle benchmarks complete request/response cycle.
func BenchmarkOpenAIFullTransformCycle(b *testing.B) {
	reqTransformer := openai.NewRequestTransformer()
	respTransformer := openai.NewResponseTransformer()

	req := models.ChatCompletionRequest{
		Model:    "gpt-4",
		Stream:   false,
		User:     "user-123",
		Messages: make([]models.Message, 5),
	}

	for i := range req.Messages {
		req.Messages[i] = models.Message{
			Role:    "user",
			Content: fmt.Sprintf("Message %d", i),
		}
	}

	resp := openai.OpenAIResponse{
		ID:      "chatcmpl-test",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "gpt-4",
		Choices: []openai.OpenAIChoice{
			{
				Index: 0,
				Message: openai.OpenAIMessage{
					Role:    "assistant",
					Content: "Test response",
				},
				FinishReason: "stop",
			},
		},
		Usage: openai.OpenAIUsage{
			PromptTokens:     50,
			CompletionTokens: 25,
			TotalTokens:      75,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Transform request
		_ = reqTransformer.Transform(req)

		// Transform response
		_, err := respTransformer.Transform(resp)
		if err != nil {
			b.Fatalf("Transform failed: %v", err)
		}
	}
}

// BenchmarkOpenAIMessageSerialization benchmarks message JSON serialization.
func BenchmarkOpenAIMessageSerialization(b *testing.B) {
	messages := make([]openai.OpenAIMessage, 10)
	for i := range messages {
		messages[i] = openai.OpenAIMessage{
			Role:    "user",
			Content: fmt.Sprintf("Message %d with some content here", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(messages)
		if err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}
	}
}

// BenchmarkOpenAIResponseDeserialization benchmarks response JSON deserialization.
func BenchmarkOpenAIResponseDeserialization(b *testing.B) {
	resp := openai.OpenAIResponse{
		ID:      "chatcmpl-test",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   "gpt-4",
		Choices: []openai.OpenAIChoice{
			{
				Index: 0,
				Message: openai.OpenAIMessage{
					Role:    "assistant",
					Content: "Test response content here",
				},
				FinishReason: "stop",
			},
		},
		Usage: openai.OpenAIUsage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}

	data, _ := json.Marshal(resp)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var parsed openai.OpenAIResponse
		err := json.Unmarshal(data, &parsed)
		if err != nil {
			b.Fatalf("Unmarshal failed: %v", err)
		}
	}
}

// BenchmarkOpenAIAdapterCreation benchmarks adapter instantiation.
func BenchmarkOpenAIAdapterCreation(b *testing.B) {
	apiKey := "test-api-key-12345"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = openai.NewAdapter(apiKey)
	}
}

// BenchmarkOpenAIStreamParsing benchmarks SSE stream parsing.
func BenchmarkOpenAIStreamParsing(b *testing.B) {
	// Create SSE data
	chunk1, _ := json.Marshal(openai.OpenAIStreamResponse{
		ID:      "chatcmpl-1",
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   "gpt-4",
		Choices: []openai.OpenAIStreamChoice{
			{
				Index: 0,
				Delta: openai.OpenAIMessageDelta{
					Content: "Hello",
				},
			},
		},
	})

	chunk2, _ := json.Marshal(openai.OpenAIStreamResponse{
		ID:      "chatcmpl-2",
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   "gpt-4",
		Choices: []openai.OpenAIStreamChoice{
			{
				Index: 0,
				Delta: openai.OpenAIMessageDelta{
					Content: " world",
				},
			},
		},
	})

	sseData := fmt.Sprintf("data: %s\n\ndata: %s\n\ndata: [DONE]\n\n", chunk1, chunk2)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader([]byte(sseData))
		_, err := openai.ParseSSE(reader)
		if err != nil {
			b.Fatalf("ParseSSE failed: %v", err)
		}
	}
}

// BenchmarkOpenAIStreamChunkParsing benchmarks individual chunk parsing.
func BenchmarkOpenAIStreamChunkParsing(b *testing.B) {
	chunk := openai.OpenAIStreamResponse{
		ID:      "chatcmpl-test",
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   "gpt-4",
		Choices: []openai.OpenAIStreamChoice{
			{
				Index: 0,
				Delta: openai.OpenAIMessageDelta{
					Role:    "assistant",
					Content: "Test content",
				},
				FinishReason: nil,
			},
		},
	}
	data, _ := json.Marshal(chunk)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := openai.ParseStreamChunk(string(data))
		if err != nil {
			b.Fatalf("ParseStreamChunk failed: %v", err)
		}
	}
}

// BenchmarkOpenAIRequestMarshaling benchmarks request marshaling.
func BenchmarkOpenAIRequestMarshaling(b *testing.B) {
	req := openai.OpenAIRequest{
		Model: "gpt-4",
		Messages: []openai.OpenAIMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello, how are you?"},
		},
		Temperature: 0.7,
		MaxTokens:   150,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(req)
		if err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}
	}
}

// BenchmarkOpenAIErrorHandling benchmarks error response handling.
func BenchmarkOpenAIErrorHandling(b *testing.B) {
	errResp := openai.OpenAIError{
		Message: "Rate limit exceeded",
		Type:    "rate_limit_error",
		Code:    "rate_limit_exceeded",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = errResp.Error()
	}
}

// BenchmarkOpenAIChoiceProcessing benchmarks choice processing.
func BenchmarkOpenAIChoiceProcessing(b *testing.B) {
	choices := make([]openai.OpenAIChoice, 10)
	for i := range choices {
		choices[i] = openai.OpenAIChoice{
			Index: i,
			Message: openai.OpenAIMessage{
				Role:    "assistant",
				Content: fmt.Sprintf("Choice %d response content here", i),
			},
			FinishReason: "stop",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := make([]models.ChatChoice, len(choices))
		for j, c := range choices {
			result[j] = models.ChatChoice{
				Index: c.Index,
				Message: models.Message{
					Role:    c.Message.Role,
					Content: c.Message.Content,
				},
				FinishReason: c.FinishReason,
			}
		}
		_ = result
	}
}

// BenchmarkOpenAIUsageCalculation benchmarks usage calculation.
func BenchmarkOpenAIUsageCalculation(b *testing.B) {
	messages := make([]models.Message, 20)
	for i := range messages {
		messages[i] = models.Message{
			Role:    "user",
			Content: fmt.Sprintf("This is message number %d with some content to count tokens.", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var totalTokens int
		for _, msg := range messages {
			// Rough token estimation (characters / 4)
			totalTokens += len(msg.Content) / 4
		}
		_ = totalTokens
	}
}

// BenchmarkOpenAIMemoryAlloc benchmarks memory allocations.
func BenchmarkOpenAIMemoryAlloc(b *testing.B) {
	b.ReportAllocs()

	transformer := openai.NewRequestTransformer()
	req := models.ChatCompletionRequest{
		Model:    "gpt-4",
		Messages: make([]models.Message, 10),
	}

	for i := range req.Messages {
		req.Messages[i] = models.Message{
			Role:    "user",
			Content: fmt.Sprintf("Message %d: Test content here for memory allocation benchmarking.", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := transformer.Transform(req)
		_ = result.Model
	}
}

// BenchmarkOpenAIModelVariations benchmarks different models.
func BenchmarkOpenAIModelVariations(b *testing.B) {
	modelNames := []string{"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo", "gpt-4o", "gpt-4o-mini"}

	for _, model := range modelNames {
		b.Run(model, func(b *testing.B) {
			transformer := openai.NewRequestTransformer()
			req := models.ChatCompletionRequest{
				Model:    model,
				Messages: []models.Message{{Role: "user", Content: "Test message"}},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = transformer.Transform(req)
			}
		})
	}
}

// BenchmarkOpenAIMessageSizeComparison benchmarks different message sizes.
func BenchmarkOpenAIMessageSizeComparison(b *testing.B) {
	sizes := map[string]int{
		"Small":   10,
		"Medium":  100,
		"Large":   1000,
		"XLarge":  10000,
	}

	for name, size := range sizes {
		b.Run(name, func(b *testing.B) {
			transformer := openai.NewRequestTransformer()
			content := make([]byte, size)
			for i := range content {
				content[i] = byte('a' + (i % 26))
			}

			req := models.ChatCompletionRequest{
				Model: "gpt-4",
				Messages: []models.Message{
					{Role: "user", Content: string(content)},
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = transformer.Transform(req)
			}
		})
	}
}

// BenchmarkOpenAIParallel benchmarks concurrent operations.
func BenchmarkOpenAIParallel(b *testing.B) {
	transformer := openai.NewRequestTransformer()
	req := models.ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []models.Message{
			{Role: "user", Content: "Parallel test message"},
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = transformer.Transform(req)
		}
	})
}

// BenchmarkEndToEndRequestLatency simulates end-to-end request latency.
func BenchmarkEndToEndRequestLatency(b *testing.B) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate processing time
		time.Sleep(1 * time.Millisecond)

		resp := openai.OpenAIResponse{
			ID:      "chatcmpl-test",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "gpt-4",
			Choices: []openai.OpenAIChoice{
				{
					Index: 0,
					Message: openai.OpenAIMessage{
						Role:    "assistant",
						Content: "Test response",
					},
					FinishReason: "stop",
				},
			},
			Usage: openai.OpenAIUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	adapter := openai.NewAdapter("test-key", openai.WithBaseURL(server.URL))

	req := models.ProviderRequest{
		APIType: "chat",
		Model:   "gpt-4",
		Payload: models.ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []models.Message{
				{Role: "user", Content: "Hello"},
			},
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := adapter.Execute(ctx, req, "gpt-4")
		if err != nil {
			b.Fatalf("Execute failed: %v", err)
		}
	}
}

// BenchmarkEndToEndStreamingLatency simulates streaming request latency.
func BenchmarkEndToEndStreamingLatency(b *testing.B) {
	// Create a mock server that streams
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()

		// Send a few chunks
		for i := 0; i < 5; i++ {
			chunk := openai.OpenAIStreamResponse{
				ID:      fmt.Sprintf("chatcmpl-%d", i),
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   "gpt-4",
				Choices: []openai.OpenAIStreamChoice{
					{
						Index: 0,
						Delta: openai.OpenAIMessageDelta{
							Content: fmt.Sprintf("chunk%d ", i),
						},
					},
				},
			}
			data, _ := json.Marshal(chunk)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.(http.Flusher).Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	adapter := openai.NewAdapter("test-key", openai.WithBaseURL(server.URL))

	req := models.ProviderRequest{
		APIType: "chat",
		Model:   "gpt-4",
		Payload: models.ChatCompletionRequest{
			Model:    "gpt-4",
			Stream:   true,
			Messages: []models.Message{{Role: "user", Content: "Hello"}},
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := adapter.Execute(ctx, req, "gpt-4")
		if err != nil {
			b.Fatalf("Execute failed: %v", err)
		}

		// Read streaming response
		if stream, ok := result.Payload.(io.ReadCloser); ok {
			io.Copy(io.Discard, stream)
			stream.Close()
		}
	}
}

// BenchmarkJSONMarshalUnmarshal benchmarks JSON operations.
func BenchmarkJSONMarshalUnmarshal(b *testing.B) {
	data := map[string]interface{}{
		"model": "gpt-4",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello"},
		},
		"temperature": 0.7,
		"max_tokens":  150,
	}

	b.Run("Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(data)
			if err != nil {
				b.Fatalf("Marshal failed: %v", err)
			}
		}
	})

	jsonData, _ := json.Marshal(data)
	b.Run("Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var result map[string]interface{}
			err := json.Unmarshal(jsonData, &result)
			if err != nil {
				b.Fatalf("Unmarshal failed: %v", err)
			}
		}
	})
}

// BenchmarkHTTPRequestCreation benchmarks HTTP request creation.
func BenchmarkHTTPRequestCreation(b *testing.B) {
	payload := []byte(`{"model":"gpt-4","messages":[{"role":"user","content":"Hello"}]}`)
	url := "https://api.openai.com/v1/chat/completions"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			b.Fatalf("Request creation failed: %v", err)
		}
		req.Header.Set("Authorization", "Bearer test-key")
		req.Header.Set("Content-Type", "application/json")
		_ = req
	}
}

// BenchmarkHTTPResponseReading benchmarks HTTP response reading.
func BenchmarkHTTPResponseReading(b *testing.B) {
	respBody := []byte(`{"id":"chatcmpl-test","object":"chat.completion","model":"gpt-4","choices":[{"index":0,"message":{"role":"assistant","content":"Hello"},"finish_reason":"stop"}]}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		resp := &http.Response{
			Body:       io.NopCloser(bytes.NewReader(respBody)),
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
		}
		b.StartTimer()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			b.Fatalf("Read failed: %v", err)
		}
		_ = body
	}
}
