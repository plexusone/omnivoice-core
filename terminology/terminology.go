// Package terminology adapts the canonical term IR from
// github.com/plexusone/terminology-spec into the two text-transform
// primitives every TTS/subtitle consumer of omnivoice-core needs:
//
//   - Pronouncer: substitutes a term's spoken form into text immediately
//     before it's sent to a TTS provider, per BCP-47 locale.
//   - CaseCorrector: fixes the written/displayed form of a term (e.g. in
//     STT-derived subtitle text), independent of locale.
//
// This is the single shared implementation of that substitution engine —
// consumers (videoascode and others) should use this instead of
// reimplementing their own regex-based term matcher.
package terminology

import (
	"regexp"
	"sort"
	"strings"

	"github.com/plexusone/omnivoice-core/tts"
	"github.com/plexusone/terminology-spec/pkg/terminology"
)

// Term aliases terminology-spec's canonical Term type, so consumers of this
// package don't need a second direct import of terminology-spec just to
// name the type.
type Term = terminology.Term

// LoadDir re-exports terminology-spec's term loader so consumers don't need
// a direct import just to read terms/*.json.
func LoadDir(dir string) ([]terminology.Term, error) {
	return terminology.LoadDir(dir)
}

// Builtin re-exports terminology-spec's embedded generic-industry-term
// layer (~237 terms: AI/ML, companies, dev tools, languages, frameworks,
// cloud/infra, etc.) — available as a Go dependency, no filesystem access
// required.
func Builtin() ([]terminology.Term, error) {
	return terminology.Builtin()
}

// Curated re-exports terminology-spec's hand-authored, multi-org canonical
// terms (protocol/standard names with real pronunciation and translation
// data, e.g. "AAuth", "OAuth").
func Curated() ([]terminology.Term, error) {
	return terminology.Curated()
}

// All re-exports terminology-spec's full canonical term set (Builtin merged
// with Curated, Curated winning on ID collision).
func All() ([]terminology.Term, error) {
	return terminology.All()
}

// FilterByScope re-exports terminology-spec's scope filter: unscoped terms
// always pass, scoped terms pass only if they match one of the given
// scopes. Use this before projecting All()/Curated() so one org's
// abbreviations don't leak into another org's TTS input.
func FilterByScope(terms []terminology.Term, scopes ...string) []terminology.Term {
	return terminology.FilterByScope(terms, scopes...)
}

// PronunciationProfile re-exports terminology-spec's pronunciation
// projection: term (CanonicalForm + every Alias) -> BCP-47 locale -> spoken
// form, the exact shape videoascode's VideoConfig.Pronunciations consumes.
func PronunciationProfile(terms []terminology.Term) map[string]map[string]string {
	return terminology.ExportPronunciationProfile(terms)
}

// Merge re-exports terminology-spec's layering primitive: later layers
// override earlier ones by ID. Use this to compose Builtin() with a
// project's own terms (e.g. Builtin() as the base layer, an org's canonical
// terms/ directory layered on top).
func Merge(layers ...[]terminology.Term) []terminology.Term {
	return terminology.Merge(layers...)
}

// BuiltinCorrections returns the embedded generic-industry-term layer
// projected into a flat lowercase-form -> canonical-form correction map —
// the exact shape videoascode's Dictionary.Corrections already uses.
func BuiltinCorrections() (map[string]string, error) {
	terms, err := Builtin()
	if err != nil {
		return nil, err
	}
	return terminology.ExportCaseDictionary("builtin", "", terms).Corrections, nil
}

func termKeys(t terminology.Term) []string {
	return append([]string{t.CanonicalForm}, t.Aliases...)
}

func compilePattern(key string) (*regexp.Regexp, bool) {
	pattern, err := regexp.Compile(`(?i)\b` + regexp.QuoteMeta(key) + `\b`)
	return pattern, err == nil
}

// PronunciationsFor projects terms into the pronunciations a TTS request
// carries for one BCP-47 locale (tts.SynthesisConfig.Pronunciations): the
// term's IPA and respelling for that locale, for its CanonicalForm and
// each Alias. Terms with neither for the locale are left out.
func PronunciationsFor(terms []terminology.Term, locale string) []tts.Pronunciation {
	var ps []tts.Pronunciation
	for _, t := range terms {
		ipa, respelling := t.IPA[locale], t.Pronunciations[locale]
		if ipa == "" && respelling == "" {
			continue
		}
		for _, key := range termKeys(t) {
			if key != "" {
				ps = append(ps, tts.Pronunciation{Term: key, IPA: ipa, Respelling: respelling})
			}
		}
	}
	return ps
}

// Pronouncer applies per-locale TTS pronunciation substitutions from
// canonical Terms. A nil *Pronouncer is safe to call Apply on (no-op).
type Pronouncer struct {
	rules []pronunciationRule
}

type pronunciationRule struct {
	pattern   *regexp.Regexp
	wordCount int
	byLocale  map[string]string
}

// NewPronouncer builds a Pronouncer from canonical terms. Only terms with at
// least one Pronunciations entry produce a rule; both CanonicalForm and
// every Alias are matched.
func NewPronouncer(terms []terminology.Term) *Pronouncer {
	var rules []pronunciationRule
	for _, t := range terms {
		if len(t.Pronunciations) == 0 {
			continue
		}
		for _, key := range termKeys(t) {
			if key == "" {
				continue
			}
			pattern, ok := compilePattern(key)
			if !ok {
				continue
			}
			rules = append(rules, pronunciationRule{
				pattern:   pattern,
				wordCount: len(strings.Fields(key)),
				byLocale:  t.Pronunciations,
			})
		}
	}
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].wordCount > rules[j].wordCount })
	return &Pronouncer{rules: rules}
}

// Apply substitutes pronunciation for the given BCP-47 locale into text.
// Terms with no entry for locale are left unmodified. Safe to call on a nil
// receiver (returns text unchanged).
func (p *Pronouncer) Apply(text, locale string) string {
	if p == nil {
		return text
	}
	result := text
	for _, r := range p.rules {
		if spoken, ok := r.byLocale[locale]; ok {
			result = r.pattern.ReplaceAllString(result, spoken)
		}
	}
	return result
}

// CaseCorrector fixes the written form of terms (e.g. "aauth" -> "AAuth")
// independent of locale — for subtitle/display text, not TTS input. A nil
// *CaseCorrector is safe to call Correct on (no-op).
type CaseCorrector struct {
	rules []caseRule
}

type caseRule struct {
	pattern       *regexp.Regexp
	wordCount     int
	canonicalForm string
}

// NewCaseCorrector builds a CaseCorrector from canonical terms'
// CanonicalForm/Aliases. Terms with an empty CanonicalForm are skipped.
func NewCaseCorrector(terms []terminology.Term) *CaseCorrector {
	var rules []caseRule
	for _, t := range terms {
		if t.CanonicalForm == "" {
			continue
		}
		for _, key := range termKeys(t) {
			if key == "" {
				continue
			}
			pattern, ok := compilePattern(key)
			if !ok {
				continue
			}
			rules = append(rules, caseRule{
				pattern:       pattern,
				wordCount:     len(strings.Fields(key)),
				canonicalForm: t.CanonicalForm,
			})
		}
	}
	sort.SliceStable(rules, func(i, j int) bool { return rules[i].wordCount > rules[j].wordCount })
	return &CaseCorrector{rules: rules}
}

// Correct applies all case corrections to text. Safe to call on a nil
// receiver (returns text unchanged).
func (cc *CaseCorrector) Correct(text string) string {
	if cc == nil {
		return text
	}
	result := text
	for _, r := range cc.rules {
		result = r.pattern.ReplaceAllString(result, r.canonicalForm)
	}
	return result
}
