package processing_test

import (
	"testing"
	"time"

	"github.com/DeafMist/hot-tour-radar/backend/internal/processing"
	"github.com/stretchr/testify/require"
)

func TestCleanText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "punctuation", input: "Hello!!!   мир", want: "Hello мир"},
		{name: "collapse whitespace", input: "foo\n\nbar\t baz", want: "foo bar baz"},
		{name: "remove urls", input: "Check https://example.com for info", want: "Check for info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := processing.CleanText(tt.input); got != tt.want {
				t.Fatalf("CleanText(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractKeywords(t *testing.T) {
	text := "Тур Тур поездка поездка поездка море и и солнце"
	got := processing.ExtractKeywords(text, 3, 3)
	want := []string{"поездка", "тур", "море"}
	require.Equal(t, want, got)

	require.Nil(t, processing.ExtractKeywords("", 5, 3))
}

func TestExtractKeywordsIgnoresURLWords(t *testing.T) {
	// Text with URL should not include URL domain/path words in keywords
	text := "Тур поездка поездка https://example.com/tour-deals море"
	got := processing.ExtractKeywords(text, 3, 3)
	// Should extract: поездка, тур, море (NOT example, com, tour, deals)
	// поездка appears 2 times, тур and море appear 1 time each
	require.ElementsMatch(t, []string{"поездка", "тур", "море"}, got)
}

func TestBuildDocumentID(t *testing.T) {
	ts := time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)
	id1 := processing.BuildDocumentID("title", "text", ts)
	id2 := processing.BuildDocumentID("title", "text", ts)
	require.NotEmpty(t, id1)
	require.Equal(t, id1, id2)
}

func TestExtractURLs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "empty", input: "", want: nil},
		{name: "no urls", input: "Hello world", want: nil},
		{name: "single url", input: "Check https://example.com for more", want: []string{"https://example.com"}},
		{name: "multiple urls", input: "Go to https://example.com or http://test.org now", want: []string{"https://example.com", "http://test.org"}},
		{name: "duplicate urls", input: "https://example.com and https://example.com again", want: []string{"https://example.com"}},
		{name: "urls with path", input: "Visit https://example.com/path/to/page for details", want: []string{"https://example.com/path/to/page"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processing.ExtractURLs(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRemoveURLs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "no urls", input: "Hello world", want: "Hello world"},
		{name: "single url", input: "Check https://example.com for more", want: "Check   for more"},
		{name: "multiple urls", input: "Go https://example.com and http://test.org now", want: "Go   and   now"},
		{name: "url only", input: "https://example.com", want: " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processing.RemoveURLs(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGenerateTitleFromText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxWords int
		want     string
	}{
		{name: "empty", text: "", maxWords: 10, want: ""},
		{name: "single sentence", text: "Отличный тур в Турцию.", maxWords: 10, want: "Отличный тур в Турцию"},
		{name: "multiple sentences", text: "Горящий тур в Египет! Всего 30000 рублей. Вылет завтра.", maxWords: 10, want: "Горящий тур в Египет"},
		{name: "long text truncated", text: "Супер предложение по турам в разные страны мира с большими скидками", maxWords: 5, want: "Супер предложение по турам в..."},
		{name: "no sentence end", text: "Тур в Грецию со скидкой", maxWords: 10, want: "Тур в Грецию со скидкой"},
		{name: "question mark", text: "Хотите в отпуск? Звоните нам!", maxWords: 10, want: "Хотите в отпуск"},
		{name: "unlimited words", text: "Отличное предложение по турам", maxWords: 0, want: "Отличное предложение по турам"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processing.GenerateTitleFromText(tt.text, tt.maxWords)
			require.Equal(t, tt.want, got)
		})
	}
}
