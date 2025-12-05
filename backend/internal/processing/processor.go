package processing

import (
	"crypto/sha1"
	"encoding/hex"
	"html"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

var (
	whitespace  = regexp.MustCompile(`\s+`)
	punctuation = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
)

var stopwords = map[string]struct{}{
	"и": {}, "в": {}, "на": {}, "с": {}, "по": {}, "к": {},
	"a": {}, "an": {}, "the": {}, "to": {}, "in": {}, "for": {},
	"что": {}, "как": {}, "это": {}, "из": {}, "от": {}, "до": {},
}

// CleanText strips HTML entities, punctuation, and squeezes whitespace.
func CleanText(input string) string {
	if input == "" {
		return ""
	}
	decoded := html.UnescapeString(input)
	decoded = punctuation.ReplaceAllString(decoded, " ")
	decoded = whitespace.ReplaceAllString(decoded, " ")
	decoded = strings.TrimSpace(decoded)
	return decoded
}

// ExtractKeywords returns the most frequent words that are not stop-words.
func ExtractKeywords(text string, limit, minLen int) []string {
	clean := strings.ToLower(CleanText(text))
	if clean == "" {
		return nil
	}

	freq := make(map[string]int)
	for _, token := range strings.Fields(clean) {
		token = strings.TrimFunc(token, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
		if len([]rune(token)) < minLen {
			continue
		}
		if _, skip := stopwords[token]; skip {
			continue
		}
		freq[token]++
	}

	if len(freq) == 0 {
		return nil
	}

	type kv struct {
		word  string
		count int
	}

	pairs := make([]kv, 0, len(freq))
	for word, count := range freq {
		pairs = append(pairs, kv{word: word, count: count})
	}

	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count == pairs[j].count {
			return pairs[i].word < pairs[j].word
		}
		return pairs[i].count > pairs[j].count
	})

	max := limit
	if max <= 0 || max > len(pairs) {
		max = len(pairs)
	}

	keywords := make([]string, 0, max)
	for i := 0; i < max; i++ {
		keywords = append(keywords, pairs[i].word)
	}

	return keywords
}

// BuildDocumentID hashes the most stable fields to form deterministic IDs.
func BuildDocumentID(title, text string, ts time.Time) string {
	s := sha1.Sum([]byte(title + "|" + text + "|" + ts.UTC().Format(time.RFC3339)))
	return hex.EncodeToString(s[:])
}
