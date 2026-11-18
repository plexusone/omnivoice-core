package verify

import (
	"strings"
	"time"

	"github.com/grokify/mogo/time/duration"

	"github.com/plexusone/omnivoice-core/stt"
)

// TimedWord is a transcribed word with its time in the recording.
type TimedWord struct {
	Text  string
	Start time.Duration
	End   time.Duration
}

// WordsFromTranscript flattens a transcript's word timestamps in order.
func WordsFromTranscript(t *stt.Transcript) []TimedWord {
	var out []TimedWord
	for _, s := range t.Segments {
		for _, w := range s.Words {
			out = append(out, TimedWord{Text: w.Text, Start: w.Start.Duration(), End: w.End.Duration()})
		}
	}
	return out
}

// Span is where a quote was found in a transcript.
type Span struct {
	// Start and End are the times of the first and last matched words.
	Start duration.DurationMilliseconds `json:"start_ms"`
	End   duration.DurationMilliseconds `json:"end_ms"`
	// FirstWord and LastWord are indexes into the transcript's words.
	FirstWord int `json:"first_word"`
	LastWord  int `json:"last_word"`
	// Score is 1 minus the edit distance divided by the quote's length in
	// tokens: 1 is an exact match.
	Score float64 `json:"score"`
	// Heard is what the transcript says over the span.
	Heard string `json:"heard"`
	// MissingStart and MissingEnd count quote tokens at the start and end
	// that matched nothing in the span. They often mean the quote leaves out
	// words without marking the gap, so its edges lie outside the span.
	MissingStart int `json:"missing_start,omitempty"`
	MissingEnd   int `json:"missing_end,omitempty"`
}

// Locate finds the stretch of words that best matches the quote, allowing
// for words the transcript misheard. It returns false when the quote has no
// tokens or the transcript has no words.
func Locate(words []TimedWord, quote string, opts NormalizeOptions) (Span, bool) {
	q := Normalize(quote, opts)
	if len(q) == 0 || len(words) == 0 {
		return Span{}, false
	}
	// Normalize the words together, so a number said across words
	// ("twenty" "five") is one token, while each token keeps its words.
	var raw []rawWord
	for wi, w := range words {
		for _, rw := range splitWords(w.Text) {
			rw.src = wi
			raw = append(raw, rw)
		}
	}
	// Reconcile word boundaries as Compare does, so a quoted "speed ups" is
	// found where the transcript has "speedups".
	hyp := joinSplitWords(normalizeWords(raw, opts), q)
	pairs, dist := AlignWithin(q, hyp)
	first, last := -1, -1
	for _, p := range pairs {
		if p.Hyp < 0 {
			continue
		}
		if first < 0 {
			first = hyp[p.Hyp].Word
		}
		last = hyp[p.Hyp].end
	}
	if first < 0 {
		return Span{}, false
	}
	missingStart, missingEnd := 0, 0
	for _, p := range pairs {
		if p.Op != OpDelete {
			break
		}
		missingStart++
	}
	for k := len(pairs) - 1; k >= 0 && pairs[k].Op == OpDelete; k-- {
		missingEnd++
	}
	heard := make([]string, 0, last-first+1)
	for _, w := range words[first : last+1] {
		heard = append(heard, w.Text)
	}
	return Span{
		Start:     duration.FromDuration(words[first].Start),
		End:       duration.FromDuration(words[last].End),
		FirstWord: first,
		LastWord:  last,
		Score:     1 - float64(dist)/float64(len(q)),
		Heard:     strings.Join(heard, " "),

		MissingStart: missingStart,
		MissingEnd:   missingEnd,
	}, true
}
