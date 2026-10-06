package main

import (
	"math"
	"testing"

	"github.com/openai/openai-go/v3/responses"
)

func TestEstimateTokenCostUSD(t *testing.T) {
	for _, tt := range []struct {
		name       string
		model      string
		input      int64
		cached     int64
		cacheWrite int64
		output     int64
		want       float64
	}{
		{name: "astra", model: "gpt-6-astra", input: 100_000, output: 10_000, want: 1.5},
		{name: "sol", model: "gpt-5.6-sol", input: 100_000, output: 10_000, want: 0.6},
		{name: "sol alias", model: "gpt-5.6", input: 100_000, output: 10_000, want: 0.6},
		{name: "dated astra", model: "gpt-6-astra-2026-10-06", input: 100_000, output: 10_000, want: 1.5},
		{name: "dated sol", model: "gpt-5.6-sol-2026-10-06", input: 100_000, output: 10_000, want: 0.6},
		{name: "cache mix", model: "gpt-6-astra", input: 100_000, cached: 20_000, cacheWrite: 30_000, output: 10_000, want: 1.395},
		{name: "all cached", model: "gpt-6-astra", input: 100_000, cached: 100_000, want: 0.1},
		{name: "all cache writes", model: "gpt-6-astra", input: 100_000, cacheWrite: 100_000, want: 1.25},
		{name: "at long input threshold", model: "gpt-6-astra", input: 272_000, output: 1_000, want: 2.77},
		{name: "over long input threshold", model: "gpt-6-astra", input: 272_001, output: 1_000, want: 5.51502},
		{name: "long input with caching", model: "gpt-6-astra", input: 272_001, cached: 20_000, cacheWrite: 30_000, output: 1_000, want: 5.30502},
		{name: "sol long input with caching", model: "gpt-5.6-sol", input: 272_001, cached: 20_000, cacheWrite: 30_000, output: 1_000, want: 2.122008},
		{name: "zero usage", model: "gpt-6-astra"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			usage := responses.ResponseUsage{
				InputTokens: tt.input,
				InputTokensDetails: responses.ResponseUsageInputTokensDetails{
					CachedTokens: tt.cached, CacheWriteTokens: tt.cacheWrite,
				},
				OutputTokens: tt.output,
			}
			got, ok := estimateTokenCostUSD(tt.model, usage)
			if !ok || math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("estimated cost = (%f, %t), want (%f, true)", got, ok, tt.want)
			}
		})
	}
}

func TestEstimateTokenCostUnknownModel(t *testing.T) {
	for _, model := range []string{"", "custom-deployment", "gpt-6-astra-pro", "gpt-6-astra-2026-99-99", "gpt-6-luna"} {
		t.Run(model, func(t *testing.T) {
			if cost, ok := estimateTokenCostUSD(model, responses.ResponseUsage{InputTokens: 100}); ok || cost != 0 {
				t.Errorf("unknown model cost = (%f, %t), want (0, false)", cost, ok)
			}
		})
	}
}
