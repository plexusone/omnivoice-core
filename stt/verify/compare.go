package verify

import (
	"fmt"
	"sort"
	"strings"
)

// Hearing is what one engine heard in a clip.
type Hearing struct {
	// Engine names the provider, such as "deepgram".
	Engine string `json:"engine"`
	// Model is the provider's model, if known.
	Model string `json:"model,omitempty"`
	// Text is the transcription. It is empty when Err is set.
	Text string `json:"text"`
	// Err records why the engine failed. A failed hearing is not compared.
	Err string `json:"error,omitempty"`
}

// Decision is the outcome of comparing hearings with expected wording.
type Decision string

// Decisions.
const (
	// DecisionAgreed: every engine heard the expected wording, apart from
	// function words on which most engines agree with it.
	DecisionAgreed Decision = "agreed"
	// DecisionCorrected: the engines agree with each other, but not with the
	// expected wording. Comparison.Corrected holds what they heard.
	DecisionCorrected Decision = "corrected"
	// DecisionDisputed: the engines disagree with each other on a
	// substantive word.
	DecisionDisputed Decision = "disputed"
	// DecisionNotFound: the expected wording could not be located.
	DecisionNotFound Decision = "notFound"
	// DecisionInsufficient: too few engines succeeded to decide.
	DecisionInsufficient Decision = "insufficient"
)

// CompareOptions configures Compare.
type CompareOptions struct {
	// MinEngines is the fewest successful hearings needed to decide.
	// Zero means 3.
	MinEngines int
	// Normalize configures normalization of the expected text and hearings.
	Normalize NormalizeOptions
	// Aliases maps an expected token sequence to other sequences that also
	// satisfy it, such as "utm" heard as "u t m". Keys and values are
	// normalized before use.
	Aliases map[string][]string
}

// Difference is one place where at least one engine heard something other
// than the expected wording.
type Difference struct {
	// Position is the index of the expected token. For an insertion, it is
	// the index of the expected token the extra words come before (the
	// length of the expected tokens for words after the last one).
	Position int `json:"position"`
	// Expected is the expected token, empty for an insertion.
	Expected string `json:"expected"`
	// Insertion is true when engines heard extra words at this position.
	Insertion bool `json:"insertion,omitempty"`
	// Kind is the kind of the expected token, or of the first heard token for
	// an insertion.
	Kind Kind `json:"kind"`
	// Substantive is true when the expected token or any heard token here is
	// substantive.
	Substantive bool `json:"substantive"`
	// Heard maps each engine to what it heard here; empty means nothing.
	Heard map[string]string `json:"heard"`
	// Resolved is the value the comparison settled on: the expected value,
	// or the alternative the engines agree on. It is unset when disputed.
	Resolved *string `json:"resolved,omitempty"`
	// Edge is true for an expected token at the start or end of the quote
	// that no engine heard. It may lie just outside the clip, so it is never
	// treated as a correction.
	Edge bool `json:"edge,omitempty"`
}

// EngineResult summarizes one engine's hearing against the expected wording.
type EngineResult struct {
	Engine string `json:"engine"`
	Model  string `json:"model,omitempty"`
	Text   string `json:"text"`
	Err    string `json:"error,omitempty"`
	// Distance is the word-level edit distance from the expected wording.
	Distance int `json:"distance"`
	// Matches is true when the engine heard every expected token.
	Matches bool `json:"matches"`
}

// Comparison is the result of Compare.
type Comparison struct {
	Decision Decision `json:"decision"`
	Expected string   `json:"expected"`
	// Corrected is the wording the engines heard, when the decision is
	// corrected. Unchanged words keep their written form.
	Corrected string `json:"corrected,omitempty"`
	// CorrectedSubstantive is true when a correction changes a substantive
	// word, so the claim itself may need fixing.
	CorrectedSubstantive bool `json:"corrected_substantive,omitempty"`
	// MissingStart and MissingEnd count expected tokens at the start and end
	// that no engine heard. A wider clip may settle them.
	MissingStart int            `json:"missing_start,omitempty"`
	MissingEnd   int            `json:"missing_end,omitempty"`
	Engines      []EngineResult `json:"engines"`
	Differences  []Difference   `json:"differences,omitempty"`
}

// Compare compares what engines heard with the expected wording.
//
// Each hearing is aligned against the expected tokens, ignoring words heard
// before and after them, since a clip is padded. At each expected token, and
// between tokens, it collects what each engine heard. Then:
//
//   - fewer than MinEngines successful hearings: insufficient;
//   - every position matches, or a non-substantive position has a majority
//     for the expected token: agreed;
//   - every differing position has one value all engines share, or a
//     majority for a non-substantive one: corrected;
//   - otherwise, or when no engine heard the first or last words: disputed.
func Compare(expected string, hearings []Hearing, opts CompareOptions) Comparison {
	minEngines := opts.MinEngines
	if minEngines == 0 {
		minEngines = 3
	}
	exp := Normalize(expected, opts.Normalize)
	c := Comparison{Expected: expected, Engines: []EngineResult{}}

	type row struct {
		engine string
		sub    []string // per expected token: what was heard ("" = nothing)
		ins    []string // per gap before token i (len = len(exp)+1)
		insK   []Kind
		subK   []Kind
	}
	var rows []row
	hs := append([]Hearing(nil), hearings...)
	sort.SliceStable(hs, func(i, j int) bool { return hs[i].Engine < hs[j].Engine })
	// Each hearing is one vote, keyed by engine name: number repeated names
	// ("openai", "openai#2") so none overwrites another.
	seen := map[string]int{}
	for i := range hs {
		seen[hs[i].Engine]++
		if n := seen[hs[i].Engine]; n > 1 {
			hs[i].Engine = fmt.Sprintf("%s#%d", hs[i].Engine, n)
		}
	}
	for _, h := range hs {
		er := EngineResult{Engine: h.Engine, Model: h.Model, Text: h.Text, Err: h.Err}
		if h.Err != "" {
			c.Engines = append(c.Engines, er)
			continue
		}
		hyp := joinSplitWords(applyAliases(Normalize(h.Text, opts.Normalize), opts), exp)
		pairs, dist := AlignWithin(exp, hyp)
		r := row{
			engine: h.Engine,
			sub:    make([]string, len(exp)),
			subK:   make([]Kind, len(exp)),
			ins:    make([]string, len(exp)+1),
			insK:   make([]Kind, len(exp)+1),
		}
		next := 0 // the expected token the next insertion comes before
		gapWords := make([][]string, len(exp)+1)
		gapKinds := make([][]Kind, len(exp)+1)
		for _, p := range pairs {
			switch p.Op {
			case OpMatch, OpSubstitute:
				r.sub[p.Ref] = hyp[p.Hyp].Text
				r.subK[p.Ref] = hyp[p.Hyp].Kind
				next = p.Ref + 1
			case OpDelete:
				r.sub[p.Ref] = ""
				next = p.Ref + 1
			case OpInsert:
				// Words heard across an elision are expected; ignore them.
				if next > 0 && exp[next-1].GapAfter {
					continue
				}
				// A word said twice in a row ("to, to") is a stutter.
				if (p.Hyp > 0 && hyp[p.Hyp].Text == hyp[p.Hyp-1].Text) ||
					(p.Hyp+1 < len(hyp) && hyp[p.Hyp].Text == hyp[p.Hyp+1].Text) {
					continue
				}
				gapWords[next] = append(gapWords[next], hyp[p.Hyp].Text)
				gapKinds[next] = append(gapKinds[next], hyp[p.Hyp].Kind)
			}
		}
		for g := range gapWords {
			r.ins[g] = strings.Join(gapWords[g], " ")
			r.insK[g] = strongestKind(gapKinds[g])
		}
		er.Distance = dist
		er.Matches = dist == 0
		c.Engines = append(c.Engines, er)
		rows = append(rows, r)
	}
	if len(rows) < minEngines {
		c.Decision = DecisionInsufficient
		return c
	}
	if len(exp) == 0 {
		c.Decision = DecisionNotFound
		return c
	}

	// Expected tokens at either edge that no engine heard.
	allMissed := func(i int) bool {
		for _, r := range rows {
			if r.sub[i] != "" {
				return false
			}
		}
		return true
	}
	for i := 0; i < len(exp) && allMissed(i); i++ {
		c.MissingStart++
	}
	for i := len(exp) - 1; i >= c.MissingStart && allMissed(i); i-- {
		c.MissingEnd++
	}
	isEdge := func(i int) bool { return i < c.MissingStart || i >= len(exp)-c.MissingEnd }

	disputed := c.MissingStart > 0 || c.MissingEnd > 0
	corrected := false
	replacement := make([]*string, len(exp))
	insertions := make([]string, len(exp)+1)

	resolve := func(d *Difference, expectedVal string) {
		values := map[string]int{}
		for _, r := range rows {
			values[d.Heard[r.engine]]++
		}
		n := len(rows)
		switch {
		case len(values) == 1:
			for v := range values {
				d.Resolved = ptr(v)
			}
		case !d.Substantive && values[expectedVal]*2 > n:
			d.Resolved = ptr(expectedVal)
		case !d.Substantive:
			if v, ok := majority(values, n); ok {
				d.Resolved = ptr(v)
			} else {
				disputed = true // no majority, even on a function word
			}
		default:
			disputed = true
		}
	}

	for i := 0; i <= len(exp); i++ {
		// Extra words before expected token i.
		heard := map[string]string{}
		anyIns := false
		sub := false
		var kind Kind
		for _, r := range rows {
			heard[r.engine] = r.ins[i]
			if r.ins[i] != "" {
				anyIns = true
				if r.insK[i] != KindFunction {
					sub = true
				}
				if kind == "" {
					kind = r.insK[i]
				}
			}
		}
		if anyIns {
			d := Difference{Position: i, Insertion: true, Kind: kind, Substantive: sub, Heard: heard}
			resolve(&d, "")
			if d.Resolved != nil && *d.Resolved != "" {
				corrected = true
				insertions[i] = *d.Resolved
			}
			c.Differences = append(c.Differences, d)
		}
		if i == len(exp) {
			break
		}
		// Expected token i.
		heard = map[string]string{}
		differs := false
		sub = exp[i].Substantive()
		for _, r := range rows {
			heard[r.engine] = r.sub[i]
			if r.sub[i] != exp[i].Text {
				differs = true
				if r.sub[i] != "" && r.subK[i] != KindFunction {
					sub = true
				}
			}
		}
		if !differs {
			continue
		}
		d := Difference{Position: i, Expected: exp[i].Text, Kind: exp[i].Kind, Substantive: sub, Heard: heard}
		if isEdge(i) {
			d.Edge = true
			c.Differences = append(c.Differences, d)
			continue
		}
		resolve(&d, exp[i].Text)
		if d.Resolved != nil && *d.Resolved != exp[i].Text {
			corrected = true
			replacement[i] = d.Resolved
		}
		c.Differences = append(c.Differences, d)
	}

	switch {
	case disputed:
		c.Decision = DecisionDisputed
	case corrected:
		c.Decision = DecisionCorrected
		c.Corrected, c.CorrectedSubstantive = rebuild(exp, replacement, insertions, c.Differences)
	default:
		c.Decision = DecisionAgreed
	}
	return c
}

// rebuild writes the corrected wording: unchanged words keep their written
// form, replaced and inserted words take the engines' normalized form.
func rebuild(exp []Token, repl []*string, ins []string, diffs []Difference) (string, bool) {
	var out []string
	for i := 0; i < len(exp); {
		// The tokens of one written word ("don't" is "do" and "not").
		j := i + 1
		for j < len(exp) && exp[j].Word == exp[i].Word {
			j++
		}
		changed := false
		for k := i; k < j; k++ {
			if repl[k] != nil || (k > i && ins[k] != "") {
				changed = true
			}
		}
		if ins[i] != "" {
			out = append(out, ins[i])
		}
		if !changed {
			out = append(out, exp[i].Orig)
			i = j
			continue
		}
		// Part of the word changed: write its tokens as resolved.
		for k := i; k < j; k++ {
			if k > i && ins[k] != "" {
				out = append(out, ins[k])
			}
			switch {
			case repl[k] == nil:
				out = append(out, exp[k].Text)
			case *repl[k] != "":
				out = append(out, *repl[k])
			}
		}
		i = j
	}
	if ins[len(exp)] != "" {
		out = append(out, ins[len(exp)])
	}
	substantive := false
	for _, d := range diffs {
		if d.Substantive && d.Resolved != nil {
			if (d.Insertion && *d.Resolved != "") || (!d.Insertion && *d.Resolved != d.Expected) {
				substantive = true
			}
		}
	}
	return strings.Join(out, " "), substantive
}

// applyAliases rewrites alias sequences in a hearing into the expected form,
// so "u t m" is compared as "utm".
func applyAliases(hyp []Token, opts CompareOptions) []Token {
	if len(opts.Aliases) == 0 {
		return hyp
	}
	keys := make([]string, 0, len(opts.Aliases))
	for k := range opts.Aliases {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		target := Normalize(k, opts.Normalize)
		if len(target) == 0 {
			continue
		}
		for _, alias := range opts.Aliases[k] {
			src := Normalize(alias, opts.Normalize)
			hyp = replaceSeq(hyp, src, target)
		}
	}
	return hyp
}

func replaceSeq(toks, src, target []Token) []Token {
	if len(src) == 0 {
		return toks
	}
	var out []Token
	for i := 0; i < len(toks); {
		if i+len(src) <= len(toks) && seqEqual(toks[i:i+len(src)], src) {
			for _, t := range target {
				t.Word = toks[i].Word
				out = append(out, t)
			}
			i += len(src)
			continue
		}
		out = append(out, toks[i])
		i++
	}
	return out
}

// joinSplitWords reconciles word boundaries between what was heard and what
// was expected:
//
//   - Two heard tokens are joined when together they spell an expected
//     token, so "work flow" is compared as "workflow" and "a synchronous" as
//     "asynchronous". They are not joined when both are expected words on
//     their own.
//   - A heard token that spells two consecutive expected tokens is split, so
//     "speedups" is compared as "speed ups". It stays whole when it is an
//     expected word itself, or when it spells two different expected pairs
//     ("anice" for "an ice" and "a nice").
//   - A word broken off as the speaker restarted ("ha-" before "heart") is
//     dropped unless every token it yields is an expected word: "of-" can
//     stand for a quoted "of", but "kinda-" is dropped whole. A broken-off
//     word is never joined to another.
func joinSplitWords(hyp, exp []Token) []Token {
	want := map[string]Kind{}
	for _, t := range exp {
		want[t.Text] = t.Kind
	}
	split := map[string][2]Token{}
	ambiguous := map[string]bool{}
	for k := 0; k+1 < len(exp); k++ {
		key := exp[k].Text + exp[k+1].Text
		if prev, ok := split[key]; ok && (prev[0].Text != exp[k].Text || prev[1].Text != exp[k+1].Text) {
			ambiguous[key] = true
		}
		split[key] = [2]Token{exp[k], exp[k+1]}
	}
	// A broken-off word is kept only if all its tokens are expected words.
	keepCut := map[int]bool{}
	for _, t := range hyp {
		if !t.cut {
			continue
		}
		if _, seen := keepCut[t.Word]; !seen {
			keepCut[t.Word] = true
		}
		if want[t.Text] == "" {
			keepCut[t.Word] = false
		}
	}
	var kept []Token
	for _, t := range hyp {
		if t.cut && !keepCut[t.Word] {
			continue
		}
		kept = append(kept, t)
	}
	hyp = kept
	var out []Token
	for i := 0; i < len(hyp); i++ {
		if pair, ok := split[hyp[i].Text]; ok && want[hyp[i].Text] == "" && !ambiguous[hyp[i].Text] {
			for _, e := range pair {
				t := hyp[i]
				t.Text, t.Kind = e.Text, e.Kind
				out = append(out, t)
			}
			continue
		}
		if i+1 < len(hyp) {
			joined := hyp[i].Text + hyp[i+1].Text
			bothExpected := want[hyp[i].Text] != "" && want[hyp[i+1].Text] != ""
			if k, ok := want[joined]; ok && !bothExpected && !hyp[i].cut && !hyp[i+1].cut {
				t := hyp[i]
				t.Text, t.Kind = joined, k
				t.end = hyp[i+1].end
				out = append(out, t)
				i++
				continue
			}
		}
		out = append(out, hyp[i])
	}
	return out
}

func seqEqual(a, b []Token) bool {
	for i := range b {
		if a[i].Text != b[i].Text {
			return false
		}
	}
	return true
}

func strongestKind(ks []Kind) Kind {
	if len(ks) == 0 {
		return ""
	}
	for _, k := range ks {
		if k != KindFunction {
			return k
		}
	}
	return KindFunction
}

func majority(values map[string]int, n int) (string, bool) {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if values[k]*2 > n {
			return k, true
		}
	}
	return "", false
}

func ptr(s string) *string { return &s }
