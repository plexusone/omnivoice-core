package verify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"time"

	"github.com/grokify/mogo/time/duration"

	"github.com/plexusone/omnivoice-core/audio/clip"
)

// Quote is wording attributed to a recording.
type Quote struct {
	// ID identifies the quote in the caller's records, such as a claim's
	// evidence ID.
	ID   string `json:"id"`
	Text string `json:"text"`
	// Start and End are where the quote is in the recording, if known. When
	// both are zero, the quote is located in the index transcript.
	Start duration.DurationMilliseconds `json:"start_ms,omitempty"`
	End   duration.DurationMilliseconds `json:"end_ms,omitempty"`
}

// Options configures a Verifier.
type Options struct {
	// Pad is added before and after the quote's span when clipping.
	// Zero means 1.5 seconds.
	Pad duration.DurationMilliseconds `json:"pad_ms"`
	// MinLocateScore is the lowest location score accepted. Zero means 0.6.
	MinLocateScore float64 `json:"min_locate_score"`
	// Keywords are known names and terms given to engines not marked
	// NoKeywords. Never pass the quote's own wording.
	Keywords []string `json:"keywords,omitempty"`
	// Compare configures the comparison.
	Compare CompareOptions `json:"-"`
}

func (o Options) pad() time.Duration {
	if o.Pad == 0 {
		return 1500 * time.Millisecond
	}
	return o.Pad.Duration()
}

func (o Options) minScore() float64 {
	if o.MinLocateScore == 0 {
		return 0.6
	}
	return o.MinLocateScore
}

// edgeWiden is how far to widen a clip for n quote tokens missing at one
// edge: about half a second per word plus three seconds, at most twenty.
func edgeWiden(n int) time.Duration {
	if n == 0 {
		return 0
	}
	d := 3*time.Second + time.Duration(n)*500*time.Millisecond
	if d > 20*time.Second {
		d = 20 * time.Second
	}
	return d
}

// ClipFunc cuts a WAV clip from a recording. clip.Clip is the default.
type ClipFunc func(ctx context.Context, path string, start, end time.Duration) ([]byte, error)

// Verifier checks quotes against one recording.
type Verifier struct {
	Engines []Engine
	Options Options
	// Clip cuts clips; nil means clip.Clip.
	Clip ClipFunc
}

// QuoteResult is the outcome for one quote.
type QuoteResult struct {
	Quote Quote `json:"quote"`
	// Located is where the quote was found in the index transcript, when it
	// had no times.
	Located *Span `json:"located,omitempty"`
	// ClipStart and ClipEnd are the clip the engines heard, padding included.
	ClipStart  duration.DurationMilliseconds `json:"clip_start_ms"`
	ClipEnd    duration.DurationMilliseconds `json:"clip_end_ms"`
	Comparison Comparison                    `json:"comparison"`
	// Widened counts how many times the clip was widened because no engine
	// heard the quote's first or last words.
	Widened int `json:"widened,omitempty"`
	// Err records a failure that stopped the check, such as a clip error.
	Err string `json:"error,omitempty"`
}

// VerifyQuote locates the quote (when it has no times), clips it, has every
// engine transcribe the clip, and compares what they heard with the quote.
func (v Verifier) VerifyQuote(ctx context.Context, audioPath string, index []TimedWord, q Quote) QuoteResult {
	r := QuoteResult{Quote: q, Comparison: Comparison{Expected: q.Text}}
	start, end := q.Start.Duration(), q.End.Duration()
	if start == 0 && end == 0 {
		span, ok := Locate(index, q.Text, v.Options.Compare.Normalize)
		if ok {
			r.Located = &span
		}
		if !ok || span.Score < v.Options.minScore() {
			r.Comparison.Decision = DecisionNotFound
			return r
		}
		start, end = span.Start.Duration(), span.End.Duration()
		// Quote words missing at an edge are usually said a little further
		// on, past a gap the quote doesn't mark. Widen the clip so the
		// engines hear them, rather than reporting them as never said.
		start -= edgeWiden(span.MissingStart)
		end += edgeWiden(span.MissingEnd)
	}
	start -= v.Options.pad()
	if start < 0 {
		start = 0
	}
	end += v.Options.pad()
	cut := v.Clip
	if cut == nil {
		cut = clip.Clip
	}
	// Words no engine heard at an edge may lie just outside the clip:
	// widen that edge and listen again, up to maxWiden times.
	const maxWiden = 2
	for attempt := 0; ; attempt++ {
		r.ClipStart, r.ClipEnd = duration.FromDuration(start), duration.FromDuration(end)
		wav, err := cut(ctx, audioPath, start, end)
		if err != nil {
			r.Err = err.Error()
			r.Comparison.Decision = DecisionInsufficient
			return r
		}
		hearings, err := TranscribeAll(ctx, v.Engines, wav, v.Options.Keywords)
		if err != nil {
			r.Err = err.Error()
			r.Comparison.Decision = DecisionInsufficient
			return r
		}
		r.Comparison = Compare(q.Text, hearings, v.Options.Compare)
		c := r.Comparison
		if attempt == maxWiden || c.Decision == DecisionInsufficient || (c.MissingStart == 0 && c.MissingEnd == 0) {
			r.Widened = attempt
			return r
		}
		if c.MissingStart > 0 {
			start -= edgeWiden(c.MissingStart)
			if start < 0 {
				start = 0
			}
		}
		end += edgeWiden(c.MissingEnd)
	}
}

// AudioInfo identifies the recording a report is about.
type AudioInfo struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// EngineInfo records an engine used in a report.
type EngineInfo struct {
	Name     string `json:"name"`
	Model    string `json:"model,omitempty"`
	Keywords bool   `json:"keywords"`
}

// Report is the record of verifying quotes against one recording.
type Report struct {
	// Tool names the module and version that produced the report.
	Tool      string       `json:"tool"`
	CreatedAt string       `json:"created_at"`
	Audio     AudioInfo    `json:"audio"`
	Index     string       `json:"index,omitempty"` // the index transcript's engine and model
	Engines   []EngineInfo `json:"engines"`
	Options   Options      `json:"options"`
	// MinEngines is the comparison's minimum number of engines.
	MinEngines int           `json:"min_engines"`
	Results    []QuoteResult `json:"results"`
	// Counts holds the number of results per decision.
	Counts map[Decision]int `json:"counts"`
}

// VerifyQuotes checks every quote and returns a report.
func (v Verifier) VerifyQuotes(ctx context.Context, audioPath string, index []TimedWord, indexName string, quotes []Quote) (*Report, error) {
	sum, err := fileSHA256(audioPath)
	if err != nil {
		return nil, err
	}
	rep := &Report{
		Tool:       Tool(),
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		Audio:      AudioInfo{Path: audioPath, SHA256: sum},
		Index:      indexName,
		Options:    v.Options,
		MinEngines: v.Options.Compare.MinEngines,
		Counts:     map[Decision]int{},
	}
	if rep.MinEngines == 0 {
		rep.MinEngines = 3
	}
	for _, e := range v.Engines {
		rep.Engines = append(rep.Engines, EngineInfo{
			Name: e.Name, Model: e.Config.Model,
			Keywords: !e.NoKeywords && len(v.Options.Keywords) > 0,
		})
	}
	for _, q := range quotes {
		r := v.VerifyQuote(ctx, audioPath, index, q)
		rep.Results = append(rep.Results, r)
		rep.Counts[r.Comparison.Decision]++
	}
	return rep, nil
}

// Tool returns this package's module path and version, for reports.
func Tool() string {
	const mod = "github.com/plexusone/omnivoice-core"
	if bi, ok := debug.ReadBuildInfo(); ok {
		if bi.Main.Path == mod {
			return mod + "/stt/verify@" + bi.Main.Version
		}
		for _, d := range bi.Deps {
			if d.Path == mod {
				return mod + "/stt/verify@" + d.Version
			}
		}
	}
	return mod + "/stt/verify"
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("verify: %w", err)
	}
	h := sha256.New()
	_, err = io.Copy(h, f)
	if cerr := f.Close(); err == nil && cerr != nil {
		err = cerr
	}
	if err != nil {
		return "", fmt.Errorf("verify: reading %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
