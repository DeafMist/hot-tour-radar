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

var urlRegex = regexp.MustCompile(`https?://[^\s]+`)

var (
	whitespace  = regexp.MustCompile(`\s+`)
	punctuation = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
)

var stopwords = map[string]struct{}{
	"и": {}, "в": {}, "на": {}, "с": {}, "по": {}, "к": {},
	"a": {}, "an": {}, "the": {}, "to": {}, "in": {}, "for": {},
	"что": {}, "как": {}, "это": {}, "из": {}, "от": {}, "до": {},
}

// ExtractURLs extracts all HTTP(S) URLs from the input text.
func ExtractURLs(input string) []string {
	if input == "" {
		return nil
	}
	matches := urlRegex.FindAllString(input, -1)
	if len(matches) == 0 {
		return nil
	}
	// Remove duplicates while preserving order
	seen := make(map[string]struct{})
	var urls []string
	for _, url := range matches {
		if _, ok := seen[url]; !ok {
			seen[url] = struct{}{}
			urls = append(urls, url)
		}
	}
	return urls
}

// RemoveURLs removes all URLs from the input text.
func RemoveURLs(input string) string {
	return urlRegex.ReplaceAllString(input, " ")
}

// CleanText strips HTML entities, punctuation, squeezes whitespace, and removes URLs.
func CleanText(input string) string {
	if input == "" {
		return ""
	}
	decoded := html.UnescapeString(input)
	decoded = RemoveURLs(decoded)
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

// GenerateTitleFromText creates a title from the first sentence or first N words of text.
// Returns empty string if text is empty.
func GenerateTitleFromText(text string, maxWords int) string {
	if text == "" {
		return ""
	}

	// Remove URLs before finding sentence boundaries
	textWithoutURLs := RemoveURLs(text)

	// Try to find first sentence (ending with . ! ?)
	sentenceEnd := strings.IndexAny(textWithoutURLs, ".!?")
	var firstSentence string
	if sentenceEnd > 0 {
		firstSentence = strings.TrimSpace(textWithoutURLs[:sentenceEnd])
	} else {
		firstSentence = textWithoutURLs
	}

	// Split into words and limit to maxWords
	words := strings.Fields(firstSentence)
	if len(words) == 0 {
		return ""
	}

	if maxWords > 0 && len(words) > maxWords {
		words = words[:maxWords]
		// Add ellipsis if truncated
		return strings.Join(words, " ") + "..."
	}

	return strings.Join(words, " ")
}
