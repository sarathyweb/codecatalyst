package main

import (
	"strings"
	"time"

	"github.com/openai/openai-go/v3/responses"
)

type tokenPricing struct {
	input       float64
	cachedInput float64
	output      float64
}

// Standard USD rates per million tokens, checked October 6, 2026:
// https://developers.openai.com/api/docs/models/gpt-6-astra
// https://developers.openai.com/api/docs/models/gpt-5.6-sol
// These are estimates using OpenAI list prices; Azure pricing may differ.
var modelTokenPricing = map[string]tokenPricing{
	"gpt-6-astra": {input: 10, cachedInput: 1, output: 50},
	"gpt-5.6-sol": {input: 4, cachedInput: 0.4, output: 20},
	"gpt-5.6":     {input: 4, cachedInput: 0.4, output: 20},
}

func estimateTokenCostUSD(model string, usage responses.ResponseUsage) (float64, bool) {
	// Azure may return a dated model ID even when the deployment has a custom name.
	if len(model) > 11 && model[len(model)-11] == '-' {
		if _, err := time.Parse("2006-01-02", model[len(model)-10:]); err == nil {
			model = model[:len(model)-11]
		}
	}
	pricing, ok := modelTokenPricing[strings.ToLower(model)]
	if !ok {
		return 0, false
	}

	// Long-context rates apply to the full request, including cache reads/writes.
	if usage.InputTokens > 272_000 {
		pricing.input *= 2
		pricing.cachedInput *= 2
		pricing.output *= 1.5
	}

	cachedTokens := usage.InputTokensDetails.CachedTokens
	cacheWriteTokens := usage.InputTokensDetails.CacheWriteTokens
	uncachedTokens := usage.InputTokens - cachedTokens - cacheWriteTokens
	// Cache writes replace the ordinary input charge with a 1.25x rate.
	cost := float64(uncachedTokens)*pricing.input +
		float64(cachedTokens)*pricing.cachedInput +
		float64(cacheWriteTokens)*pricing.input*1.25 +
		float64(usage.OutputTokens)*pricing.output
	return cost / 1_000_000, true
}
