package verify

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/plexusone/omnivoice-core/stt"
)

// Engine is one speech-to-text engine used to hear a clip.
type Engine struct {
	// Name identifies the engine in results, such as "deepgram".
	Name     string
	Provider stt.Provider
	// Config is passed to the provider. Encoding, sample rate and channels
	// are set for WAV clips when empty.
	Config stt.TranscriptionConfig
	// NoKeywords keeps key words away from this engine, so key words can't
	// steer every engine toward the same hearing.
	NoKeywords bool
}

// ErrAllEnginesUseKeywords is returned when key words are given and no
// engine is set to run without them.
var ErrAllEnginesUseKeywords = errors.New("verify: at least one engine must run without key words")

// TranscribeAll has every engine transcribe the same WAV clip at once.
//
// keywords are added to each engine's configuration except those marked
// NoKeywords. Results are sorted by engine name. A failure is recorded in the
// hearing's Err rather than returned.
func TranscribeAll(ctx context.Context, engines []Engine, wav []byte, keywords []string) ([]Hearing, error) {
	if len(keywords) > 0 {
		free := false
		for _, e := range engines {
			if e.NoKeywords {
				free = true
			}
		}
		if !free {
			return nil, ErrAllEnginesUseKeywords
		}
	}
	out := make([]Hearing, len(engines))
	var wg sync.WaitGroup
	for i, e := range engines {
		wg.Add(1)
		go func(i int, e Engine) {
			defer wg.Done()
			cfg := e.Config
			if cfg.Encoding == "" {
				cfg.Encoding = "wav"
			}
			if cfg.SampleRate == 0 {
				cfg.SampleRate = 16000
			}
			if cfg.Channels == 0 {
				cfg.Channels = 1
			}
			if !e.NoKeywords && len(keywords) > 0 {
				cfg.Keywords = append(append([]string(nil), cfg.Keywords...), keywords...)
			}
			h := Hearing{Engine: e.Name, Model: cfg.Model}
			res, err := e.Provider.Transcribe(ctx, wav, cfg)
			switch {
			case err != nil:
				h.Err = err.Error()
			case res == nil:
				h.Err = "no result"
			default:
				h.Text = res.Text
			}
			out[i] = h
		}(i, e)
	}
	wg.Wait()
	sort.SliceStable(out, func(i, j int) bool { return out[i].Engine < out[j].Engine })
	return out, nil
}
