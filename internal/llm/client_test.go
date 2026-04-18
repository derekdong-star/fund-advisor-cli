package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestOpenAICompatibleClientUsesCustomBaseURL(t *testing.T) {
	t.Parallel()
	const envName = "TEST_CUSTOM_OPENAI_KEY"
	if err := os.Setenv(envName, "secret-key"); err != nil {
		t.Fatalf("Setenv() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv(envName) })

	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.URL.String(); got != "https://custom-provider.example/v1/chat/completions" {
			t.Fatalf("url = %s, want https://custom-provider.example/v1/chat/completions", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret-key" {
			t.Fatalf("authorization header = %s", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		if got := payload["model"]; got != "custom-model" {
			t.Fatalf("model = %v, want custom-model", got)
		}
		body := "{\"choices\":[{\"message\":{\"content\":\"{\\\"rankings\\\":[{\\\"fund_code\\\":\\\"B\\\",\\\"rank\\\":1,\\\"score\\\":8.6,\\\"reason\\\":\\\"Better medium-term trend.\\\"},{\\\"fund_code\\\":\\\"A\\\",\\\"rank\\\":2,\\\"score\\\":7.9,\\\"reason\\\":\\\"Second choice.\\\"}]}\"}}]}"
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString(body)),
		}, nil
	})}

	client := newOpenAICompatibleClient(config.LLMConfig{
		Provider:       "openai",
		BaseURL:        "https://custom-provider.example/v1",
		Model:          "custom-model",
		APIKey:         "default-key",
		APIKeyEnv:      envName,
		TimeoutSeconds: 5,
	}, httpClient)

	resp, err := client.RerankCandidates(context.Background(), CandidateRerankRequest{
		PortfolioName: "test",
		RunDate:       time.Date(2026, 4, 18, 9, 0, 0, 0, time.UTC),
		Candidates: []CandidateRerankInput{
			{FundCode: "A", FundName: "Alpha", Score: 7, RuleReason: "rule alpha"},
			{FundCode: "B", FundName: "Beta", Score: 8, RuleReason: "rule beta"},
		},
	})
	if err != nil {
		t.Fatalf("RerankCandidates() error = %v", err)
	}
	if got := resp.Rankings[0].FundCode; got != "B" {
		t.Fatalf("top fund code = %s, want B", got)
	}
	if got := resp.Rankings[0].Reason; got != "Better medium-term trend." {
		t.Fatalf("top reason = %q", got)
	}
}

func TestOpenAICompatibleClientFallsBackToConfiguredAPIKey(t *testing.T) {
	t.Parallel()
	const envName = "TEST_CUSTOM_OPENAI_KEY_FALLBACK"
	t.Cleanup(func() { _ = os.Unsetenv(envName) })
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.Header.Get("Authorization"); got != "Bearer default-key" {
			t.Fatalf("authorization header = %s, want Bearer default-key", got)
		}
		body := "{\"choices\":[{\"message\":{\"content\":\"{\\\"rankings\\\":[{\\\"fund_code\\\":\\\"A\\\",\\\"rank\\\":1,\\\"score\\\":7.9,\\\"reason\\\":\\\"Fallback key used.\\\"}]}\"}}]}"
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBufferString(body)),
		}, nil
	})}
	client := newOpenAICompatibleClient(config.LLMConfig{
		Provider:       "openai",
		BaseURL:        "https://custom-provider.example/v1",
		Model:          "custom-model",
		APIKey:         "default-key",
		APIKeyEnv:      envName,
		TimeoutSeconds: 5,
	}, httpClient)
	resp, err := client.RerankCandidates(context.Background(), CandidateRerankRequest{
		Candidates: []CandidateRerankInput{{FundCode: "A", FundName: "Alpha", Score: 7, RuleReason: "rule alpha"}},
	})
	if err != nil {
		t.Fatalf("RerankCandidates() error = %v", err)
	}
	if got := resp.Rankings[0].Reason; got != "Fallback key used." {
		t.Fatalf("reason = %q, want Fallback key used.", got)
	}
}

func TestOpenAICompatibleClientRequiresEnvOrConfiguredAPIKey(t *testing.T) {
	t.Parallel()
	client := newOpenAICompatibleClient(config.LLMConfig{
		Provider:       "openai",
		BaseURL:        "https://example.invalid/v1",
		Model:          "custom-model",
		APIKeyEnv:      "MISSING_CUSTOM_OPENAI_KEY",
		TimeoutSeconds: 5,
	}, nil)
	_, err := client.RerankCandidates(context.Background(), CandidateRerankRequest{Candidates: []CandidateRerankInput{{FundCode: "A"}}})
	if err == nil {
		t.Fatalf("expected missing api key error")
	}
	if !strings.Contains(err.Error(), "MISSING_CUSTOM_OPENAI_KEY") {
		t.Fatalf("error = %v, want missing env name", err)
	}
}

func TestOpenAICompatibleClientPingUsesSamePath(t *testing.T) {
	t.Parallel()
	const envName = "TEST_CUSTOM_OPENAI_KEY_PING"
	if err := os.Setenv(envName, "secret-key"); err != nil {
		t.Fatalf("Setenv() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Unsetenv(envName) })
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if got := r.URL.String(); got != "https://custom-provider.example/v1/chat/completions" {
			t.Fatalf("url = %s, want https://custom-provider.example/v1/chat/completions", got)
		}
		body := "{\"choices\":[{\"message\":{\"content\":\"{\\\"rankings\\\":[{\\\"fund_code\\\":\\\"A\\\",\\\"rank\\\":1,\\\"score\\\":1.0,\\\"reason\\\":\\\"Ping ok.\\\"}]}\"}}]}"
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewBufferString(body))}, nil
	})}
	client := newOpenAICompatibleClient(config.LLMConfig{Provider: "openai", BaseURL: "https://custom-provider.example/v1", Model: "custom-model", APIKeyEnv: envName, TimeoutSeconds: 5}, httpClient)
	resp, err := client.Ping(context.Background(), CandidateRerankRequest{Candidates: []CandidateRerankInput{{FundCode: "A", FundName: "Alpha", Score: 1, RuleReason: "ping"}}})
	if err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	if got := resp.Rankings[0].Reason; got != "Ping ok." {
		t.Fatalf("reason = %q, want Ping ok.", got)
	}
}
