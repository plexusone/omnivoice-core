package verify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grokify/mogo/time/duration"

	"github.com/plexusone/omnivoice-core/stt"
)

// fakeProvider returns a fixed hearing and records the key words it was given.
type fakeProvider struct {
	name     string
	text     string
	err      error
	keywords *[]string
}

func (f fakeProvider) Name() string { return f.name }

func (f fakeProvider) Transcribe(_ context.Context, _ []byte, cfg stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	if f.keywords != nil {
		*f.keywords = cfg.Keywords
	}
	if f.err != nil {
		return nil, f.err
	}
	return &stt.TranscriptionResult{Text: f.text}, nil
}

func (f fakeProvider) TranscribeFile(ctx context.Context, _ string, cfg stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	return f.Transcribe(ctx, nil, cfg)
}

func (f fakeProvider) TranscribeURL(ctx context.Context, _ string, cfg stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	return f.Transcribe(ctx, nil, cfg)
}

func engine(name, text string) Engine {
	return Engine{Name: name, Provider: fakeProvider{name: name, text: text}}
}

func TestTranscribeAllKeywordsRule(t *testing.T) {
	var kwA, kwB []string
	engines := []Engine{
		{Name: "b", Provider: fakeProvider{name: "b", text: "x", keywords: &kwB}, NoKeywords: true},
		{Name: "a", Provider: fakeProvider{name: "a", text: "x", keywords: &kwA}},
	}
	hs, err := TranscribeAll(context.Background(), engines, []byte("wav"), []string{"Magerramov"})
	if err != nil {
		t.Fatal(err)
	}
	if hs[0].Engine != "a" || hs[1].Engine != "b" {
		t.Errorf("not sorted by engine: %v", hs)
	}
	if len(kwA) != 1 || len(kwB) != 0 {
		t.Errorf("keywords a=%v b=%v", kwA, kwB)
	}
	engines[0].NoKeywords = false
	if _, err := TranscribeAll(context.Background(), engines, nil, []string{"x"}); !errors.Is(err, ErrAllEnginesUseKeywords) {
		t.Errorf("want ErrAllEnginesUseKeywords, got %v", err)
	}
}

func TestVerifyQuotes(t *testing.T) {
	audio := filepath.Join(t.TempDir(), "ep.mp3")
	if err := os.WriteFile(audio, []byte("not really audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	index := words("so how do we actually speed up this workflow to the point where we can get an answer in a few minutes and that", 500*time.Millisecond)
	var clipped [2]time.Duration
	v := Verifier{
		Engines: []Engine{
			engine("deepgram", "speed up this workflow to the point where we can get an answer in a few minutes"),
			engine("elevenlabs", "we actually speed up this workflow to the point where we can get an answer in a few minutes and"),
			{Name: "openai", Provider: fakeProvider{name: "openai", err: errors.New("rate limited")}},
			engine("whisper", "speed up this work flow to the point where we can get an answer in a few minutes"),
		},
		Clip: func(_ context.Context, _ string, s, e time.Duration) ([]byte, error) {
			clipped = [2]time.Duration{s, e}
			return []byte("wav"), nil
		},
	}
	rep, err := v.VerifyQuotes(context.Background(), audio, index, "deepgram/nova-3", []Quote{
		{ID: "q1", Text: "speed up this workflow to the point where we can get an answer in a few minutes"},
		{ID: "q2", Text: "we rewrote the whole of Bedrock in a month"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// One engine failed and one heard "work flow"; three engines remain and agree.
	if got := rep.Results[0].Comparison.Decision; got != DecisionAgreed {
		t.Errorf("q1 decision %s, want agreed; %+v", got, rep.Results[0].Comparison.Differences)
	}
	if rep.Results[1].Comparison.Decision != DecisionNotFound {
		t.Errorf("q2 decision %s, want notFound", rep.Results[1].Comparison.Decision)
	}
	if clipped[0] != 2500*time.Millisecond-1500*time.Millisecond {
		t.Errorf("clip start %v", clipped[0])
	}
	if rep.Counts[DecisionNotFound] != 1 || len(rep.Audio.SHA256) != 64 || !strings.Contains(rep.Tool, "stt/verify") {
		t.Errorf("report %+v", rep)
	}
}

func TestVerifyQuoteWidensForMissingEdges(t *testing.T) {
	var clips [][2]time.Duration
	full := "they could impact other engineers who are doing testing"
	short := "they could impact other"
	heard := func(end time.Duration) string {
		if end > 10*time.Second {
			return full
		}
		return short
	}
	mk := func(name string) Engine {
		return Engine{Name: name, Provider: endAware{name: name, heard: heard, end: new(time.Duration)}}
	}
	es := []Engine{mk("a"), mk("b"), mk("c")}
	v := Verifier{Engines: es, Clip: func(_ context.Context, _ string, s, e time.Duration) ([]byte, error) {
		clips = append(clips, [2]time.Duration{s, e})
		for _, en := range es {
			*en.Provider.(endAware).end = e
		}
		return []byte("wav"), nil
	}}
	r := v.VerifyQuote(context.Background(), "x.mp3", nil, Quote{ID: "q", Text: "they could impact other engineers doing testing",
		Start: duration.FromDuration(2 * time.Second), End: duration.FromDuration(5 * time.Second)})
	if r.Widened == 0 || len(clips) < 2 || r.Comparison.MissingEnd != 0 {
		t.Fatalf("widened %d clips %v comparison %+v", r.Widened, clips, r.Comparison)
	}
	// "who are" is a run of extra function words all engines heard.
	if r.Comparison.Decision != DecisionCorrected {
		t.Errorf("decision %s, want corrected", r.Comparison.Decision)
	}
}

// endAware hears more of the quote when the clip ends later.
type endAware struct {
	name  string
	heard func(time.Duration) string
	end   *time.Duration
}

func (e endAware) Name() string { return e.name }
func (e endAware) Transcribe(context.Context, []byte, stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	return &stt.TranscriptionResult{Text: e.heard(*e.end)}, nil
}
func (e endAware) TranscribeFile(ctx context.Context, _ string, c stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	return e.Transcribe(ctx, nil, c)
}
func (e endAware) TranscribeURL(ctx context.Context, _ string, c stt.TranscriptionConfig) (*stt.TranscriptionResult, error) {
	return e.Transcribe(ctx, nil, c)
}

func TestSearch(t *testing.T) {
	ts := []IndexedTranscript{
		{Engine: "deepgram", Words: words("we can have the model run all the tests and catch integration bugs", time.Second)},
		{Engine: "whisper", Words: words("we could have the model run all the tests and catch integration bugs", time.Second)},
	}
	r := Search(ts, []string{"run the whole stack locally", "on my laptop"}, AbsenceOptions{})
	if r.Phrases[0].FoundIn != 0 || r.Phrases[1].FoundIn != 0 {
		t.Fatalf("found %+v", r.Phrases)
	}
	if !strings.Contains(r.Statement, "2 independent transcripts") {
		t.Errorf("statement %q", r.Statement)
	}
	r = Search(ts, []string{"catch integration bugs"}, AbsenceOptions{})
	if r.Phrases[0].FoundIn != 2 {
		t.Errorf("found in %d, want 2", r.Phrases[0].FoundIn)
	}
}

func TestTranscribeAllClearsConfiguredKeywordsOnPlainEngine(t *testing.T) {
	var got []string
	e := Engine{Name: "plain", NoKeywords: true,
		Provider: fakeProvider{name: "plain", text: "x", keywords: &got},
		Config:   stt.TranscriptionConfig{Keywords: []string{"Configured"}}}
	if _, err := TranscribeAll(context.Background(), []Engine{e}, nil, []string{"Mantle"}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("plain engine received key words %v", got)
	}
}

func TestVerifyQuoteNotFoundHasEmptyEngines(t *testing.T) {
	v := Verifier{Engines: []Engine{engine("a", "x")}}
	r := v.VerifyQuote(context.Background(), "x.mp3", words("nothing like it", time.Second), Quote{ID: "q", Text: "we rewrote everything in a month"})
	if r.Comparison.Decision != DecisionNotFound || r.Comparison.Engines == nil {
		t.Errorf("result %+v", r.Comparison)
	}
}
