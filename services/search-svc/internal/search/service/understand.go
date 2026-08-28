// services/search-svc/internal/search/service/understand.go
package service

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/ZoroNewbie00/kalakriti/services/search-svc/internal/search/domain"
)

// pricePattern catches the way buyers actually write a budget: "under 2000",
// "below ₹1,500", "less than 3000 rupees".
var pricePattern = regexp.MustCompile(`(?i)\b(under|below|less than|upto|up to|over|above|more than)\s*(?:rs\.?|inr|₹)?\s*([0-9][0-9,]*)`)

// vocabulary is the small closed set of words worth reading as filters. It is
// deliberately not the whole ontology: a colour that is also a craft name should
// stay part of the text, and the entity linker already handles crafts.
var (
	colourWords   = []string{"indigo", "red", "blue", "green", "yellow", "black", "white", "ecru", "madder", "turmeric"}
	materialWords = []string{"cotton", "silk", "wool", "muga", "tussar", "pashmina", "brass", "clay", "quartz", "jute"}
)

// RuleUnderstander pulls the filters that can be read off the text with no model
// at all: a budget, a GI request, a posture, a colour, a material.
//
// ponytail: rules, not an LLM. Every filter here is one a regex reads correctly
// and instantly, and the pipeline's contract is that understanding is best
// effort under a 500ms budget. When ml-svc grows an UnderstandQuery RPC, it
// implements this same interface and this stays as the fallback for when it is
// slow — which is the behaviour the timeout test already pins down.
type RuleUnderstander struct{}

// NewRuleUnderstander builds the default query understander.
func NewRuleUnderstander() RuleUnderstander { return RuleUnderstander{} }

// Understand extracts what it can and returns the rest as residual text.
func (RuleUnderstander) Understand(_ context.Context, text, _ string) (domain.Filters, string, error) {
	var filters domain.Filters
	residual := text

	if match := pricePattern.FindStringSubmatch(text); match != nil {
		amount, err := strconv.ParseInt(strings.ReplaceAll(match[2], ",", ""), 10, 64)
		if err == nil {
			// The catalogue is priced in paise and buyers think in rupees.
			paise := amount * 100
			switch strings.ToLower(match[1]) {
			case "over", "above", "more than":
				filters.MinPricePaise = &paise
			default:
				filters.MaxPricePaise = &paise
			}
			residual = strings.Replace(residual, match[0], " ", 1)
		}
	}

	for _, phrase := range []string{"gi tagged", "gi certified", "gi tag"} {
		if strings.Contains(residual, phrase) {
			filters.GIOnly = true
			residual = strings.Replace(residual, phrase, " ", 1)
		}
	}
	if strings.Contains(residual, "in stock") || strings.Contains(residual, "ready stock") {
		readyStock := "READY_STOCK"
		filters.ListingType = &readyStock
		residual = strings.NewReplacer("in stock", " ", "ready stock", " ").Replace(residual)
	}

	// Colours and materials stay in the residual text as well as becoming
	// filters: they are good retrieval signal, and removing them would strand a
	// query like "indigo" with nothing to embed.
	filters.Colours = wordsPresent(residual, colourWords)
	filters.Materials = wordsPresent(residual, materialWords)

	return filters, strings.Join(strings.Fields(residual), " "), nil
}

// wordsPresent is the vocabulary members appearing in the text, as whole words.
func wordsPresent(text string, vocabulary []string) []string {
	if text == "" {
		return nil
	}
	tokens := make(map[string]struct{}, 8)
	for _, token := range domain.Tokens(text) {
		tokens[token] = struct{}{}
	}

	out := make([]string, 0, 2)
	for _, word := range vocabulary {
		if _, ok := tokens[word]; ok {
			out = append(out, word)
		}
	}
	return out
}
