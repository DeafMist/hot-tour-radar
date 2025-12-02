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

func TestBuildDocumentID(t *testing.T) {
	ts := time.Date(2024, 2, 3, 4, 5, 6, 0, time.UTC)
	id1 := processing.BuildDocumentID("title", "text", ts)
	id2 := processing.BuildDocumentID("title", "text", ts)
	require.NotEmpty(t, id1)
	require.Equal(t, id1, id2)
}
