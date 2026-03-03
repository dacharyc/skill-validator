package judge

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUseMaxCompletionTokens(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		// Legacy models use max_tokens
		{"gpt-4o", false},
		{"gpt-4o-mini", false},
		{"gpt-4-turbo", false},
		{"gpt-3.5-turbo", false},
		// GPT-5+ models use max_completion_tokens
		{"gpt-5", true},
		{"gpt-5.2", true},
		{"gpt-5-turbo", true},
		// O-series reasoning models use max_completion_tokens
		{"o1", true},
		{"o1-mini", true},
		{"o3", true},
		{"o3-mini", true},
		{"o4-mini", true},
		// Case insensitivity
		{"GPT-5.2", true},
		{"O3-mini", true},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := useMaxCompletionTokens(tt.model)
			if got != tt.want {
				t.Errorf("useMaxCompletionTokens(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}

func TestMaxTokensStyleOverride(t *testing.T) {
	tests := []struct {
		name           string
		model          string
		maxTokensStyle string
		wantMaxTokens  bool // true = max_tokens field set; false = max_completion_tokens field set
	}{
		{
			name:           "auto with legacy model uses max_tokens",
			model:          "gpt-4o",
			maxTokensStyle: "auto",
			wantMaxTokens:  true,
		},
		{
			name:           "auto with o-series uses max_completion_tokens",
			model:          "o3-mini",
			maxTokensStyle: "auto",
			wantMaxTokens:  false,
		},
		{
			name:           "explicit max_tokens overrides o-series detection",
			model:          "o3-mini",
			maxTokensStyle: "max_tokens",
			wantMaxTokens:  true,
		},
		{
			name:           "explicit max_completion_tokens overrides legacy detection",
			model:          "gpt-4o",
			maxTokensStyle: "max_completion_tokens",
			wantMaxTokens:  false,
		},
		{
			name:           "empty string defaults to auto behavior",
			model:          "gpt-4o",
			maxTokensStyle: "",
			wantMaxTokens:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedBody openaiRequest

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
					t.Fatalf("decoding request body: %v", err)
				}
				resp := openaiResponse{
					Choices: []struct {
						Message struct {
							Content string `json:"content"`
						} `json:"message"`
					}{
						{Message: struct {
							Content string `json:"content"`
						}{Content: `{"score": 4}`}},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(resp); err != nil {
					t.Errorf("encoding response: %v", err)
				}
			}))
			defer srv.Close()

			client, err := NewClient("openai", "test-key", srv.URL, tt.model, tt.maxTokensStyle)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}

			_, err = client.Complete(t.Context(), "system", "user")
			if err != nil {
				t.Fatalf("Complete: %v", err)
			}

			if tt.wantMaxTokens {
				if capturedBody.MaxTokens == 0 {
					t.Error("expected max_tokens to be set, but it was 0")
				}
				if capturedBody.MaxCompletionTokens != 0 {
					t.Errorf("expected max_completion_tokens to be 0, got %d", capturedBody.MaxCompletionTokens)
				}
			} else {
				if capturedBody.MaxCompletionTokens == 0 {
					t.Error("expected max_completion_tokens to be set, but it was 0")
				}
				if capturedBody.MaxTokens != 0 {
					t.Errorf("expected max_tokens to be 0, got %d", capturedBody.MaxTokens)
				}
			}
		})
	}
}
