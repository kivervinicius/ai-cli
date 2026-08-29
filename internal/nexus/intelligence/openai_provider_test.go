package intelligence

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIProviderUnavailableReturnsErrorInsteadOfFallback(t *testing.T) {
	p := NewOpenAIProvider("https://api.example.invalid/v1", "", "test-model")
	_, err := p.AnalyzeIntent(context.Background(), "build product", nil)
	if !errors.Is(err, ErrIntelligenceUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}

func TestOpenAIProviderHTTPFailureIsNotConvertedToFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "provider unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "test-key", "test-model")
	_, err := p.AnalyzeIntent(context.Background(), "build product", nil)
	if err == nil {
		t.Fatal("expected provider HTTP error, got nil")
	}
}

func TestOpenAIPlanGenerationCannotMintMaestroSkills(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"packages\":[{\"title\":\"Backend\",\"goal\":\"Build API\",\"priority\":\"HIGH\",\"dependencies\":[],\"role\":\"implementer\",\"skills\":[\"skill-fabricated\"],\"acceptance\":[\"tests pass\"]}]}"}}]}`))
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, "test-key", "test-model")
	pkgs, err := p.GeneratePlanOutline(context.Background(), &IntentAnalysis{Intent: "Build API"}, nil)
	if err != nil {
		t.Fatalf("generate plan: %v", err)
	}
	if len(pkgs) != 1 {
		t.Fatalf("expected one package, got %d", len(pkgs))
	}
	if len(pkgs[0].Skills) != 0 {
		t.Fatalf("intelligence must not mint Maestro skills, got %#v", pkgs[0].Skills)
	}
}
