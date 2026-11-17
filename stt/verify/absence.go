package verify

import (
	"fmt"
	"sort"
)

// IndexedTranscript is a full transcript from one engine, for searching.
type IndexedTranscript struct {
	Engine string
	Model  string
	Words  []TimedWord
}

// AbsenceOptions configures Search.
type AbsenceOptions struct {
	// MinScore is the lowest location score that counts as found. Zero
	// means 0.8.
	MinScore  float64
	Normalize NormalizeOptions
}

// PhraseSearch is what one transcript holds for one phrase.
type PhraseSearch struct {
	Engine string `json:"engine"`
	Model  string `json:"model,omitempty"`
	// Best is the closest match, found or not.
	Best  *Span `json:"best,omitempty"`
	Found bool  `json:"found"`
}

// PhraseResult is the search for one phrase across every transcript.
type PhraseResult struct {
	Phrase   string         `json:"phrase"`
	Searches []PhraseSearch `json:"searches"`
	// FoundIn counts the transcripts the phrase was found in.
	FoundIn int `json:"found_in"`
}

// AbsenceResult is the outcome of Search.
type AbsenceResult struct {
	// Transcripts counts the independent transcripts searched.
	Transcripts int            `json:"transcripts"`
	Phrases     []PhraseResult `json:"phrases"`
	// Statement says what the search shows, in words a claim can use. It
	// never says a phrase is absent from the recording, only that it was not
	// found in the transcripts.
	Statement string `json:"statement"`
}

// Search looks for each phrase in every transcript. Use transcripts from
// independent engines: one engine's transcript can miss a passage.
func Search(transcripts []IndexedTranscript, phrases []string, opts AbsenceOptions) AbsenceResult {
	minScore := opts.MinScore
	if minScore == 0 {
		minScore = 0.8
	}
	ts := append([]IndexedTranscript(nil), transcripts...)
	sort.SliceStable(ts, func(i, j int) bool { return ts[i].Engine < ts[j].Engine })
	res := AbsenceResult{Transcripts: len(ts)}
	foundAny := false
	for _, p := range phrases {
		pr := PhraseResult{Phrase: p}
		for _, t := range ts {
			ps := PhraseSearch{Engine: t.Engine, Model: t.Model}
			if span, ok := Locate(t.Words, p, opts.Normalize); ok {
				ps.Best = &span
				ps.Found = span.Score >= minScore
			}
			if ps.Found {
				pr.FoundIn++
			}
			pr.Searches = append(pr.Searches, ps)
		}
		if pr.FoundIn > 0 {
			foundAny = true
		}
		res.Phrases = append(res.Phrases, pr)
	}
	switch {
	case foundAny:
		res.Statement = "At least one phrase was found; see each phrase's searches."
	default:
		res.Statement = fmt.Sprintf("None of the %d phrases was found in %d independent transcripts.", len(phrases), len(ts))
	}
	return res
}
