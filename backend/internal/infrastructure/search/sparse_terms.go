package search

import (
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"unicode"
)

const sparseKeywordVectorName = "keywords"
const sparseKeywordSpace uint32 = 1 << 20

type sparseToken struct {
	index uint32
	value float32
}

func buildSparseKeywordVector(text string) ([]uint32, []float32) {
	tokens := tokenizeSparseTerms(text)
	if len(tokens) == 0 {
		return nil, nil
	}

	weights := make(map[uint32]float64, len(tokens))
	for _, token := range tokens {
		if token == "" {
			continue
		}
		index := sparseTermIndex(token)
		weights[index] += sparseTermWeight(token)
	}
	if len(weights) == 0 {
		return nil, nil
	}

	items := make([]sparseToken, 0, len(weights))
	var norm float64
	for index, weight := range weights {
		norm += weight * weight
		items = append(items, sparseToken{index: index, value: float32(weight)})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].index < items[j].index
	})

	if norm == 0 {
		return nil, nil
	}
	scale := float32(1 / math.Sqrt(norm))
	indices := make([]uint32, 0, len(items))
	values := make([]float32, 0, len(items))
	for _, item := range items {
		indices = append(indices, item.index)
		values = append(values, item.value*scale)
	}
	return indices, values
}

func tokenizeSparseTerms(text string) []string {
	splitter := func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}
	rawTokens := strings.FieldsFunc(strings.ToLower(text), splitter)
	tokens := make([]string, 0, len(rawTokens))
	for _, token := range rawTokens {
		token = strings.TrimSpace(token)
		if len(token) < 2 {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens
}

func sparseTermIndex(token string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(token))
	return h.Sum32() % sparseKeywordSpace
}

func sparseTermWeight(token string) float64 {
	weight := 1.0
	if len(token) >= 4 {
		weight += 0.15
	}
	if containsDigit(token) {
		weight += 0.25
	}
	return weight
}

func containsDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}
