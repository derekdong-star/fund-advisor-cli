package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
)

var ErrClientUnavailable = errors.New("llm client unavailable")

func NewClient(cfg config.LLMConfig) Client {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "mock":
		return mockClient{}
	case "openai":
		return newOpenAICompatibleClient(cfg, nil)
	default:
		return unsupportedClient{reason: fmt.Sprintf("provider %q is not implemented", cfg.Provider)}
	}
}

type unsupportedClient struct {
	reason string
}

func (c unsupportedClient) RerankCandidates(context.Context, CandidateRerankRequest) (*CandidateRerankResponse, error) {
	if strings.TrimSpace(c.reason) == "" {
		return nil, ErrClientUnavailable
	}
	return nil, fmt.Errorf("%w: %s", ErrClientUnavailable, c.reason)
}

func (c unsupportedClient) Ping(ctx context.Context, req CandidateRerankRequest) (*CandidateRerankResponse, error) {
	return c.RerankCandidates(ctx, req)
}

type mockClient struct{}

func (mockClient) RerankCandidates(_ context.Context, req CandidateRerankRequest) (*CandidateRerankResponse, error) {
	items := append([]CandidateRerankInput(nil), req.Candidates...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		if items[i].Return60D != items[j].Return60D {
			return items[i].Return60D > items[j].Return60D
		}
		if items[i].Return20D != items[j].Return20D {
			return items[i].Return20D > items[j].Return20D
		}
		return items[i].FundCode < items[j].FundCode
	})
	resp := &CandidateRerankResponse{Rankings: make([]CandidateRanking, 0, len(items))}
	for idx, item := range items {
		reason := fmt.Sprintf("LLM rerank keeps %s near the top because rule score=%d, 60D=%.2f%%, 20D=%.2f%%.", item.FundName, item.Score, item.Return60D*100, item.Return20D*100)
		resp.Rankings = append(resp.Rankings, CandidateRanking{
			FundCode: item.FundCode,
			Rank:     idx + 1,
			Score:    float64(item.Score) + item.Return60D*10 + item.Return20D*5,
			Reason:   reason,
		})
	}
	return resp, nil
}

func (mockClient) Ping(ctx context.Context, req CandidateRerankRequest) (*CandidateRerankResponse, error) {
	return mockClient{}.RerankCandidates(ctx, req)
}

type openAICompatibleClient struct {
	baseURL    string
	model      string
	apiKey     string
	apiKeyEnv  string
	httpClient *http.Client
}

func newOpenAICompatibleClient(cfg config.LLMConfig, httpClient *http.Client) Client {
	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second}
	}
	return &openAICompatibleClient{
		baseURL:    strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		model:      strings.TrimSpace(cfg.Model),
		apiKey:     strings.TrimSpace(cfg.APIKey),
		apiKeyEnv:  strings.TrimSpace(cfg.APIKeyEnv),
		httpClient: client,
	}
}

type openAIChatCompletionRequest struct {
	Model       string              `json:"model"`
	Temperature float64             `json:"temperature,omitempty"`
	Messages    []openAIChatMessage `json:"messages"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatCompletionResponse struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *openAICompatibleClient) RerankCandidates(ctx context.Context, req CandidateRerankRequest) (*CandidateRerankResponse, error) {
	if strings.TrimSpace(c.baseURL) == "" {
		return nil, fmt.Errorf("%w: base URL is empty", ErrClientUnavailable)
	}
	if strings.TrimSpace(c.model) == "" {
		return nil, fmt.Errorf("%w: model is empty", ErrClientUnavailable)
	}
	apiKey, err := c.resolveAPIKey()
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(openAIChatCompletionRequest{
		Model:       c.model,
		Temperature: 0,
		Messages: []openAIChatMessage{
			{Role: "system", Content: CandidateRerankSystemPrompt()},
			{Role: "user", Content: BuildCandidateRerankPrompt(req)},
		},
	})
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var envelope openAIChatCompletionResponse
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("decode openai-compatible response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		message := strings.TrimSpace(resp.Status)
		if envelope.Error != nil && strings.TrimSpace(envelope.Error.Message) != "" {
			message = envelope.Error.Message
		}
		return nil, fmt.Errorf("openai-compatible provider error: %s", message)
	}
	if len(envelope.Choices) == 0 {
		return nil, fmt.Errorf("openai-compatible provider returned no choices")
	}
	jsonPayload, err := extractJSONObject(envelope.Choices[0].Message.Content)
	if err != nil {
		return nil, err
	}
	var result CandidateRerankResponse
	if err := json.Unmarshal([]byte(jsonPayload), &result); err != nil {
		return nil, fmt.Errorf("decode candidate rerank payload: %w", err)
	}
	return &result, nil
}

func (c *openAICompatibleClient) resolveAPIKey() (string, error) {
	if envName := strings.TrimSpace(c.apiKeyEnv); envName != "" {
		if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
			return value, nil
		}
	}
	if strings.TrimSpace(c.apiKey) != "" {
		return strings.TrimSpace(c.apiKey), nil
	}
	if strings.TrimSpace(c.apiKeyEnv) != "" {
		return "", fmt.Errorf("%w: api key env %s is empty and no llm.api_key fallback is configured", ErrClientUnavailable, c.apiKeyEnv)
	}
	return "", fmt.Errorf("%w: llm.api_key and llm.api_key_env are both empty", ErrClientUnavailable)
}

func (c *openAICompatibleClient) Ping(ctx context.Context, req CandidateRerankRequest) (*CandidateRerankResponse, error) {
	return c.RerankCandidates(ctx, req)
}

func extractJSONObject(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return "", fmt.Errorf("llm response does not contain a json object")
	}
	return raw[start : end+1], nil
}
